// Package kvstore provides a registry plugin based on a key-value store.
package kvstore

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/registry/regutil/cache"
)

func nodeKey(s registry.ServiceNode, delimiter string) string {
	return strings.Join([]string{
		s.Namespace,
		s.Region,
		s.Name,
		s.Version,
		s.Node,
	}, delimiter)
}

func keyToServiceNode(key string, delimiter string) (registry.ServiceNode, error) {
	parts := strings.Split(key, delimiter)
	if len(parts) != 5 {
		return registry.ServiceNode{}, errors.New("invalid key format")
	}

	return registry.ServiceNode{
		Namespace: parts[0],
		Region:    parts[1],
		Name:      parts[2],
		Version:   parts[3],
		Node:      parts[4],
	}, nil
}

// This is here to make sure Registry implements registry.Registry.
var _ registry.Registry = (*Registry)(nil)

// Registry is the memory registry for go-orb.
type Registry struct {
	ctx context.Context

	codec codecs.Marshaler

	config Config

	logger log.Logger

	kvstore kvstore.Type

	// cache is used to cache registry operations.
	cache *cache.Cache
}

// Start starts the registry.
func (c *Registry) Start(ctx context.Context) error {
	c.ctx = ctx

	// Start the kvstore
	if err := c.kvstore.Start(ctx); err != nil {
		return err
	}

	// Start the cache - this will populate it and begin watching for changes
	if c.config.Cache {
		return c.cache.Start(ctx)
	}

	return nil
}

// Stop stops the registry.
func (c *Registry) Stop(ctx context.Context) error {
	// Stop the cache first
	if c.config.Cache {
		if err := c.cache.Stop(ctx); err != nil {
			c.logger.Warn("Error stopping cache", "error", err)
		}
	}

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
func (c *Registry) Deregister(_ context.Context, serviceNode registry.ServiceNode) error {
	key := nodeKey(serviceNode, c.config.ServiceDelimiter)
	c.logger.Trace("deregistering service", "serviceNode", serviceNode, "key", key)

	return c.kvstore.Purge(key, c.config.Database, c.config.Table)
}

// Register registers a service within the registry.
func (c *Registry) Register(_ context.Context, serviceNode registry.ServiceNode) error {
	if err := serviceNode.Valid(); err != nil {
		return err
	}

	key := nodeKey(serviceNode, c.config.ServiceDelimiter)
	c.logger.Trace("registering service", "serviceNode", serviceNode, "key", key)

	b, err := c.codec.Marshal(serviceNode)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return c.kvstore.Set(key, c.config.Database, c.config.Table, b)
}

// GetService returns a service from the registry.
func (c *Registry) GetService(ctx context.Context, namespace, region, name string, schemes []string) ([]registry.ServiceNode, error) {
	if c.config.Cache {
		return c.cache.GetService(ctx, namespace, region, name, schemes)
	}

	key := strings.Join([]string{
		namespace,
		region,
		name,
	}, c.config.ServiceDelimiter)

	key += c.config.ServiceDelimiter

	services, err := c.listServices(ctx, kvstore.KeysPrefix(key))
	if err != nil {
		return nil, err
	}

	// Filter by schemes if specified
	if len(schemes) > 0 {
		var filtered []registry.ServiceNode

		for _, service := range services {
			if slices.Contains(schemes, service.Scheme) {
				filtered = append(filtered, service)
			}
		}

		services = filtered
	}

	// If no services found, return ErrNotFound
	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	return services, nil
}

// ListServices lists services within the registry.
func (c *Registry) ListServices(ctx context.Context, namespace, region string, schemes []string) ([]registry.ServiceNode, error) {
	if c.config.Cache {
		return c.cache.ListServices(ctx, namespace, region, schemes)
	}

	key := strings.Join([]string{
		namespace,
		region,
	}, c.config.ServiceDelimiter)

	key += c.config.ServiceDelimiter

	services, err := c.listServices(ctx, kvstore.KeysPrefix(key))
	if err != nil {
		return nil, err
	}

	// Filter by schemes if specified
	if len(schemes) > 0 {
		var filtered []registry.ServiceNode

		for _, service := range services {
			if slices.Contains(schemes, service.Scheme) {
				filtered = append(filtered, service)
			}
		}

		services = filtered
	}

	return services, nil
}

// Watch returns a Watcher which you can watch on.
func (c *Registry) Watch(_ context.Context, _ ...registry.WatchOption) (registry.Watcher, error) {
	return NewWatcher(c)
}

func (c *Registry) listServices(
	_ context.Context,
	opts ...kvstore.KeysOption,
) ([]registry.ServiceNode, error) {
	keys, err := c.kvstore.Keys(c.config.Database, c.config.Table, opts...)
	if err != nil {
		return nil, err
	}

	// Map to store unique service nodes
	result := []registry.ServiceNode{}

	for _, k := range keys {
		svc, err := c.getNode(k)
		if err != nil {
			if errors.Is(err, registry.ErrNotFound) {
				// Skip not found errors and continue
				continue
			}

			return nil, err
		}

		result = append(result, svc)
	}

	return result, nil
}

// getNode retrieves a node from the store.
func (c *Registry) getNode(s string) (registry.ServiceNode, error) {
	recs, err := c.kvstore.Get(s, c.config.Database, c.config.Table)
	if err != nil {
		return registry.ServiceNode{}, err
	}

	if len(recs) == 0 {
		return registry.ServiceNode{}, registry.ErrNotFound
	}

	var svc registry.ServiceNode
	if err := c.codec.Unmarshal(recs[0].Value, &svc); err != nil {
		return registry.ServiceNode{}, err
	}

	return svc, nil
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

	kvstoreDatas := map[string]any{}

	tmp, err := config.WalkMap([]string{registry.DefaultConfigSection}, datas)
	if err != nil {
		kvstoreDatas = tmp
	}

	kvstore, err := kvstore.New(
		kvstoreDatas,
		logger,
	)
	if err != nil {
		return registry.Type{}, err
	}

	reg, err := New(cfg, logger, kvstore)

	return registry.Type{Registry: reg}, err
}

// New creates a new memory registry.
func New(
	cfg Config,
	logger log.Logger,
	kvstore kvstore.Type,
) (*Registry, error) {
	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		return nil, err
	}

	reg := &Registry{
		config:  cfg,
		logger:  logger.With("component", "registry-kvstore"),
		codec:   codec,
		kvstore: kvstore,
	}

	// Initialize the cache with a reference to this registry
	reg.cache = cache.New(cache.Config{}, logger, reg)

	return reg, nil
}
