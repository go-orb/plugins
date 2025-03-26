package http

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	mtls "github.com/go-orb/go-orb/util/tls"
)

const (
	// DefaultNetwork to use for new HTTP servers.
	DefaultNetwork = "tcp"

	// DefaultAddress to use for new HTTP servers.
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

	// DefaultHTTP2 dicates whether to also allow HTTP/2 and HTTP/3 connections.
	DefaultHTTP2 = true

	// DefaultHTTP3 dicates whether to also start an HTTP/3.0 server.
	DefaultHTTP3 = false

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

// Errors.
var (
	ErrNoMatchingCodecs = errors.New("no matching codecs found, did you register the codec plugins?")
)

// Config provides options to the entrypoint.
type Config struct {
	server.EntrypointConfig `yaml:",inline"`

	// Network to listen on.
	// Either 'tcp' or 'unix'.
	// Defaults to 'tcp'.
	Network string `json:"network" yaml:"network"`

	// Address to listen on.
	// If no IP is provided, an interface will be selected automatically. Private
	// interfaces are preferred, if none are found a public interface will be used.
	//
	// If no port is provided, a random port will be selected. To listen on a
	// specific interface, but with a random port, you can use '<IP>:0'.
	//
	// If network is "unix", the address is the path to the unix socket.
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

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	ReadTimeout config.Duration `json:"readTimeout" yaml:"readTimeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout config.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout config.Duration `json:"idleTimeout" yaml:"idleTimeout"`

	// Logger allows you to dynamically change the log level and plugin for a
	// specific entrypoint.
	Logger log.Config `json:"logger" yaml:"logger"`
}

// NewConfig will create a new default config for the entrypoint.
func NewConfig(options ...server.Option) *Config {
	cfg := &Config{
		EntrypointConfig: server.EntrypointConfig{
			Plugin:  Plugin,
			Enabled: true,
		},
		Network:              DefaultNetwork,
		Address:              DefaultAddress,
		Insecure:             DefaultInsecure,
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		MaxHeaderBytes:       DefaultMaxHeaderBytes,
		H2C:                  DefaultAllowH2C,
		HTTP2:                DefaultHTTP2,
		HTTP3:                DefaultHTTP3,
		Gzip:                 DefaultEnableGzip,
		ReadTimeout:          config.Duration(DefaultReadTimeout),
		WriteTimeout:         config.Duration(DefaultWriteTimeout),
		IdleTimeout:          config.Duration(DefaultIdleTimeout),
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
}

// WithNetwork specifies the network to listen on.
func WithNetwork(network string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Network = network
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

// WithHTTP3 will additionally enable an HTTP3 server on the entrypoint.
func WithHTTP3() server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.HTTP3 = true
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

// WithGzip enables gzip response compression server wide onall responses.
// Only use this if your messages are sufficiently large. For small messages
// the compute overhead is not worth the reduction in transport time.
//
// Alternatively, you can send a gzip compressed request, and the server
// will send back a gzip compressed respponse.
func WithGzip() server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Gzip = true
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
			cfg.ReadTimeout = config.Duration(timeout)
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
			cfg.WriteTimeout = config.Duration(timeout)
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
			cfg.IdleTimeout = config.Duration(timeout)
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
