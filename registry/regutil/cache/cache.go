// Package cache provides a cache for registry plugins.
package cache

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
)

func serviceKey(s registry.ServiceNode) string {
	return strings.Join([]string{s.Namespace, s.Region, s.Name}, "@")
}

func nodeKey(s registry.ServiceNode) string {
	return strings.Join([]string{s.Namespace, s.Region, s.Name, s.Version, s.Scheme, s.Address}, "@")
}

type dataStore struct {
	logger log.Logger

	registry registry.Registry

	Records map[string]map[string]registry.ServiceNode

	watcher     registry.Watcher
	watchCancel context.CancelFunc

	startOnce sync.Once
	sync.RWMutex

	regCount int64
}

// populate initializes the cache with services from the registry.
func (d *dataStore) populate(ctx context.Context) {
	services, err := d.registry.ListServices(ctx, "", "", nil)
	if err != nil {
		d.logger.Warn("Failed to list services when populating cache", "error", err)
		return
	}

	// Create a map to track unique service names
	serviceNames := make(map[string]registry.ServiceNode)
	for _, service := range services {
		serviceNames[nodeKey(service)] = service
	}

	d.Lock()
	defer d.Unlock()

	// Fetch full details for each service
	for _, svc := range serviceNames {
		serviceNodes, err := d.registry.GetService(ctx, svc.Namespace, svc.Region, svc.Name, nil)
		if err != nil {
			if !errors.Is(err, registry.ErrNotFound) {
				d.logger.Warn("Failed to get service details when populating cache", "name", svc.Name, "error", err)
			}

			continue
		}

		for _, serviceNode := range serviceNodes {
			d.registerServiceNodeInternal(serviceNode)
		}
	}

	d.logger.Debug("Populated with services from registry")
}

// startWatching begins watching a registry for service changes and updates the cache accordingly.
// This keeps the cache in sync with the registry.
func (d *dataStore) startWatching(ctx context.Context) error {
	d.Lock()
	defer d.Unlock()

	// Start the watcher.
	var err error

	// Create a context for this watcher.
	var watchCtx context.Context
	watchCtx, d.watchCancel = context.WithCancel(ctx)

	d.watcher, err = d.registry.Watch(watchCtx)
	if err != nil {
		return err
	}

	// Start watching in a goroutine.
	go d.watch(watchCtx)

	d.logger.Debug("Started watching registry for changes")

	return nil
}

// watch processes events from the watcher and updates the cache.
func (d *dataStore) watch(ctx context.Context) {
	for {
		// Check if the context is done.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get the next result.
		result, err := d.watcher.Next()
		if err != nil {
			if errors.Is(err, registry.ErrWatcherStopped) {
				d.logger.Debug("watcher stopped")
				return
			}

			d.logger.Warn("getting next event from watcher", "error", err)

			continue
		}

		// Process the result based on the action.
		switch result.Action {
		case registry.Create, registry.Update:
			d.registerServiceNode(result.Node)
		case registry.Delete:
			d.logger.Trace("deregister from watcher")

			if err := d.deregisterServiceNode(result.Node); err != nil {
				d.logger.Warn("deregistering service from watcher", "error", err)
			}
		default:
			d.logger.Warn("unknown watch action", "action", result.Action)
		}
	}
}

// deregisterServiceNode removes a service from the cache.
func (d *dataStore) deregisterServiceNode(serviceNode registry.ServiceNode) error {
	d.Lock()
	defer d.Unlock()

	sKey := serviceKey(serviceNode)
	nKey := nodeKey(serviceNode)

	if records, ok := d.Records[sKey]; ok {
		if _, ok := records[nKey]; ok {
			d.logger.Trace("deregister",
				"namespace", serviceNode.Namespace,
				"region", serviceNode.Region,
				"name", serviceNode.Name,
				"version", serviceNode.Version,
				"address", serviceNode.Address,
				"scheme", serviceNode.Scheme,
			)

			delete(records, nKey)
		}

		if len(d.Records[sKey]) == 0 {
			delete(d.Records, sKey)
		}

		return nil
	}

	return nil
}

// registerServiceNode adds or updates a service in the cache.
func (d *dataStore) registerServiceNode(serviceNode registry.ServiceNode) {
	d.registerServiceNodeInternal(serviceNode)
}

// registerServiceNodeInternal adds or updates a service in the cache (internal method).
func (d *dataStore) registerServiceNodeInternal(serviceNode registry.ServiceNode) {
	d.Lock()
	defer d.Unlock()

	sKey := serviceKey(serviceNode)
	nKey := nodeKey(serviceNode)

	if _, ok := d.Records[sKey]; ok {
		records := d.Records[sKey]
		if _, ok := records[nKey]; ok {
			records[nKey] = serviceNode
		} else {
			d.Records[sKey][nKey] = serviceNode
		}
	} else {
		d.Records[sKey] = map[string]registry.ServiceNode{}
		d.Records[sKey][nKey] = serviceNode
	}
}

//nolint:gochecknoglobals
var store = &dataStore{
	Records: make(map[string]map[string]registry.ServiceNode),
}

func (d *dataStore) start(ctx context.Context, logger log.Logger, registry registry.Registry) {
	d.startOnce.Do(func() {
		d.logger = logger
		d.registry = registry

		// Start watching for updates.
		if err := d.startWatching(ctx); err != nil {
			d.logger.Warn("Failed to start watching registry", "error", err)
			return
		}

		d.populate(ctx)
	})

	atomic.AddInt64(&d.regCount, 1)
}

func (d *dataStore) stop(_ context.Context) {
	atomic.AddInt64(&d.regCount, -1)

	if atomic.LoadInt64(&d.regCount) == 0 {
		d.watchCancel()
	}
}

// Cache is a utility for caching registry operations.
type Cache struct {
	// config holds cache configuration.
	config Config

	// registry is the registry this cache is using.
	registry registry.Registry

	// logger for this cache instance.
	logger log.Logger
}

// Start starts the cache pruning mechanism and begins watching the registry.
func (c *Cache) Start(ctx context.Context) error {
	store.start(ctx, c.logger, c.registry)

	return nil
}

// Stop stops the cache and its watcher.
func (c *Cache) Stop(ctx context.Context) error {
	store.stop(ctx)

	return nil
}

// String returns the name of the cache.
func (c *Cache) String() string {
	return "registry-cache"
}

// GetService returns a service from the cache.
func (c *Cache) GetService(_ context.Context, namespace, region, name string, schemes []string) ([]registry.ServiceNode, error) {
	store.RLock()
	defer store.RUnlock()

	services := []registry.ServiceNode{}

	for _, records := range store.Records {
		if len(records) < 1 {
			continue
		}

		randomRecord := registry.ServiceNode{}
		for _, record := range records {
			randomRecord = record
			break
		}

		if randomRecord.Namespace != namespace {
			continue
		}

		if randomRecord.Region != region {
			continue
		}

		if randomRecord.Name != name {
			continue
		}

		for _, record := range records {
			if len(schemes) > 0 && !slices.Contains(schemes, record.Scheme) {
				continue
			}

			services = append(services, record)
		}
	}

	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	return services, nil
}

// ListServices lists services within the cache.
func (c *Cache) ListServices(_ context.Context, namespace, region string, schemes []string) ([]registry.ServiceNode, error) {
	store.RLock()
	defer store.RUnlock()

	services := []registry.ServiceNode{}

	for _, records := range store.Records {
		if len(records) < 1 {
			continue
		}

		randomRecord := registry.ServiceNode{}
		for _, record := range records {
			randomRecord = record
			break
		}

		if randomRecord.Namespace != namespace {
			continue
		}

		if randomRecord.Region != region {
			continue
		}

		for _, record := range records {
			if len(schemes) > 0 && !slices.Contains(schemes, record.Scheme) {
				continue
			}

			services = append(services, record)
		}
	}

	return services, nil
}

// Register registers a service with the cache.
func (c *Cache) Register(_ context.Context, serviceNode registry.ServiceNode) error {
	store.registerServiceNode(serviceNode)
	return nil
}

// Deregister deregisters a service with the cache.
func (c *Cache) Deregister(_ context.Context, serviceNode registry.ServiceNode) error {
	return store.deregisterServiceNode(serviceNode)
}

// New creates a new registry cache.
func New(cfg Config, logger log.Logger, registry registry.Registry) *Cache {
	return &Cache{
		config:   cfg,
		registry: registry,
		logger:   logger.With("subcomponent", "registry-cache"),
	}
}
