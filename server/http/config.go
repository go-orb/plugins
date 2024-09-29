package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/util/slicemap"
	mtls "github.com/go-orb/go-orb/util/tls"
	"github.com/google/uuid"

	"github.com/go-orb/plugins/server/http/router"
)

const (
	// DefaultAddress to use for new HTTP servers.
	// If set to "random", the default, a random address will be selected,
	// preferably on a private interface (XX subet). TODO: implement.
	// TODO(davincible): revisit default address, probably use random addr.
	DefaultAddress = "0.0.0.0:42069"

	// DefaultInsecure will create an HTTP server without TLS, for insecure connections.
	// Note: as a result you can only make insecure HTTP requests, and no HTTP2
	// unless you set WithH2C.
	//
	// WARNING: don't use this in production, unless you really know what you are
	// doing. this will result in unencrypted traffic. Really, it is even advised
	// against using this in testing environments.
	DefaultInsecure = false

	// DefaultAllowH2C allows insecure, unencrypted traffic to HTTP2 servers.
	// Don't use this, see the notes at DefaultInsecure for more details.
	DefaultAllowH2C = false

	// DefaultMaxConcurrentStreams for HTTP2.
	DefaultMaxConcurrentStreams = 512

	// DefaultHTTP2 dicates whether to also allow HTTP/2 and HTTP/3 connections.
	DefaultHTTP2 = true

	// DefaultHTTP3 dicates whether to also start an HTTP/3.0 server.
	DefaultHTTP3 = false

	// DefaultRouter to use as serve mux. There's not really a reason to change this
	// but if you really wanted to, you could.
	DefaultRouter = "chi"

	// DefaultReadTimeout see net/http pkg for more details.
	DefaultReadTimeout = 5 * time.Second

	// DefaultWriteTimeout see net/http pkg for more details.
	DefaultWriteTimeout = 5 * time.Second

	// DefaultIdleTimeout see net/http pkg for more details.
	DefaultIdleTimeout = 5 * time.Second

	// DefaultEnableGzip enables gzip response compression server wide onall responses.
	// Only use this if your messages are sufficiently large. For small messages
	// the compute overhead is not worth the reduction in transport time.
	//
	// Alternatively, you can send a gzip compressed request, and the server
	// will send back a gzip compressed respponse.
	DefaultEnableGzip = false

	// DefaultConfigSection is the section key used in config files used to
	// configure the server options.
	DefaultConfigSection = Plugin

	// DefaultMaxHeaderBytes is the maximum size to parse from a client's
	// HTTP request headers.
	DefaultMaxHeaderBytes = 1024 * 64
)

// DefaultCodecWhitelist is the default allowed list of codecs to be used for
// HTTP request encoding/decoding. This means that if any of these plugins are
// registered, they will be included in the server's available codecs.
// If they are not registered, the server will not be able to handle these formats.
func DefaultCodecWhitelist() []string {
	return []string{"proto", "jsonpb", "form", "xml"}
}

// Errors.
var (
	ErrNoRouter            = errors.New("no router plugin name set in config")
	ErrRouterNotFound      = errors.New("router plugin not found, did you register it?")
	ErrEmptyCodecWhitelist = errors.New("codec whitelist is empty")
	ErrNoMatchingCodecs    = errors.New("no matching codecs found, did you register the codec plugins?")
)

// Option is a functional option to provide custom values to the config.
type Option func(*Config)

// Config provides options to the entrypoint.
type Config struct {
	// Name is the entrypoint name.
	//
	// The default name is 'http-<random uuid>'
	Name string `json:"name" yaml:"name"`

	// Address to listen on.
	// TODO(davincible): implement this, and the address method.
	// If no IP is provided, an interface will be selected automatically. Private
	// interfaces are preferred, if none are found a public interface will be used.
	//
	// If no port is provided, a random port will be selected. To listen on a
	// specific interface, but with a random port, you can use '<IP>:0'.
	Address string `json:"address" yaml:"address"`

	// Insecure will create an HTTP server without TLS, for insecure connections.
	// Note: as a result you can only make insecure HTTP1 requests, no HTTP2
	// unless you set WithH2C.
	//
	// WARNING: don't use this in production, unless you really know what you are
	// doing. this will result in unencrypted traffic. Really, it is even advised
	// against using this in testing environments.
	Insecure bool `json:"insecure" yaml:"insecure"`

	// TLS config, if none is provided a self-signed certificates will be generated.
	//
	// You can load a tls config from yaml/json with the following options:
	//
	// ```yaml
	// rootCAFiles:
	//    - xxx
	// clientCAFiles:
	//    - xxx
	// clientAuth: "none" | "request" | "require" |  "verify" | "require+verify"
	// certificates:
	//   - certFile: xxx
	//     keyFile: xxx
	// ```
	TLS *mtls.Config `json:"tls,omitempty" yaml:"tls,omitempty"`

	// H2C allows h2c connections; HTTP2 without TLS.
	H2C bool `json:"h2c" yaml:"h2c"`

	// HTTP2 dicates whether to also allow HTTP/2 connections. Defaults to true.
	HTTP2 bool `json:"http2" yaml:"http2"`

	// HTTP3 dicates whether to also start an HTTP/3.0 server. Defaults to false.
	HTTP3 bool `json:"http3" yaml:"http3"`

	// Gzip enables gzip response compression server wide onall responses.
	// Only use this if your messages are sufficiently large. For small messages
	// the compute overhead is not worth the reduction in transport time.
	//
	// Alternatively, you can send a gzip compressed request, and the server
	// will send back a gzip compressed respponse.
	Gzip bool `json:"gzip" yaml:"gzip"`

	// MaxConcurrentStreams for HTTP2.
	MaxConcurrentStreams int `json:"maxConcurrentStreams" yaml:"maxConcurrentStreams"`

	// MaxHeaderBytes is the maximum size to parse from a client's
	// HTTP request headers.
	MaxHeaderBytes int `json:"maxHeaderBytes" yaml:"maxHeaderBytes"`

	// CodecWhitelist is the list of codec names that are allowed to be used
	// with the HTTP server. This means that if registered, codecs in this list
	// will be added to the server, allowing you to make RPC requests in that format.
	// If any of the codecs in this list are not registred nothing will happen.
	//
	// We explicitly whitelist codecs, as we don't
	// want to add every codec plugin that has been registered to be automaically
	// added to the server.
	CodecWhitelist []string `json:"codecWhitelist" yaml:"codecWhitelist"`

	// Router is the router plugin to use. Default is chi.
	Router string `json:"router" yaml:"router"`

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration `json:"idleTimeout" yaml:"idleTimeout"`

	// HandlerRegistrations are all handler registration functions that will be
	// registered to the server upon startup.
	//
	// You can statically add handlers by using the fuctional server options.
	// Optionally, you can dynamically add handlers by registering them to the
	// Handlers global, and setting them explicitly in the config.
	HandlerRegistrations server.HandlerRegistrations `json:"handlers" yaml:"handlers"`

	// Middlewares is a list of middleware to use.
	Middlewares []string `json:"middlewares" yaml:"middlewares"`

	// Logger allows you to dynamically change the log level and plugin for a
	// specific entrypoint.
	Logger log.Config `json:"logger" yaml:"logger"`
}

// NewConfig will create a new default config for the entrypoint.
func NewConfig(options ...Option) *Config {
	cfg := Config{
		Name:                 "http-" + uuid.NewString(),
		Address:              DefaultAddress,
		Insecure:             DefaultInsecure,
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		MaxHeaderBytes:       DefaultMaxHeaderBytes,
		H2C:                  DefaultAllowH2C,
		HTTP2:                DefaultHTTP2,
		HTTP3:                DefaultHTTP3,
		Gzip:                 DefaultEnableGzip,
		CodecWhitelist:       DefaultCodecWhitelist(),
		Router:               DefaultRouter,
		ReadTimeout:          DefaultReadTimeout,
		WriteTimeout:         DefaultWriteTimeout,
		IdleTimeout:          DefaultIdleTimeout,
		HandlerRegistrations: make(server.HandlerRegistrations),
		Middlewares:          []string{},
	}

	cfg.ApplyOptions(options...)

	return &cfg
}

// GetAddress returns the entrypoint address.
func (c Config) GetAddress() string {
	return c.Address
}

// Copy creates a copy of the entrypoint config.
func (c Config) Copy() server.EntrypointConfig {
	return &c
}

// ApplyOptions applies a set of options to the config.
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

	codecs.Plugins.Range(func(name string, codec codecs.Marshaler) bool {
		if slicemap.In(c.CodecWhitelist, name) {
			// One codec can support multiple mime types, we add all of them to the map.
			for _, mime := range codec.ContentTypes() {
				cm[mime] = codec
			}
		}

		return true
	})

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

	newRouter, ok := router.Plugins.Get(c.Router)
	if !ok {
		return nil, ErrRouterNotFound
	}

	return newRouter(), nil
}

// WithName sets the entrypoint name. The default name is in the format of
// 'http-<uuid>'.
//
// Setting a custom name allows you to dynamically reference the entrypoint in
// the file config, and makes it easier to attribute the logs.
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithAddress specifies the address to listen on.
// If you want to listen on all interfaces use the format ":8080"
// If you want to listen on a specific interface/address use the full IP.
func WithAddress(address string) Option {
	return func(c *Config) {
		c.Address = address
	}
}

// WithTLS sets a tls config.
func WithTLS(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.TLS = &mtls.Config{Config: tlsConfig}
	}
}

// WithInsecure will create the entrypoint without using TLS.
// Note: as a result you can only make insecure HTTP requests, and no HTTP2
// unless you set WithH2C.
//
// WARNING: don't use this in production, unless you really know what you are
// doing. this will result in unencrypted traffic. Really, it is even advised
// against using this in testing environments.
func WithInsecure() Option {
	return func(c *Config) {
		c.Insecure = true
	}
}

// WithHTTP3 will additionally enable an HTTP3 server on the entrypoint.
func WithHTTP3() Option {
	return func(c *Config) {
		c.HTTP3 = true
	}
}

// WithDisableHTTP2 will prevent the creation of an HTTP2 server on the entrypoint.
func WithDisableHTTP2() Option {
	return func(c *Config) {
		c.HTTP2 = false
	}
}

// WithGzip enables gzip response compression server wide onall responses.
// Only use this if your messages are sufficiently large. For small messages
// the compute overhead is not worth the reduction in transport time.
//
// Alternatively, you can send a gzip compressed request, and the server
// will send back a gzip compressed respponse.
func WithGzip() Option {
	return func(c *Config) {
		c.Gzip = true
	}
}

// WithAllowH2C will allow H2C connections on the entrypoint. H2C is HTTP2 without TLS.
// It is not recommended to turn this on.
func WithAllowH2C() Option {
	return func(c *Config) {
		c.H2C = true
	}
}

// WithConfig will set replace the server config with config provided as argument.
// Warning: any options applied previous to this option will be overwritten by
// the contents of the config provided here.
func WithConfig(config Config) Option {
	return func(c *Config) {
		*c = config
	}
}

// WithMaxConcurrentStreams sets the concurrent streams limit for HTTP2.
func WithMaxConcurrentStreams(value int) Option {
	return func(c *Config) {
		c.MaxConcurrentStreams = value
	}
}

// WithRouter sets the router plguin name.
func WithRouter(router string) Option {
	return func(c *Config) {
		c.Router = router
	}
}

// WithCodecWhitelist sets the list of codecs allowed in the HTTP entrypoint.
// If registered, any codecs set here will be imported into the server.
// You still need to register the codec plugins by importing them.
func WithCodecWhitelist(list []string) Option {
	return func(c *Config) {
		c.CodecWhitelist = list
	}
}

// WithReadTimeout sets the maximum duration for reading the entire request,
// including the body. A zero or negative value means there will be no timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ReadTimeout = timeout
	}
}

// WithWriteTimeout sets the maximum duration before timing out writes of the
// response. It is reset whenever a new request's header is read. Like
// ReadTimeout, it does not let Handlers make decisions on a per-request basis.
// A zero or negative value means there will be no timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.WriteTimeout = timeout
	}
}

// WithIdleTimeout is the maximum amount of time to wait for the next request when
// keep-alives are enabled. If IdleTimeout is zero, the value of ReadTimeout is
// used. If both are zero, there is no timeout.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.IdleTimeout = timeout
	}
}

// WithRegistration adds a named registration function to the config.
// The name set here allows you to dynamically add this handler to entrypoints
// through a config.
//
// Registration functions are used to register handlers to a server.
func WithRegistration(name string, registration server.RegistrationFunc) Option {
	server.Handlers.Set(name, registration)

	return func(c *Config) {
		c.HandlerRegistrations[name] = registration
	}
}

// WithMiddleware appends middlewares to the server.
// You can use any standard Go HTTP middleware.
//
// Each middlware is uniquely identified with a name. The name provided here
// can be used to dynamically add middlware to an entrypoint in a config.
func WithMiddleware(middlewares ...string) Option {
	return func(c *Config) {
		c.Middlewares = append(c.Middlewares, middlewares...)
	}
}

// WithLogLevel changes the log level from the inherited logger.
func WithLogLevel(level string) Option {
	return func(c *Config) {
		c.Logger.Level = level
	}
}

// WithLogPlugin changes the log level from the inherited logger.
func WithLogPlugin(plugin string) Option {
	return func(c *Config) {
		c.Logger.Plugin = plugin
	}
}

// WithDefaults sets default options to use on the creation of new HTTP entrypoints.
func WithDefaults(options ...Option) server.Option {
	return func(c *server.Config) {
		cfg, ok := c.Defaults[Plugin].(*Config)
		if !ok {
			// Should never happen.
			panic(fmt.Errorf("http.WithDefaults received invalid type, not *server.Config, but '%T'", cfg))
		}

		cfg.ApplyOptions(options...)

		c.Defaults[Plugin] = cfg
	}
}

// WithEntrypoint adds an HTTP entrypoint with the provided options.
func WithEntrypoint(options ...Option) server.Option {
	return func(c *server.Config) {
		cfgAny, ok := c.Defaults[Plugin]
		if !ok {
			// Should never happen, but just in case.
			panic("no defaults for http entrypoint found")
		}

		cfg := cfgAny.Copy().(*Config) //nolint:errcheck

		cfg.ApplyOptions(options...)

		c.Templates[cfg.Name] = server.EntrypointTemplate{
			Enabled: true,
			Type:    Plugin,
			Config:  cfg,
		}
	}
}
