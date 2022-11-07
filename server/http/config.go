package http

import (
	"errors"

	"github.com/go-micro/plugins/server/http/entrypoint"
	"github.com/go-micro/plugins/server/http/router/router"
	"github.com/go-orb/config"
	"github.com/go-orb/config/source"
	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/types"
	"go-micro.dev/v5/util/slice"
)

// Default config values.
//
//nolint:gochecknoglobals
var (
	DefaultAddress        = "0.0.0.0:42069"
	DefaultRouter         = "chi"
	DefaultEnableGzip     = false
	DefaultCodecWhitelist = []string{"proto", "jsonpb", "form", "xml"}

	DefaultConfigSection = "http"
)

// Errors.
var (
	ErrNoRouter            = errors.New("no router plugin name set in config")
	ErrRouterNotFound      = errors.New("router plugin not found, did you register it?")
	ErrEmptyCodecWhitelist = errors.New("codec whitelist is empty")
	ErrNoMatchingCodecs    = errors.New("no matching codecs found, did you register the codec plugins?")
)

// Option is a functional HTTP server option.
type Option func(*Config)

// Entrypoint is a list of entrypoint options used to facilitate the creation
// of one entrypoint, with all options provided. You MUST atleast add the address
// to listen on WithAddress, and can optionally specify more options to use.
// For all options not specified, default values will be used.
type Entrypoint []entrypoint.Option

// Config is the server config. It contains the list of addresses on which
// entrypoints will be created, and the default config used for each entrypoint.
type Config struct {
	// EntrypointDefaults are the defaults used when creating new entrypoints.
	EntrypointDefaults entrypoint.Config

	// Entrypoints is the list of entrypoints that will be created.
	Entrypoints []Entrypoint

	// Router is the router plugin name to use.
	Router string

	// CodecWhitelist is the list of codec names that are allowed to be used
	// with the HTTP server. We explicitly whitelist codecs, as we don't
	// want to add every codec plugin that has been registered to be automaically
	// added to the server.
	//
	// Adding a codec to this list will mean that if the codec has been registered,
	// you will be able to make RPC requests in that format.
	CodecWhitelist []string

	// EnableGzip enables gzip response compression. Only use this if your
	// messages are sufficiently large. For small messages the compute overhead
	// is not worth the reduction in transport time.
	EnableGzip bool
}

// NewConfig creates a new server config with default values.
// To customize the options pas in a list of options.
func NewConfig(serviceName types.ServiceName, data []source.Data, options ...Option) (Config, error) {
	cfg := Config{
		EntrypointDefaults: entrypoint.NewEntrypointConfig(),
		Entrypoints:        make([]Entrypoint, 0, 1),
		Router:             DefaultRouter,
		CodecWhitelist:     DefaultCodecWhitelist,
		EnableGzip:         DefaultEnableGzip,
	}

	cfg.ApplyOptions(options...)

	// TODO: optimize this. extract this part into a separate function? How can we make config easy
	sections := types.SplitServiceName(serviceName)
	if err := config.Parse(append(sections, DefaultConfigSection), data, &cfg); err != nil {
		return cfg, err
	}

	if len(cfg.Entrypoints) == 0 {
		e := Entrypoint{entrypoint.WithAddress(DefaultAddress)}
		cfg.Entrypoints = append(cfg.Entrypoints, e)
	}

	return cfg, nil
}

// ApplyOptions takes a list of options and applies them to the current config.
func (c *Config) ApplyOptions(options ...Option) {
	for _, option := range options {
		option(c)
	}
}

// NewCodecMap fetches the whitelisted codec plugins from the registered codecs
// if present.
func (c *Config) NewCodecMap() (codecs.Map, error) {
	if len(c.CodecWhitelist) == 0 {
		return nil, ErrEmptyCodecWhitelist
	}

	cm := make(codecs.Map, len(c.CodecWhitelist))

	for name, codec := range codecs.Plugins.All() {
		if slice.In(c.CodecWhitelist, name) {
			for _, mime := range codec.ContentTypes() {
				cm[mime] = codec
			}
		}
	}

	if len(cm) == 0 {
		return nil, ErrNoMatchingCodecs
	}

	return cm, nil
}

// NewRouter uses the config.Router to craete a new router.
// It fetches the factory from the registered router plugins.
func (c *Config) NewRouter() (router.Router, error) {
	if len(c.Router) == 0 {
		return nil, ErrNoRouter
	}

	newRouter, err := router.Plugins.Get(c.Router)
	if err != nil {
		return nil, ErrRouterNotFound
	}

	return newRouter(), nil
}

// WithEntrypointOptions takes a list of entrypoint.Option to apply to the
// EntrypointDefaults, to use as defaults when new entrypoints are created.
// See entrypoint.Config for more details about the possible options.
func WithEntrypointOptions(options ...entrypoint.Option) Option {
	return func(c *Config) {
		c.EntrypointDefaults.ApplyOptions(options...)
	}
}

// WithEntrypointDefaults takes a complete entrypoint.Config to use as default
// when creating new entrypoints. Only use this if you want to specify (almost)
// all values in the entrypoint.Config. Otherwise use WithEntrypointOptions to
// specify a list of options.
func WithEntrypointDefaults(defaults entrypoint.Config) Option {
	return func(c *Config) {
		c.EntrypointDefaults = defaults
	}
}

// WithAddress takes a list of addresses to listen on.
// This is an alias for WithEntrypoints, specifying only the address.
//
// It will create an Entrypoint for each address, attaching an HTTP server on each.
// Each entrypoint will be created with the Config.DefaultEntryPointConfig.
// If you want to change the defaults used for the creation of each entrypoint
// specify them WithEntrypointOptions, if you want to create custom entrypoints
// with specific non-default options, use WithEntrypoints.
//
// To listen on all interfaces on one specific port use the ":<port>" notation.
// To listen on a specific interface use the "<IP>:<port>" notation.
func WithAddress(addrs ...string) Option {
	return func(c *Config) {
		for _, addr := range addrs {
			e := Entrypoint{entrypoint.WithAddress(addr)}
			c.Entrypoints = append(c.Entrypoints, e)
		}
	}
}

// WithEntrypoints is used to provide a list of entrypoints to create, will
// custom options only applied to each entypoint individually.
// You MUST atleast provide an address to listen on WithAddress.
//
// Example:
//
//	WithEntrypoints([]http.Entrypoint{
//	 {entrypoint.WithAddress(":8080")},
//	 {entrypoint.WithAddress("192.168.1.50:8081"), WithHTTP3()},
//	 {entrypoint.WithAddress(":8082"), WithInsecure(), WithWriteTimeout(...)},
//	})
func WithEntrypoints(entrypoints ...Entrypoint) Option {
	return func(c *Config) {
		c.Entrypoints = append(c.Entrypoints, entrypoints...)
	}
}

// WithInsecure starts the servers without TLS certificate, use this if you
// want to server HTTP traffick without encryption.
func WithInsecure() Option {
	return WithEntrypointOptions(entrypoint.WithInsecure())
}

// WithHTTP3 enabled the HTTP3 sever by default on the creation of new entrypoints.
func WithHTTP3() Option {
	return WithEntrypointOptions(entrypoint.WithHTTP3())
}

// WithGzip enables gzip compression on the servers. Only use this if your
// messages are sufficiently large. For small messages the compute overhead
// is not worth the reduction in transport time.
func WithGzip() Option {
	return func(c *Config) {
		c.EnableGzip = true
	}
}

// WithRouter sets the default router plugin to use.
// Make sure to import your router plugin.
func WithRouter(name string) Option {
	return func(c *Config) {
		c.Router = name
	}
}

// WithDisableHTTP2 disables HTTP2 by default on all newly created entrypoints.
func WithDisableHTTP2() Option {
	return WithEntrypointOptions(entrypoint.DisableHTTP2())
}

// WithAllowH2C enables insecure HTTP2 support; HTTP/2 without TLS.
// This is thus also only possible if you set WithInsecure.
func WithAllowH2C() Option {
	return WithEntrypointOptions(entrypoint.AllowH2C())
}
