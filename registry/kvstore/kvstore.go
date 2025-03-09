// Package kvstore provides a registry plugin based on a key-value store.
package kvstore

import (
	"context"
	"errors"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/google/uuid"
)

// This is here to make sure Registry implements registry.Registry.
var _ registry.Registry = (*Registry)(nil)

// Registry is the memory registry for go-orb.
type Registry struct {
	ctx context.Context

	serviceName    string
	serviceVersion string

	codec codecs.Marshaler

	config Config

	id string

	logger log.Logger

	kvstore kvstore.Type
}

// ServiceName returns the configured name of this service.
func (c *Registry) ServiceName() string {
	return c.serviceName
}

// ServiceVersion returns the configured version of this service.
func (c *Registry) ServiceVersion() string {
	return c.serviceVersion
}

// NodeID returns the ID of this service node in the registry.
func (c *Registry) NodeID() string {
	if c.id != "" {
		return c.id
	}

	c.id = uuid.New().String()

	return c.id
}

// Start starts the registry.
func (c *Registry) Start(ctx context.Context) error {
	c.ctx = ctx
	return c.kvstore.Start(ctx)
}

// Stop stops the registry.
func (c *Registry) Stop(ctx context.Context) error {
	return c.kvstore.Stop(ctx)
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
	key := s.Name + c.config.ServiceDelimiter + c.NodeID() + c.config.ServiceDelimiter + s.Version
	c.logger.Trace("deregistering service", "service", s, "key", key)

	return c.kvstore.Purge(key, c.config.Database, c.config.Table)
}

// Register registers a service within the registry.
func (c *Registry) Register(service *registry.Service, _ ...registry.RegisterOption) error {
	if service == nil {
		return orberrors.ErrBadRequest.Wrap(errors.New("wont store nil service"))
	}

	b, err := c.codec.Encode(service)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	key := service.Name + c.config.ServiceDelimiter + c.NodeID() + c.config.ServiceDelimiter + service.Version

	c.logger.Trace("registering service", "service", service, "key", key)

	return c.kvstore.Set(
		key,
		c.config.Database,
		c.config.Table,
		b,
	)
}

// GetService returns a service from the registry.
func (c *Registry) GetService(name string, _ ...registry.GetOption) ([]*registry.Service, error) {
	services, err := c.listServices(kvstore.KeysPrefix(name + c.config.ServiceDelimiter))
	if err != nil {
		return nil, err
	}

	// If no services found, return ErrNotFound
	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	return services, nil
}

// ListServices lists services within the registry.
func (c *Registry) ListServices(_ ...registry.ListOption) ([]*registry.Service, error) {
	return c.listServices()
}

// Watch returns a Watcher which you can watch on.
func (c *Registry) Watch(_ ...registry.WatchOption) (registry.Watcher, error) {
	return NewWatcher(c)
}

func (c *Registry) listServices(opts ...kvstore.KeysOption) ([]*registry.Service, error) {
	keys, err := c.kvstore.Keys(c.config.Database, c.config.Table, opts...)
	if err != nil {
		return nil, err
	}

	// Use name+version as the key for grouping services
	serviceMap := map[string]*registry.Service{}

	for _, k := range keys {
		s, err := c.getNode(k)
		if err != nil {
			if errors.Is(err, registry.ErrNotFound) {
				// Skip not found errors and continue
				continue
			}

			return nil, err
		}

		// Create a unique key for this service name and version
		key := s.Name + "-" + s.Version

		if serviceMap[key] == nil {
			// First time seeing this service name+version
			serviceMap[key] = s
		} else {
			// Add nodes to existing service entry
			serviceMap[key].Nodes = append(serviceMap[key].Nodes, s.Nodes...)
		}
	}

	svcs := make([]*registry.Service, 0, len(serviceMap))
	for _, s := range serviceMap {
		svcs = append(svcs, s)
	}

	return svcs, nil
}

// getNode retrieves a node from the store. It returns a service to also keep track of the version.
func (c *Registry) getNode(s string) (*registry.Service, error) {
	recs, err := c.kvstore.Get(s, c.config.Database, c.config.Table)
	if err != nil {
		return nil, err
	}

	if len(recs) == 0 {
		return nil, registry.ErrNotFound
	}

	var svc registry.Service
	if err := c.codec.Decode(recs[0].Value, &svc); err != nil {
		return nil, err
	}

	return &svc, nil
}

// Provide creates a new memory registry.
func Provide(
	name types.ServiceName,
	version types.ServiceVersion,
	datas types.ConfigData,
	components *types.Components,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg := NewConfig(opts...)

	sections := types.SplitServiceName(name)
	sections = append(sections, client.DefaultConfigSection)

	if err := config.Parse(sections, datas, &cfg); err != nil {
		return registry.Type{}, err
	}

	kvstore, err := kvstore.Provide(
		types.JoinServiceName(sections),
		datas,
		components,
		logger,
	)
	if err != nil {
		return registry.Type{}, err
	}

	reg, err := New(string(name), string(version), cfg, logger, kvstore)

	return registry.Type{Registry: reg}, err
}

// New creates a new memory registry.
func New(
	serviceName string,
	serviceVersion string,
	cfg Config,
	logger log.Logger,
	kvstore kvstore.Type,
) (*Registry, error) {
	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		return nil, err
	}

	return &Registry{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		config:         cfg,
		logger:         logger,
		codec:          codec,
		kvstore:        kvstore,
	}, nil
}
