package kvstore

import (
	"time"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/registry"
)

func init() {
	registry.Plugins.Add(Name, Provide)
}

//nolint:gochecknoglobals
var (
	// DefaultServiceDelimiter is the default delimiter used to separate service name and version.
	DefaultServiceDelimiter = "@"

	// DefaultDatabase is the default database name.
	DefaultDatabase = "service-registry"

	// DefaultTable is the default table name.
	DefaultTable = "service-registry"

	// DefaultTTL is the default time after which a node is considered stale.
	DefaultTTL = 10 * time.Millisecond

	// DefaultCache enables caching.
	DefaultCache = true
)

// Name provides the name of this registry.
const Name = "kvstore"

// Config provides configuration for the memory registry.
type Config struct {
	registry.Config `yaml:",inline"`

	// ServiceDelimiter is the delimiter used to separate service name and version.
	ServiceDelimiter string `json:"serviceDelimiter" yaml:"serviceDelimiter"`

	// TTL is the time after which a node is considered stale.
	TTL config.Duration `json:"ttl" yaml:"ttl"`

	// Database is the database name in the kvstore.
	Database string `json:"database" yaml:"database"`

	// Table is the table name in the kvstore.
	Table string `json:"table" yaml:"table"`

	// Cache enables/disables caching.
	Cache bool `json:"cache" yaml:"cache"`
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
		Config:           registry.NewConfig(),
		ServiceDelimiter: DefaultServiceDelimiter,
		TTL:              config.Duration(DefaultTTL),
		Database:         DefaultDatabase,
		Table:            DefaultTable,
		Cache:            DefaultCache,
	}

	cfg.ApplyOptions(opts...)

	return cfg
}

// WithServiceDelimiter sets the service delimiter.
func WithServiceDelimiter(n string) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.ServiceDelimiter = n
		}
	}
}

// WithTTL sets the TTL.
func WithTTL(n time.Duration) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.TTL = config.Duration(n)
		}
	}
}

// WithDatabase sets the database name.
func WithDatabase(n string) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Database = n
		}
	}
}

// WithTable sets the table name.
func WithTable(n string) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Table = n
		}
	}
}

// WithNoCache disables caching.
func WithNoCache() registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Cache = false
		}
	}
}
