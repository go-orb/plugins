package memory

import (
	"time"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/registry"
)

func init() {
	registry.Plugins.Add(Name, Provide)
}

var (
	// DefaultWatcherSendTimeout is the default timeout for sending events to watchers.
	//nolint:gochecknoglobals
	DefaultWatcherSendTimeout = 10 * time.Millisecond

	// DefaultTTL is the default time after which a node is considered stale.
	//nolint:gochecknoglobals
	DefaultTTL = 10 * time.Millisecond
)

// Name provides the name of this registry.
const Name = "memory"

// Config provides configuration for the memory registry.
type Config struct {
	registry.Config `yaml:",inline"`

	// WatcherSendTimeout is the timeout for sending events to watchers.
	WatcherSendTimeout config.Duration `json:"watcherSendTimeout" yaml:"watcherSendTimeout"`
	// TTL is the time after which a node is considered stale.
	TTL config.Duration `json:"ttl" yaml:"ttl"`
}

// ApplyOptions applies a set of options to the config.
func (c *Config) ApplyOptions(opts ...registry.Option) {
	for _, o := range opts {
		o(c)
	}
}

// NewConfig creates a new config object.
func NewConfig(opts ...registry.Option) Config {
	cfg := Config{
		Config: registry.NewConfig(),
	}
	cfg.WatcherSendTimeout = config.Duration(DefaultWatcherSendTimeout)
	cfg.TTL = config.Duration(DefaultTTL)

	cfg.ApplyOptions(opts...)

	return cfg
}
