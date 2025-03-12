// Package cache provides a cache for registry plugins.
package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
)

type node struct {
	registry.Node
}

type record struct {
	Name      string
	Version   string
	Metadata  map[string]string
	Nodes     map[string]node
	Endpoints []registry.Endpoint
}

type dataStore struct {
	logger log.Logger

	registry registry.Registry

	Records map[string]map[string]record

	watcher     registry.Watcher
	watchCancel context.CancelFunc

	startOnce sync.Once
	sync.RWMutex
}

// populate initializes the cache with services from the registry.
func (d *dataStore) populate() {
	services, err := d.registry.ListServices()
	if err != nil {
		d.logger.Warn("Failed to list services when populating cache", "error", err)
		return
	}

	store.Lock()
	defer store.Unlock()

	for _, service := range services {
		fullServices, err := d.registry.GetService(service.Name)
		if err != nil {
			d.logger.Warn("Failed to get service details when populating cache", "name", service.Name, "error", err)
			continue
		}

		for _, fullService := range fullServices {
			d.registerServiceInternal(fullService)
		}
	}

	d.logger.Debug("Populated cache with services from registry")
}

// startWatching begins watching a registry for service changes and updates the cache accordingly.
// This keeps the cache in sync with the registry.
func (d *dataStore) startWatching(ctx context.Context) error {
	d.Lock()
	defer d.Unlock()

	// Start the watcher.
	var err error

	d.watcher, err = d.registry.Watch()
	if err != nil {
		return err
	}

	// Create a context for this watcher.
	ctx, d.watchCancel = context.WithCancel(ctx)

	// Start watching in a goroutine.
	go d.watch(ctx)

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
				d.logger.Debug("Registry watcher stopped")
				return
			}

			d.logger.Warn("Error getting next event from registry watcher", "error", err)

			continue
		}

		// Process the result based on the action.
		switch result.Action {
		case "create", "update":
			d.registerService(result.Service)
		case "delete":
			if err := d.deregisterService(result.Service); err != nil {
				d.logger.Warn("Error deregistering service from watcher", "error", err)
			}
		default:
			d.logger.Warn("Unknown watch action", "action", result.Action)
		}
	}
}

// deregisterService removes a service from the cache.
func (d *dataStore) deregisterService(s *registry.Service) error {
	store.Lock()
	defer store.Unlock()

	if s == nil {
		return nil
	}

	// If service doesn't exist in our records, nothing to do
	records, ok := d.Records[s.Name]
	if !ok {
		return nil
	}

	// Get the version record if it exists
	versionRecord, ok := records[s.Version]
	if !ok {
		return nil
	}

	// Remove the specific nodes
	for _, node := range s.Nodes {
		delete(versionRecord.Nodes, node.ID)
	}

	// Clean up empty records
	if len(versionRecord.Nodes) == 0 {
		delete(records, s.Version)

		if len(records) == 0 {
			d.logger.Trace("Registry removed service completely", "name", s.Name)
			delete(d.Records, s.Name)
		}
	}

	return nil
}

// registerService adds or updates a service in the cache.
func (d *dataStore) registerService(service *registry.Service) {
	d.Lock()
	defer d.Unlock()

	d.registerServiceInternal(service)
}

// registerServiceInternal adds or updates a service in the cache (internal method).
func (d *dataStore) registerServiceInternal(service *registry.Service) {
	if service == nil {
		return
	}

	r := serviceToRecord(service)

	// Create map if it doesn't exist.
	if _, ok := d.Records[service.Name]; !ok {
		d.Records[service.Name] = make(map[string]record)
	}

	if _, ok := d.Records[service.Name][service.Version]; !ok {
		// New service - store it and we're done.
		d.Records[service.Name][service.Version] = r
		d.logger.Trace("registry cache added new service", "name", service.Name, "version", service.Version)

		return
	}

	// Existing service - update record.
	existingRecord := d.Records[service.Name][service.Version]

	// Update the service metadata.
	existingRecord.Metadata = make(map[string]string)
	for k, v := range service.Metadata {
		existingRecord.Metadata[k] = v
	}

	// Update the endpoints.
	for _, ep := range service.Endpoints {
		existingRecord.Endpoints = append(existingRecord.Endpoints, registry.Endpoint{
			Name:     ep.Name,
			Request:  ep.Request,
			Response: ep.Response,
			Metadata: ep.Metadata,
		})
	}

	// Track if we made any changes.
	changes := false

	// Handle nodes.
	for _, newNode := range service.Nodes {
		if existingNode, ok := existingRecord.Nodes[newNode.ID]; !ok { //nolint:nestif
			// This is a new node, add it.
			changes = true
			metadata := make(map[string]string)

			for k, v := range newNode.Metadata {
				metadata[k] = v
			}

			existingRecord.Nodes[newNode.ID] = node{
				Node: *newNode,
			}
		} else {
			// This is an existing node, update it.
			if existingNode.Node.Address != newNode.Address {
				existingNode.Node.Address = newNode.Address
				changes = true
			}

			// Update metadata.
			for k, v := range newNode.Metadata {
				if existingValue, ok := existingNode.Metadata[k]; !ok || existingValue != v {
					if existingNode.Metadata == nil {
						existingNode.Metadata = make(map[string]string)
					}

					existingNode.Metadata[k] = v
					changes = true
				}
			}
		}
	}

	// If we made changes or this is a regular TTL refresh, send an update.
	if changes {
		d.logger.Debug("Updated service", "name", service.Name, "version", service.Version)
	} else {
		d.logger.Trace("Refreshed service TTL", "name", service.Name, "version", service.Version)
	}

	d.Records[service.Name][service.Version] = existingRecord
}

//nolint:gochecknoglobals
var store = &dataStore{
	Records: make(map[string]map[string]record),
}

func (d *dataStore) Start(ctx context.Context, logger log.Logger, registry registry.Registry) {
	d.startOnce.Do(func() {
		d.logger = logger
		d.registry = registry

		// Start watching for updates.
		if err := d.startWatching(ctx); err != nil {
			d.logger.Warn("Failed to start watching registry", "error", err)
			return
		}

		d.populate()
	})
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
	store.Start(ctx, c.logger, c.registry)

	return nil
}

// Stop stops the cache and its watcher.
func (c *Cache) Stop(_ context.Context) error {
	return nil
}

// GetService returns a service from the cache.
func (c *Cache) GetService(name string, _ ...registry.GetOption) ([]*registry.Service, error) {
	store.RLock()
	defer store.RUnlock()

	records, ok := store.Records[name]
	if !ok {
		return nil, registry.ErrNotFound
	}

	services := make([]*registry.Service, 0, len(records))
	for _, record := range records {
		// Convert record back to service.
		services = append(services, recordToService(record))
	}

	return services, nil
}

// ListServices lists services within the cache.
func (c *Cache) ListServices(_ ...registry.ListOption) ([]*registry.Service, error) {
	store.RLock()
	defer store.RUnlock()

	var services []*registry.Service

	for _, records := range store.Records {
		for _, record := range records {
			services = append(services, recordToService(record))
		}
	}

	return services, nil
}

// New creates a new registry cache.
func New(cfg Config, logger log.Logger, registry registry.Registry) *Cache {
	return &Cache{
		config:   cfg,
		registry: registry,
		logger:   logger,
	}
}
