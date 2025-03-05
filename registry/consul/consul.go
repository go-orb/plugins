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
	"strconv"
	"sync"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	maddr "github.com/go-orb/go-orb/util/addr"
	"github.com/google/uuid"
	consul "github.com/hashicorp/consul/api"
	hash "github.com/mitchellh/hashstructure"
)

// This is here to make sure RegistryConsul implements registry.Registry.
var _ registry.Registry = (*RegistryConsul)(nil)

// RegistryConsul is the consul registry for go-orb.
type RegistryConsul struct {
	Address        []string
	serviceName    string
	serviceVersion string

	config Config
	logger log.Logger

	id string

	client       *consul.Client
	consulConfig *consul.Config

	queryOptions *consul.QueryOptions

	sync.Mutex
	register map[string]uint64
	// lastChecked tracks when a node was last checked as existing in Consul
	lastChecked map[string]time.Time
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

// ServiceName returns the configured name of this service.
func (c *RegistryConsul) ServiceName() string {
	return c.serviceName
}

// ServiceVersion returns the configured version of this service.
func (c *RegistryConsul) ServiceVersion() string {
	return c.serviceVersion
}

// NodeID returns the ID of this service node in the registry.
func (c *RegistryConsul) NodeID() string {
	if c.id != "" {
		return c.id
	}

	c.id = c.serviceName + "-" + uuid.New().String()

	return c.id
}

// Deregister deregisters a service within the registry.
func (c *RegistryConsul) Deregister(s *registry.Service, _ ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("require at least one node")
	}

	// delete our hash and time check of the service
	c.Lock()
	delete(c.register, s.Name)
	delete(c.lastChecked, s.Name)
	c.Unlock()

	node := s.Nodes[0]

	return c.Client().Agent().ServiceDeregister(node.ID)
}

// Register registers a service within the registry.
// TODO(jochumdev): work on the nolints.
//
//nolint:gocognit,gocyclo,cyclop,funlen
func (c *RegistryConsul) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	if len(service.Nodes) == 0 {
		return errors.New("require at least one node")
	}

	var (
		regTCPCheck bool
		regInterval time.Duration
		options     registry.RegisterOptions
	)

	for _, o := range opts {
		o(&options)
	}

	if c.config.TCPCheck > 0 {
		regTCPCheck = true
		regInterval = c.config.TCPCheck
	}

	// create hash of service; uint64
	serviceHash, err := hash.Hash(service, nil)
	if err != nil {
		return err
	}

	// use first node
	node := service.Nodes[0]

	// get existing hash and last checked time
	c.Lock()
	v, ok := c.register[service.Name]
	lastChecked := c.lastChecked[service.Name]
	c.Unlock()

	// if it's already registered and matches then just pass the check
	if ok && v == serviceHash && options.TTL == time.Duration(0) { //nolint:nestif
		// ensure that our service hasn't been deregistered by Consul
		if time.Since(lastChecked) <= getDeregisterTTL(regInterval) {
			return nil
		}

		services, _, err := c.Client().Health().Checks(service.Name, c.queryOptions)
		if err == nil {
			for _, v := range services {
				if v.ServiceID == node.ID {
					return nil
				}
			}
		}
	} else if ok && v == serviceHash {
		// if the err is nil we're all good, bail out
		// if not, we don't know what the state is, so full re-register
		if err := c.Client().Agent().PassTTL("service:"+node.ID, ""); err == nil {
			return nil
		}
	}

	// encode the tags
	tags := encodeEndpoints(service.Endpoints)
	tags = append(tags, encodeVersion(service.Version)...)

	var check *consul.AgentServiceCheck

	if regTCPCheck {
		deregTTL := getDeregisterTTL(regInterval)

		check = &consul.AgentServiceCheck{
			TCP:                            node.Address,
			Interval:                       fmt.Sprintf("%v", regInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	} else if options.TTL > time.Duration(0) {
		// if the TTL is greater than 0 create an associated check
		deregTTL := getDeregisterTTL(options.TTL)

		check = &consul.AgentServiceCheck{
			TTL:                            fmt.Sprintf("%v", options.TTL),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	}

	host, pt, err := net.SplitHostPort(node.Address)
	if err != nil {
		return err
	}

	if host == "" {
		host = node.Address
	}

	port, err := strconv.Atoi(pt)
	if err != nil {
		return err
	}

	if node.Metadata == nil {
		node.Metadata = make(map[string]string)
	}

	// register the service
	asr := &consul.AgentServiceRegistration{
		ID:      node.ID,
		Name:    service.Name,
		Tags:    tags,
		Port:    port,
		Address: host,
		Meta:    node.Metadata,
		Check:   check,
	}

	// Add the transport scheme to metadata if required
	if _, ok := asr.Meta[metaTransportKey]; !ok {
		asr.Meta[metaTransportKey] = node.Transport
	}

	// Specify consul connect
	if c.config.Connect {
		asr.Connect = &consul.AgentServiceConnect{
			Native: true,
		}
	}

	c.logger.Trace("Registering a service", "name", asr.Name, "address", asr.Address, "port", asr.Port, "transport", node.Transport)

	if err := c.Client().Agent().ServiceRegister(asr); err != nil {
		return err
	}

	// save our hash and time check of the service
	c.Lock()
	c.register[service.Name] = serviceHash
	c.lastChecked[service.Name] = time.Now()
	c.Unlock()

	// if the TTL is 0 we don't mess with the checks
	if options.TTL == time.Duration(0) {
		return nil
	}

	// pass the healthcheck
	return c.Client().Agent().PassTTL("service:"+node.ID, "")
}

// GetService returns a service from the registry.
func (c *RegistryConsul) GetService(name string, _ ...registry.GetOption) ([]*registry.Service, error) {
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
		return []*registry.Service{}, registry.ErrNotFound
	}

	serviceMap := map[string]*registry.Service{}

	for _, service := range rsp {
		if service.Service.Service != name {
			continue
		}

		// version is now a tag
		version, _ := decodeVersion(service.Service.Tags)
		// service ID is now the node id
		id := service.Service.ID
		// key is always the version
		key := version

		// address is service address
		address := service.Service.Address

		// use node address
		if len(address) == 0 {
			address = service.Node.Address
		}

		svc, ok := serviceMap[key]
		if !ok {
			svc = &registry.Service{
				Endpoints: decodeEndpoints(service.Service.Tags),
				Name:      service.Service.Service,
				Version:   version,
			}
			serviceMap[key] = svc
		}

		var del bool

		for _, check := range service.Checks {
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

		rNode := &registry.Node{
			ID:       id,
			Address:  maddr.HostPort(address, service.Service.Port),
			Metadata: service.Service.Meta,
		}

		// Extract the transport from Metadata
		if transport, ok := rNode.Metadata[metaTransportKey]; ok {
			rNode.Transport = transport
		}

		svc.Nodes = append(svc.Nodes, rNode)
	}

	services := []*registry.Service{}
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

// ListServices lists services within the registry.
func (c *RegistryConsul) ListServices(_ ...registry.ListOption) ([]*registry.Service, error) {
	rsp, _, err := c.Client().Catalog().Services(c.queryOptions)
	if err != nil {
		return nil, err
	}

	services := []*registry.Service{}

	for service := range rsp {
		services = append(services, &registry.Service{Name: service})
	}

	return services, nil
}

// Watch returns a Watcher which you can watch on.
func (c *RegistryConsul) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
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

// ProvideRegistryConsul creates a new Consul registry.
func ProvideRegistryConsul(
	name types.ServiceName,
	version types.ServiceVersion,
	datas types.ConfigData,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg, err := NewConfig(name, datas, opts...)
	if err != nil {
		return registry.Type{}, fmt.Errorf("create consul registry config: %w", err)
	}

	// Return the new registry.
	reg := New(string(name), string(version), cfg, logger)

	return registry.Type{Registry: reg}, nil
}

// New creates a new consul registry.
func New(serviceName string, _ string, cfg Config, logger log.Logger) *RegistryConsul {
	cRegistry := &RegistryConsul{
		serviceName: serviceName,
		config:      cfg,
		logger:      logger,
		register:    make(map[string]uint64),
		lastChecked: make(map[string]time.Time),
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

	return cRegistry
}

// Start the registry.
func (c *RegistryConsul) Start(_ context.Context) error {
	// setup the client
	c.Client()

	return nil
}

// Stop the registry.
func (c *RegistryConsul) Stop(_ context.Context) error {
	// remove the client
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
