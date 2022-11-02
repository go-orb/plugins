package entrypoint

import (
	"crypto/tls"
	"time"
)

// Default config options.
const (
	DefaultAllowInsecure        = false
	DefaultMaxConcurrentStreams = 250
	DefaultReadTimeout          = 5 * time.Second
	DefaultWriteTimeout         = 5 * time.Second
	DefaultIdleimeout           = 5 * time.Second
	DefaultHTTP2                = true
	DefaultHTTP3                = false
)

// Option is a functional option to provide custom values to the config.
type Option func(*Config)

// Config provides options to the entrypoint.
type Config struct {
	Address string

	CertFile string
	KeyFile  string

	// Insecure will create an HTTP server without TLS, for insecure connections.
	Insecure bool
	// MaxConcurrentStreams for HTTP2.
	MaxConcurrentStreams int
	// TLS config, if none is provided self-signed certificates will be generated.
	TLS *tls.Config
	// AllowH2C allows h2c connections; HTTP2 without TLS.
	AllowH2C bool
	// HTTP2 dicates whether to also allow HTTP/2 connectionsl Defaults to true.
	HTTP2 bool
	// HTTP3 dicates whether to also start an HTTP/3.0 server. Defaults to false.
	HTTP3 bool

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewEntrypointConfig will create a new config with default values for the entrypoint.
func NewEntrypointConfig(options ...Option) Config {
	cfg := Config{
		Insecure:             false,
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		AllowH2C:             false,
		HTTP2:                DefaultHTTP2,
		HTTP3:                DefaultHTTP3,
		ReadTimeout:          DefaultReadTimeout,
		WriteTimeout:         DefaultWriteTimeout,
		IdleTimeout:          DefaultIdleimeout,
	}

	cfg.ApplyOptions(options...)

	return cfg
}

func (c *Config) ApplyOptions(options ...Option) {
	for _, option := range options {
		option(c)
	}
}

func WithAddress(address string) Option {
	return func(c *Config) {
		c.Address = address
	}
}

func WithTLSFile(certfile, keyfile string) Option {
	return func(c *Config) {
		c.CertFile = certfile
		c.KeyFile = keyfile
	}
}

func WithTLS(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.TLS = tlsConfig
	}
}

func WithInsecure() Option {
	return func(c *Config) {
		c.Insecure = true
	}
}

func WithHTTP3() Option {
	return func(c *Config) {
		c.HTTP3 = true
	}
}

func DisableHTTP2() Option {
	return func(c *Config) {
		c.HTTP2 = false
	}
}

func AllowH2C() Option {
	return func(c *Config) {
		c.AllowH2C = true
	}
}

// TODO: other options and option comments
