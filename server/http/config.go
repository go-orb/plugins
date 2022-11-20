package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/util/slicemap"
	"golang.org/x/exp/slog"

	"github.com/go-micro/plugins/server/http/router/router"
)

//nolint:gochecknoglobals
var (
	// TODO: revisit default address, probably use random addr
	// DefaultAddress to use for new HTTP servers.
	// If set to "random", the default, a random address will be selected,
	// preferably on a private interface (XX subet). TODO: implement.
	DefaultAddress = "0.0.0.0:42069"
	// DefaultInsecure will create an HTTP server without TLS, for insecure connections.
	// Note: as a result you can only make insecure HTTP requests, and no HTTP2
	// unless you set WithH2C.
	//
	// WARNING: don't use this in production, unless you really know what you are doing.
	// this will result in unencrypted HTTP traffick. Really, it is even advised
	// against using this in testing.
	DefaultInsecure = false
	// DefaultAllowH2C allows insecure, unencrypted traffick to HTTP2 servers.
	// Don't use this, see the notes at DefaultInsecure for more details.
	DefaultAllowH2C = false
	// DefaultMaxConcurrentStreams for HTTP2.
	DefaultMaxConcurrentStreams = 250
	// HTTP2 dicates whether to also allow HTTP/2 connections.
	DefaultHTTP2 = true
	// HTTP3 dicates whether to also start an HTTP/3.0 server.
	DefaultHTTP3 = false
	// DefaultRouter to use as serve mux. There's not really a reason to change this
	// but if you really wanted to, you could.
	DefaultRouter = "chi"
	// DefaultReadTimeout, see net/http pkg for more details.
	DefaultReadTimeout = 5 * time.Second
	// DefaultWriteTimeout, see net/http pkg for more details.
	DefaultWriteTimeout = 5 * time.Second
	// DefaultIdleimeout, see net/http pkg for more details.
	DefaultIdleimeout = 5 * time.Second
	// DefaultEnableGzip enables gzip response compression server wide onall responses.
	// Only use this if your messages are sufficiently large. For small messages
	// the compute overhead is not worth the reduction in transport time.
	//
	// Alternatively, you can send a gzip compressed request, and the server
	// will send back a gzip compressed respponse.
	DefaultEnableGzip = false
	// DefaultCodecWhitelist is the default allowed list of codecs to be used for
	// HTTP request encoding/decoding. This means that if any of these plugins are
	// registered, they will be included in the server's available codecs.
	// If they are not registered, the server will not be able to handle these formats.
	DefaultCodecWhitelist = []string{"proto", "jsonpb", "form", "xml"}
	// DefaultConfigSection is the section key used in config files used to
	// configure the server options.
	DefaultConfigSection = "http"
)

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
	Name string `json:"name" yaml:"name"`
	// Address to listen on.
	Address string `json:"address" yaml:"address"`
	// Insecure will create an HTTP server without TLS, for insecure connections.
	// Note: as a result you can only make insecure HTTP1 requests, no HTTP2
	// unless you set WithH2C.
	//
	// WARNING: don't use this in production, unless you really know what you are doing.
	// this will result in unencrypted HTTP traffick. Really, it is even advised
	// against using this in testing.
	Insecure bool `json:"insecure" yaml:"insecure"`
	// MaxConcurrentStreams for HTTP2.
	MaxConcurrentStreams int `json:"maxConcurrentStreams" yaml:"maxConcurrentStreams"`
	// TLS config, if none is provided a self-signed certificates will be generated.
	TLS *tls.Config // TODO: how do add certs from config? add back cerfile/keyfile?
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
	// RegistrationFuncs are all handler registration functions that will be
	// registered to the server upon startup. You can statically add handlers
	// By using the fuctional server options. Optionally, you can dynamically
	// add handlers by registering them to the Handlers global, and setting them
	// explicitly in the config.
	RegistrationFuncs server.HandlerRegistrations `json:"handlers" yaml:"handlers"`
	// Middleware is a list of middleware to use.
	Middleware router.Middlewares `json:"middleware" yaml:"middleware"`
	// Logger allows you to dynamically change the log level and plugin for a
	// specific entrypoint.
	Logger struct {
		Level  slog.Level `json:"level,omitempty" yaml:"level,omitempty"` // TODO: change with custom level
		Plugin string     `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	} `json:"logger" yaml:"logger"`
}

// NewConfig will create a new default config for the entrypoint.
func NewConfig(options ...Option) *Config {
	cfg := Config{
		Name:                 "http-" + uuid.NewString(),
		Address:              DefaultAddress,
		Insecure:             DefaultInsecure,
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		H2C:                  DefaultAllowH2C,
		HTTP2:                DefaultHTTP2,
		HTTP3:                DefaultHTTP3,
		Gzip:                 DefaultEnableGzip,
		CodecWhitelist:       DefaultCodecWhitelist,
		Router:               DefaultRouter,
		ReadTimeout:          DefaultReadTimeout,
		WriteTimeout:         DefaultWriteTimeout,
		IdleTimeout:          DefaultIdleimeout,
		RegistrationFuncs:    make(server.HandlerRegistrations),
		Middleware:           make(router.Middlewares),
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

	for name, codec := range codecs.Plugins.All() {
		if slicemap.In(c.CodecWhitelist, name) {
			// One codec can support multiple mime types, we add all of them to the map.
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

// WithName sets the entrypoint name. The default name is in the format of
// http-<uuid>.
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

// WithTLSConfig sets a tls config.
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.TLS = tlsConfig
	}
}

// WithInsecure will create the entrypoint without using TLS.
// Note: as a result you can only make insecure HTTP requests, and no HTTP2
// unless you set WithH2C.
//
// WARNING: don't use this in production, unless you really know what you are doing.
// this will result in unencrypted HTTP traffick. Really, it is even advised
// against using this in testing.
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

// WithDefaults sets default options to use on the creattion of new HTTP entrypoints.
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

// WithConfig will set replace the server config with config provided as argument.
// Warning: any options applied previous to this option will be overwritten by
// the contents of the config provided here.
func WithConfig(config Config) Option {
	return func(c *Config) {
		*c = config
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

// WithRegistration adds a named registration function to the config.
// The name set here allows you to dynamically add this handler to entrypoints
// through a config.
//
// Registration functions are used to register handlers to a server.
func WithRegistration(name string, registration server.RegistrationFunc) Option {
	server.Handlers.Register(name, registration)

	return func(c *Config) {
		c.RegistrationFuncs[name] = registration
	}
}

// WithMiddleware appends middlewares to the server.
// You can use any standard Go HTTP middleware.
//
// Each middlware is uniquely identified with a name. The name provided here
// can be used to dynamically add middlware to an entrypoint in a config.
func WithMiddleware(name string, middleware func(http.Handler) http.Handler) Option {
	router.Middleware.Register(name, middleware)

	return func(c *Config) {
		c.Middleware[name] = middleware
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

// TODO: other options and option comments
