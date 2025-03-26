package consul

import (
	"crypto/tls"
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/go-orb/go-orb/registry"
)

const metaPrefix = "orb_app_"
const myMetaPrefix = "orb_internal_"

// Name provides the name of this registry.
const Name = "consul"

// Defaults.
//
//nolint:gochecknoglobals
var (
	DefaultAddresses  = []string{"localhost:8500"}
	DefaultAllowStale = true

	// DefaultCache enables caching.
	DefaultCache = true
)

func init() {
	registry.Plugins.Add(Name, Provide)
}

// Config provides configuration for the consul registry.
type Config struct {
	registry.Config `yaml:",inline"`

	Addresses []string    `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	Secure    bool        `json:"secure,omitempty"    yaml:"secure,omitempty"`
	TLSConfig *tls.Config `json:"-"                   yaml:"-"`

	Connect bool `json:"connect,omitempty" yaml:"connect,omitempty"`

	ConsulConfig *consul.Config       `json:"-"                    yaml:"-"`
	AllowStale   bool                 `json:"allowStale,omitempty" yaml:"allowStale,omitempty"`
	QueryOptions *consul.QueryOptions `json:"-"                    yaml:"-"`
	TCPCheck     time.Duration        `json:"tcpCheck,omitempty"   yaml:"tcpCheck,omitempty"`

	// Cache enables/disables caching.
	Cache bool `json:"cache,omitempty" yaml:"cache,omitempty"`
}

// NewConfig creates a new config object.
func NewConfig(
	opts ...registry.Option,
) Config {
	cfg := Config{
		Config:     registry.NewConfig(),
		AllowStale: DefaultAllowStale,
		Cache:      DefaultCache,
	}

	cfg.ApplyOptions(opts...)

	return cfg
}

// ApplyOptions applies a set of options to the config.
func (c *Config) ApplyOptions(opts ...registry.Option) {
	for _, o := range opts {
		o(c)
	}
}

// WithAddress sets the Consul server addresses.
func WithAddress(n ...string) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Addresses = n
		}
	}
}

// WithSecure defines if we want a secure connection to Consul.
func WithSecure(n bool) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Secure = n
		}
	}
}

// WithTLSConfig defines the TLS config to use for the secure connection.
func WithTLSConfig(n *tls.Config) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.TLSConfig = n
		}
	}
}

// WithConnect defines if services should be registered as Consul Connect services.
func WithConnect(n bool) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Connect = n
		}
	}
}

// WithConsulConfig defines the consul config.
func WithConsulConfig(n *consul.Config) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.ConsulConfig = n
		}
	}
}

// WithAllowStale sets whether any Consul server (non-leader) can service
// a read. This allows for lower latency and higher throughput
// at the cost of potentially stale data.
// Works similar to Consul DNS Config option [1].
// Defaults to true.
//
// [1] https://www.consul.io/docs/agent/options.html#allow_stale
func WithAllowStale(n bool) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.AllowStale = n
		}
	}
}

// WithQueryOptions specifies the QueryOptions to be used when calling
// Consul. See `Consul API` for more information [1].
//
// [1] https://godoc.org/github.com/hashicorp/consul/api#QueryOptions
func WithQueryOptions(n *consul.QueryOptions) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.QueryOptions = n
		}
	}
}

// WithTCPCheck will tell the service provider to check the service address
// and port every `t` interval. It will enabled only if `t` is greater than 0.
// See `TCP + Interval` for more information [1].
//
// [1] https://www.consul.io/docs/agent/checks.html
func WithTCPCheck(t time.Duration) registry.Option {
	return func(c registry.ConfigType) {
		if t <= time.Duration(0) {
			return
		}

		cfg, ok := c.(*Config)
		if ok {
			cfg.TCPCheck = t
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
