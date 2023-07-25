package consul

import (
	"crypto/tls"
	"fmt"
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source/cli"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
)

// metaTransportKey is the key to use to store the scheme in metadata.
const metaTransportKey = "_md_scheme_"

// Name provides the name of this registry.
const Name = "consul"

// Defaults.
//
//nolint:gochecknoglobals
var (
	DefaultAddresses  = []string{"localhost:8500"}
	DefaultAllowStale = true
)

func init() {
	//nolint:errcheck
	_ = cli.Flags.Add(cli.NewFlag(
		"registry_addresses",
		DefaultAddresses,
		cli.ConfigPathSlice([]string{"registry", "addresses"}),
		cli.Usage("Registry addresses."),
	))

	registry.Plugins.Register(Name, ProvideRegistryConsul)
}

// Config provides configuration for the consul registry.
type Config struct {
	registry.Config `yaml:",inline"`

	Addresses []string    `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	Secure    bool        `json:"secure,omitempty" yaml:"secure,omitempty"`
	TLSConfig *tls.Config `json:"-" yaml:"-"`

	Connect bool `json:"connect,omitempty" yaml:"connect,omitempty"`

	ConsulConfig *consul.Config       `json:"-" yaml:"-"`
	AllowStale   bool                 `json:"allowStale,omitempty" yaml:"allowStale,omitempty"`
	QueryOptions *consul.QueryOptions `json:"-" yaml:"-"`
	TCPCheck     time.Duration        `json:"tcpCheck,omitempty" yaml:"tcpCheck,omitempty"`
}

// NewConfig creates a new config object.
func NewConfig(
	serviceName types.ServiceName,
	datas types.ConfigData,
	opts ...registry.Option,
) (Config, error) {
	cfg := Config{
		Config:     registry.NewConfig(),
		AllowStale: DefaultAllowStale,
	}

	cfg.ApplyOptions(opts...)

	sections := types.SplitServiceName(serviceName)
	if err := config.Parse(append(sections, registry.ComponentType), datas, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
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
		} else {
			panic(fmt.Sprintf("wrong type: %T", c))
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
