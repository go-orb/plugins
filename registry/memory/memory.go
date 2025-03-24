// Package memory provides a memory-based registry for in-process services.
package memory

import (
	"context"
	"errors"
	"slices"
	"strings"
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

type serviceNode struct {
	registry.ServiceNode
	LastSeen time.Time
}

type dataStore struct {
	config Config

	logger log.Logger

	Records  map[string]serviceNode
	Watchers map[string]*watcher

	startOnce sync.Once
	sync.RWMutex
}

func nodeKey(s registry.ServiceNode) string {
	return strings.Join([]string{
		s.Namespace,
		s.Region,
		s.Name,
		s.Version,
		s.Scheme,
		s.Address,
	}, ":")
}

func (d *dataStore) Start(ctx context.Context) {
	d.startOnce.Do(func() {
		go d.ttlPrune(ctx)
	})
}

func (d *dataStore) ttlPrune(ctx context.Context) {
	prune := time.NewTicker(time.Duration(d.config.TTL))
	defer prune.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-prune.C:
			d.Lock()
			toDelete := []string{}

			for key, record := range d.Records {
				if record.ServiceNode.TTL != 0 && time.Since(record.LastSeen) > record.ServiceNode.TTL {
					d.logger.Debug("Registry TTL expired for node of service", "address", record.ServiceNode.Address, "service", record.ServiceNode.Name)

					go d.SendEvent(&registry.Result{Action: registry.Delete, Node: record.ServiceNode})

					toDelete = append(toDelete, key)
				}
			}

			for _, key := range toDelete {
				delete(d.Records, key)
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
		case <-w.ctx.Done():
			d.Lock()
			delete(d.Watchers, w.id)
			d.Unlock()
		default:
			timeout := time.After(time.Duration(d.config.WatcherSendTimeout))
			select {
			case w.res <- r:
			case <-timeout:
			}
		}
	}
}

// Registry is the memory registry for go-orb.
type Registry struct {
	config Config

	logger log.Logger

	dataStore *dataStore
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
func (c *Registry) Deregister(_ context.Context, s registry.ServiceNode) error {
	c.dataStore.Lock()
	defer c.dataStore.Unlock()

	key := nodeKey(s)

	if _, ok := c.dataStore.Records[key]; ok {
		c.logger.Debug("Registry deregister service", "name", s.Name)
		delete(c.dataStore.Records, key)

		go c.dataStore.SendEvent(&registry.Result{Action: registry.Delete, Node: s})

		return nil
	}

	return nil
}

// Register registers a service within the registry.
func (c *Registry) Register(_ context.Context, s registry.ServiceNode) error {
	c.dataStore.Lock()
	defer c.dataStore.Unlock()

	key := nodeKey(s)
	if _, ok := c.dataStore.Records[key]; ok {
		c.logger.Debug("Registry updated service", "name", s.Name, "version", s.Version)

		c.dataStore.Records[key] = serviceNode{
			ServiceNode: s,
			LastSeen:    time.Now(),
		}

		go c.dataStore.SendEvent(&registry.Result{Action: registry.Update, Node: s})

		return nil
	}

	c.dataStore.Records[key] = serviceNode{
		ServiceNode: s,
		LastSeen:    time.Now(),
	}

	c.logger.Debug("Registry registered service", "name", s.Name, "version", s.Version)

	go c.dataStore.SendEvent(&registry.Result{Action: registry.Create, Node: s})

	return nil
}

// GetService returns a service from the registry.
func (c *Registry) GetService(_ context.Context, namespace, region, name string, schemes []string) ([]registry.ServiceNode, error) {
	c.dataStore.RLock()
	defer c.dataStore.RUnlock()

	services := []registry.ServiceNode{}

	for _, record := range c.dataStore.Records {
		if name != "" && record.Name != name {
			continue
		}

		if namespace != "" && record.Namespace != namespace {
			continue
		}

		if region != "" && record.Region != region {
			continue
		}

		if len(schemes) > 0 && !slices.Contains(schemes, record.Scheme) {
			continue
		}

		services = append(services, record.ServiceNode)
	}

	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	return services, nil
}

// ListServices lists services within the registry.
func (c *Registry) ListServices(_ context.Context, namespace, region string, schemes []string) ([]registry.ServiceNode, error) {
	c.dataStore.RLock()
	defer c.dataStore.RUnlock()

	services := []registry.ServiceNode{}

	for _, record := range c.dataStore.Records {
		if namespace != "" && record.Namespace != namespace {
			continue
		}

		if region != "" && record.Region != region {
			continue
		}

		if len(schemes) > 0 && !slices.Contains(schemes, record.Scheme) {
			continue
		}

		services = append(services, record.ServiceNode)
	}

	return services, nil
}

// Watch returns a Watcher which you can watch on.
func (c *Registry) Watch(ctx context.Context, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	w := &watcher{
		ctx: ctx,
		res: make(chan *registry.Result),
		id:  uuid.New().String(),
		wo:  wo,
	}

	c.dataStore.Lock()
	c.dataStore.Watchers[w.id] = w
	c.dataStore.Unlock()

	return w, nil
}

// Provide creates a new memory registry.
func Provide(
	datas map[string]any,
	_ *types.Components,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, registry.DefaultConfigSection, datas, &cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return registry.Type{}, err
	}

	reg := New(cfg, logger)

	return registry.Type{Registry: reg}, nil
}

//nolint:gochecknoglobals
var store *dataStore

// New creates a new memory registry.
func New(cfg Config, logger log.Logger) *Registry {
	if store == nil {
		store = &dataStore{
			config:   cfg,
			logger:   logger,
			Records:  make(map[string]serviceNode),
			Watchers: make(map[string]*watcher),
		}
	}

	instance := &Registry{
		config: cfg,
		logger: logger,

		dataStore: store,
	}

	return instance
}
