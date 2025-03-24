// Package consul provides the consul registry for go-orb.
package consul

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	"github.com/go-orb/plugins/registry/regutil/cache"
)

const nodeIDDelimiter = "@"

func nodeID(s registry.ServiceNode) string {
	return strings.Join([]string{
		s.Namespace,
		s.Region,
		s.Name,
		s.Version,
		s.Scheme,
		s.Address,
	}, nodeIDDelimiter)
}

func idToNode(id string) (registry.ServiceNode, error) {
	parts := strings.Split(id, nodeIDDelimiter)
	if len(parts) != 6 {
		return registry.ServiceNode{}, errors.New("invalid id format")
	}

	return registry.ServiceNode{
		Namespace: parts[0],
		Region:    parts[1],
		Name:      parts[2],
		Version:   parts[3],
		Scheme:    parts[4],
		Address:   parts[5],
		Metadata:  make(map[string]string),
	}, nil
}

func splitID(id string) (string, string, string) {
	splits := strings.Split(id, nodeIDDelimiter)

	return splits[0], splits[1], splits[2]
}

func idToPrefix(id string) string {
	return strings.Join(strings.Split(id, nodeIDDelimiter)[:3], nodeIDDelimiter) + nodeIDDelimiter
}

// This is here to make sure RegistryConsul implements registry.Registry.
var _ registry.Registry = (*RegistryConsul)(nil)

// RegistryConsul is the consul registry for go-orb.
type RegistryConsul struct {
	Address []string

	config Config
	logger log.Logger

	client       *consul.Client
	consulConfig *consul.Config

	queryOptions *consul.QueryOptions

	// cache is used to cache registry operations.
	cache *cache.Cache
}

func getDeregisterTTL(t time.Duration) time.Duration {
	// splay slightly for the watcher?
	splay := time.Second * 5
	deregTTL := t + splay

	// consul has a minimum timeout on deregistration of 1 minute.
	if t < time.Minute {
		deregTTL = time.Minute + splay
	}

	return deregTTL
}

func newTransport(config *tls.Config) *http.Transport {
	if config == nil {
		config = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		}
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     config,
	}
	runtime.SetFinalizer(&t, func(tr **http.Transport) {
		(*tr).CloseIdleConnections()
	})

	return t
}

// Deregister deregisters a service within the registry.
func (c *RegistryConsul) Deregister(_ context.Context, serviceNode registry.ServiceNode) error {
	return c.Client().Agent().ServiceDeregister(nodeID(serviceNode))
}

// Register registers a service within the registry.
func (c *RegistryConsul) Register(_ context.Context, serviceNode registry.ServiceNode) error {
	var (
		regTCPCheck bool
		regInterval time.Duration
	)

	if c.config.TCPCheck > 0 {
		regTCPCheck = true
		regInterval = c.config.TCPCheck
	}

	var check *consul.AgentServiceCheck

	if regTCPCheck {
		deregTTL := getDeregisterTTL(regInterval)

		check = &consul.AgentServiceCheck{
			TCP:                            serviceNode.Address,
			Interval:                       fmt.Sprintf("%v", regInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	} else if serviceNode.TTL > time.Duration(0) {
		// if the TTL is greater than 0 create an associated check
		deregTTL := getDeregisterTTL(serviceNode.TTL)

		check = &consul.AgentServiceCheck{
			TTL:                            fmt.Sprintf("%v", serviceNode.TTL),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	}

	host, pt, err := net.SplitHostPort(serviceNode.Address)
	if err != nil {
		return err
	}

	if host == "" {
		host = serviceNode.Address
	}

	port, err := strconv.Atoi(pt)
	if err != nil {
		return err
	}

	metadata := map[string]string{}
	// Add service metadata to the service
	for k, v := range serviceNode.Metadata {
		metadata[metaPrefix+k] = v
	}

	// register the service
	asr := &consul.AgentServiceRegistration{
		ID:      nodeID(serviceNode),
		Name:    serviceNode.Name,
		Port:    port,
		Address: host,
		Meta:    metadata,
		Check:   check,
	}

	// Specify consul connect
	if c.config.Connect {
		asr.Connect = &consul.AgentServiceConnect{
			Native: true,
		}
	}

	if err := c.Client().Agent().ServiceRegister(asr); err != nil {
		return err
	}

	// if the TTL is 0 we don't mess with the checks
	if serviceNode.TTL == time.Duration(0) {
		return nil
	}

	// pass the healthcheck
	if err := c.Client().Agent().PassTTL("service:"+asr.ID, ""); err != nil {
		return err
	}

	return nil
}

// GetService returns a service from the registry.
//
//nolint:gocyclo
func (c *RegistryConsul) GetService(ctx context.Context, namespace, region, name string, schemes []string) ([]registry.ServiceNode, error) {
	if c.config.Cache {
		return c.cache.GetService(ctx, namespace, region, name, schemes)
	}

	var (
		rsp []*consul.ServiceEntry
		err error
	)

	// if we're connect enabled only get connect services
	if c.config.Connect {
		rsp, _, err = c.Client().Health().Connect(name, "", false, c.queryOptions)
	} else {
		rsp, _, err = c.Client().Health().Service(name, "", false, c.queryOptions)
	}

	if err != nil {
		return nil, err
	}

	if len(rsp) == 0 {
		return []registry.ServiceNode{}, registry.ErrNotFound
	}

	services := []registry.ServiceNode{}

	for _, node := range rsp {
		if node.Service.Service != name {
			continue
		}

		serviceNode, err := idToNode(node.Service.ID)
		if err != nil {
			continue
		}

		svcMeta := map[string]string{}

		for k, v := range node.Service.Meta {
			if strings.HasPrefix(k, metaPrefix) {
				svcMeta[strings.TrimPrefix(k, metaPrefix)] = v
			}
		}

		serviceNode.Metadata = svcMeta

		if serviceNode.Namespace != namespace {
			continue
		}

		if serviceNode.Region != region {
			continue
		}

		if len(schemes) > 0 && !slices.Contains(schemes, serviceNode.Scheme) {
			continue
		}

		var del bool

		for _, check := range node.Checks {
			// delete the node if the status is critical
			if check.Status == "critical" {
				del = true
				break
			}
		}

		// if delete then skip the node
		if del {
			continue
		}

		services = append(services, serviceNode)
	}

	if len(services) < 1 {
		return nil, registry.ErrNotFound
	}

	return services, nil
}

// ListServices lists services within the registry.
func (c *RegistryConsul) ListServices(ctx context.Context, namespace, region string, schemes []string) ([]registry.ServiceNode, error) {
	if c.config.Cache {
		return c.cache.ListServices(ctx, namespace, region, schemes)
	}

	rsp, _, err := c.Client().Catalog().Services(c.queryOptions)
	if err != nil {
		return nil, err
	}

	var nodes []registry.ServiceNode

	for service := range rsp {
		srvNodes, err := c.GetService(ctx, namespace, region, service, schemes)
		if err != nil {
			if errors.Is(err, registry.ErrNotFound) {
				continue
			}

			return nil, err
		}

		nodes = append(nodes, srvNodes...)
	}

	return nodes, nil
}

// Watch returns a Watcher which you can watch on.
func (c *RegistryConsul) Watch(_ context.Context, opts ...registry.WatchOption) (registry.Watcher, error) {
	return newConsulWatcher(c, opts...)
}

// Client returns the consul client.
func (c *RegistryConsul) Client() *consul.Client {
	if c.client != nil {
		return c.client
	}

	for _, addr := range c.Address {
		// set the address
		c.consulConfig.Address = addr

		// create a new client
		tmpClient, err := consul.NewClient(c.consulConfig)
		if err != nil {
			continue
		}

		// test the client
		_, err = tmpClient.Agent().Host()
		if err != nil {
			continue
		}

		// set the client
		c.client = tmpClient

		return c.client
	}

	// set the default
	c.client, _ = consul.NewClient(c.consulConfig) //nolint:errcheck

	// return the client
	return c.client
}

// Provide creates a new Consul registry.
func Provide(
	datas map[string]any,
	_ *types.Components,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, registry.DefaultConfigSection, datas, &cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return registry.Type{}, fmt.Errorf("parse config: %w", err)
	}

	// Return the new registry.
	reg := New(cfg, logger)

	return registry.Type{Registry: reg}, nil
}

// New creates a new consul registry.
func New(cfg Config, logger log.Logger) *RegistryConsul {
	cRegistry := &RegistryConsul{
		config: cfg,
		logger: logger,
		queryOptions: &consul.QueryOptions{
			AllowStale: true,
		},
	}

	// use default non pooled config
	config := consul.DefaultNonPooledConfig()

	// Use the consul config passed in the options, if available
	if cfg.ConsulConfig != nil {
		config = cfg.ConsulConfig
	}

	// Use the consul query options passed in the options, if available
	if cfg.QueryOptions != nil {
		cRegistry.queryOptions = cfg.QueryOptions
	}

	cRegistry.queryOptions.AllowStale = cfg.AllowStale

	// check if there are any addrs
	var addrs []string

	// iterate the options addresses
	for _, address := range cfg.Addresses {
		// check we have a port
		addr, port, err := net.SplitHostPort(address)

		addrError := &net.AddrError{}

		switch {
		case errors.As(err, &addrError):
			port = "8500"
			addrs = append(addrs, net.JoinHostPort(addr, port))
		default:
			addrs = append(addrs, net.JoinHostPort(addr, port))
		}
	}

	// set the addrs
	if len(addrs) > 0 {
		cRegistry.Address = addrs
		config.Address = cRegistry.Address[0]
	}

	if config.HttpClient == nil {
		config.HttpClient = new(http.Client)
	}

	// requires secure connection?
	if cfg.Secure || cfg.TLSConfig != nil {
		config.Scheme = "https"
		// We're going to support InsecureSkipVerify
		config.HttpClient.Transport = newTransport(cfg.TLSConfig)
	}

	// set timeout
	if cfg.Timeout > 0 {
		config.HttpClient.Timeout = time.Duration(cfg.Timeout) * time.Second
	}

	// set the config
	cRegistry.consulConfig = config

	// remove the client
	cRegistry.client = nil

	// Initialize the cache with a reference to this registry
	cRegistry.cache = cache.New(cache.Config{}, logger, cRegistry)

	return cRegistry
}

// Start the registry.
func (c *RegistryConsul) Start(ctx context.Context) error {
	// setup the client
	c.Client()

	// Start the cache - this will populate it and begin watching for changes
	if c.config.Cache {
		return c.cache.Start(ctx)
	}

	return nil
}

// Stop the registry.
func (c *RegistryConsul) Stop(ctx context.Context) error {
	// Stop the cache first
	if c.config.Cache {
		if err := c.cache.Stop(ctx); err != nil {
			c.logger.Warn("Error stopping cache", "error", err)
		}
	}

	// Then remove the client
	c.client = nil

	return nil
}

// String returns the plugin name.
func (c *RegistryConsul) String() string {
	return Name
}

// Type returns the component type.
func (c *RegistryConsul) Type() string {
	return registry.ComponentType
}
