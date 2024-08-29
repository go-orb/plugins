// Package nats provides a NATS registry using broadcast queries
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// This is here to make sure RegistryNATS implements registry.Registry.
var _ registry.Registry = (*RegistryNATS)(nil)

// RegistryNATS implements the registry interface. It runs a NATS service registry.
type RegistryNATS struct {
	serviceName    string
	serviceVersion string
	id             string

	config Config
	logger log.Logger

	sync.RWMutex
	conn      *nats.Conn
	services  map[string][]*registry.Service
	listeners map[string]chan bool
}

// ProvideRegistryNATS creates a new NATS registry.
func ProvideRegistryNATS(
	name types.ServiceName,
	version types.ServiceVersion,
	datas types.ConfigData,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg, err := NewConfig(name, datas, opts...)
	if err != nil {
		return registry.Type{}, fmt.Errorf("create nats registry config: %w", err)
	}

	// Return the new registry.
	reg := New(string(name), string(version), cfg, logger)

	return registry.Type{Registry: reg}, nil
}

// New creates a new NATS registry. This functions should rarely be called manually.
// To create a new registry use ProvideRegistryNATS.
func New(serviceName string, serviceVersion string, cfg Config, log log.Logger) *RegistryNATS {
	if cfg.Timeout == 0 {
		cfg.Timeout = registry.DefaultTimeout
	}

	cfg.Addresses = setAddrs(cfg.Addresses)

	if cfg.QueryTopic == "" {
		cfg.QueryTopic = DefaultQueryTopic
	}

	if cfg.WatchTopic == "" {
		cfg.WatchTopic = DefaultWatchTopic
	}

	return &RegistryNATS{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		config:         cfg,
		logger:         log,
		services:       make(map[string][]*registry.Service),
		listeners:      make(map[string]chan bool),
	}
}

// Start the registry.
func (n *RegistryNATS) Start() error {
	if _, err := n.getConn(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	return nil
}

// Stop the registry.
func (n *RegistryNATS) Stop(_ context.Context) error {
	if !n.conn.IsClosed() {
		n.conn.Close()
	}

	return nil
}

// String returns the plugin name.
func (n *RegistryNATS) String() string {
	return Name
}

// Type returns the component type.
func (n *RegistryNATS) Type() string {
	return registry.ComponentType
}

// ServiceName returns the configured name of this service.
func (n *RegistryNATS) ServiceName() string {
	return n.serviceName
}

// ServiceVersion returns the configured version of this service.
func (n *RegistryNATS) ServiceVersion() string {
	return n.serviceVersion
}

// NodeID returns the ID of this service node in the registry.
func (n *RegistryNATS) NodeID() string {
	if n.id != "" {
		return n.id
	}

	n.id = n.serviceName + "-" + uuid.New().String()

	return n.id
}

func setAddrs(addrs []string) []string {
	var cAddrs []string //nolint:prealloc

	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}

		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}

		cAddrs = append(cAddrs, addr)
	}

	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}

	return cAddrs
}

func (n *RegistryNATS) getConn() (*nats.Conn, error) {
	n.Lock()
	defer n.Unlock()

	if n.conn != nil {
		return n.conn, nil
	}

	opts := nats.GetDefaultOptions()
	opts.Servers = n.config.Addresses
	opts.Secure = n.config.Secure
	opts.TLSConfig = n.config.TLSConfig

	// secure might not be set
	if opts.TLSConfig != nil {
		opts.Secure = true
	}

	c, err := opts.Connect()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", n.config.Addresses[0], err)
	}

	n.conn = c

	return n.conn, nil
}

func (n *RegistryNATS) registerSubscriber(service *registry.Service, conn *nats.Conn) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var result *registry.Result

		if err := json.Unmarshal(msg.Data, &result); err != nil {
			return
		}

		var services []*registry.Service

		switch result.Action {
		// is this a get query and we own the service?
		case "get":
			if result.Service.Name != service.Name {
				return
			}

			n.RLock()
			services = cp(n.services[service.Name])
			n.RUnlock()
		// it's a list request, but we're still only a
		// subscriber for this service... so just get this service
		// totally suboptimal
		case "list":
			n.RLock()
			services = cp(n.services[service.Name])
			n.RUnlock()
		default:
			// does not match
			return
		}

		// respond to query
		for _, service := range services {
			b, err := json.Marshal(service)
			if err != nil {
				continue
			}

			if err := conn.Publish(msg.Reply, b); err != nil {
				continue
			}
		}
	}
}

func (n *RegistryNATS) register(service *registry.Service) error {
	conn, err := n.getConn()
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	// cache service
	n.services[service.Name] = addServices(n.services[service.Name], cp([]*registry.Service{service}))

	// create query listener
	if n.listeners[service.Name] == nil {
		listener := make(chan bool)

		// create a subscriber that responds to queries
		sub, err := conn.Subscribe(n.config.QueryTopic, n.registerSubscriber(service, conn))
		if err != nil {
			return err
		}

		// Unsubscribe if we're told to do so
		go func() {
			<-listener
			sub.Unsubscribe() //nolint:errcheck,gosec
		}()

		n.listeners[service.Name] = listener
	}

	return nil
}

func (n *RegistryNATS) deregister(s *registry.Service) error {
	n.Lock()
	defer n.Unlock()

	services := delServices(n.services[s.Name], cp([]*registry.Service{s}))
	if len(services) > 0 {
		n.services[s.Name] = services
		return nil
	}

	// delete cached service
	delete(n.services, s.Name)

	// delete query listener
	if listener, lexists := n.listeners[s.Name]; lexists {
		close(listener)
		delete(n.listeners, s.Name)
	}

	return nil
}

func (n *RegistryNATS) query(s string, quorum int) ([]*registry.Service, error) { //nolint:gocyclo,funlen
	conn, err := n.getConn()
	if err != nil {
		return nil, err
	}

	var action string

	var service *registry.Service

	if len(s) > 0 {
		action = "get"
		service = &registry.Service{Name: s}
	} else {
		action = "list"
	}

	inbox := nats.NewInbox()

	response := make(chan *registry.Service, 10)

	sub, err := conn.Subscribe(inbox, func(msg *nats.Msg) {
		var service *registry.Service

		if err := json.Unmarshal(msg.Data, &service); err != nil {
			return
		}

		select {
		case response <- service:
		case <-time.After(time.Millisecond * time.Duration(n.config.Timeout)):
		}
	})
	if err != nil {
		return nil, err
	}

	defer sub.Unsubscribe() //nolint:errcheck

	b, err := json.Marshal(&registry.Result{Action: action, Service: service})
	if err != nil {
		return nil, err
	}

	if err := conn.PublishMsg(&nats.Msg{
		Subject: n.config.QueryTopic,
		Reply:   inbox,
		Data:    b,
	}); err != nil {
		return nil, err
	}

	timeout := time.After(time.Millisecond * time.Duration(n.config.Timeout))
	serviceMap := make(map[string]*registry.Service)

loop:
	for {
		select {
		case service := <-response:
			key := service.Name + "-" + service.Version
			srv, ok := serviceMap[key]
			if ok {
				srv.Nodes = append(srv.Nodes, service.Nodes...)
				serviceMap[key] = srv
			} else {
				serviceMap[key] = service
			}

			if quorum > 0 && len(serviceMap[key].Nodes) >= quorum {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	var services []*registry.Service //nolint:prealloc
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

// Register registers a service within the registry.
func (n *RegistryNATS) Register(s *registry.Service, _ ...registry.RegisterOption) error {
	if err := n.register(s); err != nil {
		return err
	}

	conn, err := n.getConn()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&registry.Result{Action: "create", Service: s})
	if err != nil {
		return err
	}

	return conn.Publish(n.config.WatchTopic, b)
}

// Deregister deregisters a service within the registry.
func (n *RegistryNATS) Deregister(
	service *registry.Service, _ ...registry.DeregisterOption,
) error {
	if err := n.deregister(service); err != nil {
		return err
	}

	conn, err := n.getConn()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&registry.Result{Action: "delete", Service: service})
	if err != nil {
		return err
	}

	return conn.Publish(n.config.WatchTopic, b)
}

// GetService returns a service from the registry.
func (n *RegistryNATS) GetService(s string, _ ...registry.GetOption) ([]*registry.Service, error) {
	services, err := n.query(s, n.config.Quorum)
	if err != nil {
		return nil, err
	}

	return services, nil
}

// ListServices lists services within the registry.
func (n *RegistryNATS) ListServices(_ ...registry.ListOption) ([]*registry.Service, error) {
	s, err := n.query("", 0)
	if err != nil {
		return nil, err
	}

	var services []*registry.Service //nolint:prealloc

	serviceMap := make(map[string]*registry.Service)

	for _, v := range s {
		serviceMap[v.Name] = &registry.Service{Name: v.Name}
	}

	for _, v := range serviceMap {
		services = append(services, v)
	}

	return services, nil
}

// Watch returns a registry.Watcher which you can watch on.
func (n *RegistryNATS) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	conn, err := n.getConn()
	if err != nil {
		return nil, err
	}

	sub, err := conn.SubscribeSync(n.config.WatchTopic)
	if err != nil {
		return nil, err
	}

	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	return &natsWatcher{sub, wo}, nil
}
