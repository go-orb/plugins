// Package memory provides a memory-based registry for in-process services.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/google/uuid"
)

// This is here to make sure Registry implements registry.Registry.
var _ registry.Registry = (*Registry)(nil)

type node struct {
	*registry.Node

	LastSeen time.Time
	TTL      time.Duration
}

type record struct {
	Name      string
	Version   string
	Metadata  map[string]string
	Nodes     map[string]*node
	Endpoints []*registry.Endpoint
}

type dataStore struct {
	config Config

	logger log.Logger

	Records  map[string]map[string]*record
	Watchers map[string]*watcher

	startOnce sync.Once
	sync.RWMutex
}

func (d *dataStore) Start(ctx context.Context) {
	d.startOnce.Do(func() {
		go d.ttlPrune(ctx)
	})
}

func (d *dataStore) ttlPrune(ctx context.Context) {
	prune := time.NewTicker(d.config.TTL)
	defer prune.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-prune.C:
			d.Lock()
			for name, records := range d.Records {
				for version, record := range records {
					for id, n := range record.Nodes {
						if n.TTL != 0 && time.Since(n.LastSeen) > n.TTL {
							d.logger.Debug("Registry TTL expired for node of service", "node", n.ID, "service", name)
							delete(d.Records[name][version].Nodes, id)
						}
					}
				}
			}
			d.Unlock()
		}
	}
}

func (d *dataStore) SendEvent(r *registry.Result) {
	d.RLock()
	watchers := make([]*watcher, 0, len(d.Watchers))

	for _, w := range d.Watchers {
		watchers = append(watchers, w)
	}
	d.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.exit:
			d.Lock()
			delete(d.Watchers, w.id)
			d.Unlock()
		default:
			timeout := time.After(d.config.WatcherSendTimeout)
			select {
			case w.res <- r:
			case <-timeout:
			}
		}
	}
}

// Registry is the memory registry for go-orb.
type Registry struct {
	serviceName    string
	serviceVersion string

	config Config

	logger log.Logger

	dataStore *dataStore
}

// ServiceName returns the configured name of this service.
func (c *Registry) ServiceName() string {
	return c.serviceName
}

// ServiceVersion returns the configured version of this service.
func (c *Registry) ServiceVersion() string {
	return c.serviceVersion
}

// Start starts the registry.
func (c *Registry) Start(ctx context.Context) error {
	c.dataStore.Start(ctx)

	return nil
}

// Stop stops the registry.
func (c *Registry) Stop(_ context.Context) error {
	return nil
}

// String returns the plugin name.
func (c *Registry) String() string {
	return Name
}

// Type returns the component type.
func (c *Registry) Type() string {
	return registry.ComponentType
}

// Deregister deregisters a service within the registry.
func (c *Registry) Deregister(s *registry.Service, _ ...registry.DeregisterOption) error {
	c.dataStore.Lock()
	defer c.dataStore.Unlock()

	//nolint:nestif
	if _, ok := c.dataStore.Records[s.Name]; ok {
		if _, ok := c.dataStore.Records[s.Name][s.Version]; ok {
			for _, n := range s.Nodes {
				if _, ok := c.dataStore.Records[s.Name][s.Version].Nodes[n.ID]; ok {
					c.logger.Trace("Registry removed node from service", "name", s.Name, "version", s.Version)
					delete(c.dataStore.Records[s.Name][s.Version].Nodes, n.ID)
				}
			}

			if len(c.dataStore.Records[s.Name][s.Version].Nodes) == 0 {
				delete(c.dataStore.Records[s.Name], s.Version)
				c.logger.Trace("Registry removed service", "name", s.Name, "version", s.Version)
			}
		}

		if len(c.dataStore.Records[s.Name]) == 0 {
			delete(c.dataStore.Records, s.Name)
			c.logger.Trace("Registry removed service", "name", s.Name)
		}

		go c.dataStore.SendEvent(&registry.Result{Action: "delete", Service: s})
	}

	return nil
}

// Register registers a service within the registry.
//
//nolint:funlen
func (c *Registry) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	c.dataStore.Lock()
	defer c.dataStore.Unlock()

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	r := serviceToRecord(service, options.TTL)

	if _, ok := c.dataStore.Records[service.Name]; !ok {
		c.dataStore.Records[service.Name] = make(map[string]*record)
	}

	if _, ok := c.dataStore.Records[service.Name][service.Version]; !ok {
		// New service - store it and we're done
		c.dataStore.Records[service.Name][service.Version] = r
		c.logger.Trace("Registry added new service", "name", service.Name, "version", service.Version)

		go c.dataStore.SendEvent(&registry.Result{Action: "create", Service: service})

		return nil
	}

	// Existing service - update record
	existingRecord := c.dataStore.Records[service.Name][service.Version]

	// Update the service metadata
	existingRecord.Metadata = make(map[string]string)
	for k, v := range service.Metadata {
		existingRecord.Metadata[k] = v
	}

	// Update the endpoints
	existingRecord.Endpoints = service.Endpoints

	// Track if we made any changes
	changes := false

	// Handle nodes
	for _, newNode := range service.Nodes {
		if existingNode, ok := existingRecord.Nodes[newNode.ID]; !ok { //nolint:nestif
			// This is a new node, add it
			changes = true
			metadata := make(map[string]string)

			for k, v := range newNode.Metadata {
				metadata[k] = v
			}

			existingRecord.Nodes[newNode.ID] = &node{
				Node: &registry.Node{
					ID:        newNode.ID,
					Address:   newNode.Address,
					Transport: newNode.Transport,
					Metadata:  metadata,
				},
				TTL:      options.TTL,
				LastSeen: time.Now(),
			}
		} else {
			// This is an existing node, update it
			if existingNode.Address != newNode.Address {
				existingNode.Address = newNode.Address
				changes = true
			}

			if existingNode.Transport != newNode.Transport {
				existingNode.Transport = newNode.Transport
				changes = true
			}

			// Update metadata
			for k, v := range newNode.Metadata {
				if existingValue, ok := existingNode.Metadata[k]; !ok || existingValue != v {
					if existingNode.Metadata == nil {
						existingNode.Metadata = make(map[string]string)
					}

					existingNode.Metadata[k] = v
					changes = true
				}
			}

			// Always update TTL and LastSeen
			existingNode.TTL = options.TTL
			existingNode.LastSeen = time.Now()
		}
	}

	// If we made changes or this is a regular TTL refresh, send an update
	if changes {
		c.logger.Debug("Updated service", "name", service.Name, "version", service.Version)
	} else {
		c.logger.Debug("Refreshed service TTL", "name", service.Name, "version", service.Version)
	}

	// Send an update event regardless of changes to maintain expected behavior
	go c.dataStore.SendEvent(&registry.Result{Action: "update", Service: service})

	return nil
}

// GetService returns a service from the registry.
func (c *Registry) GetService(name string, _ ...registry.GetOption) ([]*registry.Service, error) {
	c.dataStore.RLock()
	defer c.dataStore.RUnlock()

	records, ok := c.dataStore.Records[name]
	if !ok {
		return nil, registry.ErrNotFound
	}

	services := make([]*registry.Service, len(c.dataStore.Records[name]))
	i := 0

	for _, record := range records {
		services[i] = recordToService(record)
		i++
	}

	return services, nil
}

// ListServices lists services within the registry.
func (c *Registry) ListServices(_ ...registry.ListOption) ([]*registry.Service, error) {
	c.dataStore.RLock()
	defer c.dataStore.RUnlock()

	var services []*registry.Service

	for _, records := range c.dataStore.Records {
		for _, record := range records {
			services = append(services, recordToService(record))
		}
	}

	return services, nil
}

// Watch returns a Watcher which you can watch on.
func (c *Registry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	w := &watcher{
		exit: make(chan bool),
		res:  make(chan *registry.Result),
		id:   uuid.New().String(),
		wo:   wo,
	}

	c.dataStore.Lock()
	c.dataStore.Watchers[w.id] = w
	c.dataStore.Unlock()

	return w, nil
}

// Provide creates a new memory registry.
func Provide(
	name string,
	version string,
	datas map[string]any,
	_ *types.Components,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, registry.DefaultConfigSection, datas, &cfg); err != nil {
		return registry.Type{}, err
	}

	reg := New(name, version, cfg, logger)

	return registry.Type{Registry: reg}, nil
}

//nolint:gochecknoglobals
var store *dataStore

// New creates a new memory registry.
func New(serviceName string, serviceVersion string, cfg Config, logger log.Logger) *Registry {
	if store == nil {
		store = &dataStore{
			config:   cfg,
			logger:   logger,
			Records:  make(map[string]map[string]*record),
			Watchers: make(map[string]*watcher),
		}
	}

	instance := &Registry{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		config:         cfg,
		logger:         logger,

		dataStore: store,
	}

	return instance
}
