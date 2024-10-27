package hertz

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	mtls "github.com/go-orb/go-orb/util/tls"
	"github.com/google/uuid"
)

const (
	// DefaultAddress to use for new Hertz servers.
	DefaultAddress = ":0"

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

	// DefaultHTTP2 dicates whether to also allow HTTP/2 connections.
	DefaultHTTP2 = true

	// DefaultReadTimeout see net/http pkg for more details.
	DefaultReadTimeout = 5 * time.Second

	// DefaultWriteTimeout see net/http pkg for more details.
	DefaultWriteTimeout = 5 * time.Second

	// DefaultIdleTimeout see net/http pkg for more details.
	DefaultIdleTimeout = 5 * time.Second

	// DefaultStopTimeout sets the timeout for ServerHertz.Stop().
	DefaultStopTimeout = time.Second

	// DefaultConfigSection is the section key used in config files used to
	// configure the server options.
	DefaultConfigSection = Plugin

	// DefaultMaxHeaderBytes is the maximum size to parse from a client's
	// HTTP request headers.
	DefaultMaxHeaderBytes = 1024 * 64
)

// Errors.
var (
	ErrNoRouter         = errors.New("no router plugin name set in config")
	ErrRouterNotFound   = errors.New("router plugin not found, did you register it?")
	ErrNoMatchingCodecs = errors.New("no matching codecs found, did you register the codec plugins?")
)

// Config provides options to the entrypoint.
type Config struct {
	server.EntrypointConfig `yaml:",inline"`

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

	// MaxConcurrentStreams for HTTP2.
	MaxConcurrentStreams int `json:"maxConcurrentStreams" yaml:"maxConcurrentStreams"`

	// MaxHeaderBytes is the maximum size to parse from a client's
	// HTTP request headers.
	MaxHeaderBytes int `json:"maxHeaderBytes" yaml:"maxHeaderBytes"`

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

	// StopTimeout is the timeout for ServerHertz.Stop().
	StopTimeout time.Duration `json:"stopTimeout" yaml:"stopTimeout"`

	// Middlewares is a list of middleware to use.
	Middlewares []server.MiddlewareConfig `json:"middlewares" yaml:"middlewares"`

	// Handlers is a list of pre-registered handlers.
	Handlers []string `json:"handlers" yaml:"handlers"`

	// Logger allows you to dynamically change the log level and plugin for a
	// specific entrypoint.
	Logger log.Config `json:"logger" yaml:"logger"`
}

// NewConfig will create a new default config for the entrypoint.
func NewConfig(options ...server.Option) *Config {
	cfg := &Config{
		EntrypointConfig: server.EntrypointConfig{
			Name:    Plugin + "-" + uuid.NewString(),
			Plugin:  Plugin,
			Enabled: true,
		},
		Address:              DefaultAddress,
		Insecure:             DefaultInsecure,
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		MaxHeaderBytes:       DefaultMaxHeaderBytes,
		H2C:                  DefaultAllowH2C,
		HTTP2:                DefaultHTTP2,
		ReadTimeout:          DefaultReadTimeout,
		WriteTimeout:         DefaultWriteTimeout,
		IdleTimeout:          DefaultIdleTimeout,
		StopTimeout:          DefaultStopTimeout,
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
}

// WithName sets the entrypoint name. The default name is in the format of
// 'http-<uuid>'.
//
// Setting a custom name allows you to dynamically reference the entrypoint in
// the file config, and makes it easier to attribute the logs.
func WithName(name string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Name = name
		}
	}
}

// WithAddress specifies the address to listen on.
// If you want to listen on all interfaces use the format ":8080"
// If you want to listen on a specific interface/address use the full IP.
func WithAddress(addr string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Address = addr
		}
	}
}

// WithTLS sets a tls config.
func WithTLS(config *tls.Config) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.TLS = &mtls.Config{Config: config}
		}
	}
}

// WithInsecure will create the entrypoint without using TLS.
// Note: as a result you can only make insecure HTTP requests, and no HTTP2
// unless you set WithH2C.
//
// WARNING: don't use this in production, unless you really know what you are
// doing. this will result in unencrypted traffic. Really, it is even advised
// against using this in testing environments.
func WithInsecure() server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Insecure = true
		}
	}
}

// WithDisableHTTP2 will prevent the creation of an HTTP2 server on the entrypoint.
func WithDisableHTTP2() server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.HTTP2 = false
		}
	}
}

// WithAllowH2C will allow H2C connections on the entrypoint. H2C is HTTP2 without TLS.
// It is not recommended to turn this on.
func WithAllowH2C() server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.H2C = true
		}
	}
}

// WithMaxConcurrentStreams sets the concurrent streams limit for HTTP2.
func WithMaxConcurrentStreams(value int) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.MaxConcurrentStreams = value
		}
	}
}

// WithReadTimeout sets the maximum duration for reading the entire request,
// including the body. A zero or negative value means there will be no timeout.
func WithReadTimeout(timeout time.Duration) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.ReadTimeout = timeout
		}
	}
}

// WithWriteTimeout sets the maximum duration before timing out writes of the
// response. It is reset whenever a new request's header is read. Like
// ReadTimeout, it does not let Handlers make decisions on a per-request basis.
// A zero or negative value means there will be no timeout.
func WithWriteTimeout(timeout time.Duration) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.WriteTimeout = timeout
		}
	}
}

// WithIdleTimeout is the maximum amount of time to wait for the next request when
// keep-alives are enabled. If IdleTimeout is zero, the value of ReadTimeout is
// used. If both are zero, there is no timeout.
func WithIdleTimeout(timeout time.Duration) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.IdleTimeout = timeout
		}
	}
}

// WithMiddleware adds a pre-registered middleware.
func WithMiddleware(m string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Middlewares = append(cfg.Middlewares, server.MiddlewareConfig{Plugin: m})
		}
	}
}

// WithHandlers adds custom handlers.
func WithHandlers(h ...server.RegistrationFunc) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.OptHandlers = append(cfg.OptHandlers, h...)
		}
	}
}

// WithLogLevel changes the log level from the inherited logger.
func WithLogLevel(level string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Logger.Level = level
		}
	}
}

// WithLogPlugin changes the log level from the inherited logger.
func WithLogPlugin(plugin string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Logger.Plugin = plugin
		}
	}
}
