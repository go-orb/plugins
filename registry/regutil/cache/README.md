# Registry Cache Utility

This package provides a caching layer for go-orb registry implementations. It can be integrated with any registry implementation to provide local caching of service information.

## Overview

The cache works as a utility for registry implementations, maintaining an in-memory store of services. It automatically manages service data by watching the underlying registry for changes and keeping the cache in sync.

## Usage

Registry implementations can use this cache to:

1. Speed up local lookups
2. Provide fast responses when the primary registry is temporarily unavailable
3. Reduce load on distributed registry systems

### Integration Example

Here's how a registry implementation can integrate with this cache:

```go
package myregistry

import (
	"context"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/plugins/registry/regutil/cache"
)

type Registry struct {
	// ... your registry fields
	cache *cache.Cache
	logger log.Logger
}

func New(logger log.Logger) *Registry {
	r := &Registry{
		// ... initialize your registry
		logger: logger,
	}
	
	cacheConfig := cache.Config{
		TTL: time.Minute * 5, // Configure TTL for cached nodes
	}
	
	// Create the cache with a reference to this registry
	r.cache = cache.New(cacheConfig, logger, r)
	
	return r
}

func (r *Registry) Start(ctx context.Context) error {
	// Start the cache - this will also:
	// 1. Begin TTL pruning
	// 2. Populate the cache with existing services
	// 3. Start watching for changes
	if err := r.cache.Start(ctx); err != nil {
		return err
	}
	
	// ... your registry start logic
	
	return nil
}

func (r *Registry) Stop(ctx context.Context) error {
	// Stop the cache - this will automatically stop watching
	if err := r.cache.Stop(ctx); err != nil {
		return err
	}
	
	// ... your registry stop logic
	
	return nil
}

func (r *Registry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	// Try to get service from cache first
	services, err := r.cache.GetService(name, opts...)
	if err == nil {
		return services, nil
	}
	
	// If not in cache, get from primary source
	services, err = r.getFromPrimarySource(name, opts...)
	if err != nil {
		return nil, err
	}
	
	// Cache is updated automatically via watcher
	
	return services, nil
}

// Other methods simply delegate to the underlying implementation
// The cache will be kept up-to-date automatically via the watcher
```

## Features

- Automatic registry watching with real-time cache updates
- TTL-based node expiration
- Thread-safe operations
- Minimal memory footprint
- Trace/debug logging of cache operations
- Simple, straightforward API

## Methods

The cache provides the following core methods:

- `Start(ctx)`: Starts the cache, populates it, and begins watching the registry
- `Stop(ctx)`: Stops the cache and its watcher
- `GetService(name, ...options)`: Retrieve services by name
- `ListServices(...options)`: List all services

## Usage

The cache is designed to work automatically with minimal configuration:

1. Create a new cache instance with a reference to your registry
2. Start the cache when your registry starts
3. Use the cache's GetService and ListServices methods where appropriate

The cache will handle keeping itself in sync with the registry, automatically updating when services are registered, updated, or deregistered.

## Configuration

The cache accepts a simple configuration with a TTL value:

```go
type Config struct {
	// TTL is the time after which a node is considered stale.
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}