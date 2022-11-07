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

// ApplyOptions applies a set of options to the config.
func (c *Config) ApplyOptions(options ...Option) {
	for _, option := range options {
		option(c)
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

// WithTLSFile loads in a certificate and keyfile.
func WithTLSFile(certfile, keyfile string) Option {
	return func(c *Config) {
		// TODO: load them in already here, and then set the contents in the configt.
		c.CertFile = certfile
		c.KeyFile = keyfile
	}
}

// WithTLS sets a tls config.
func WithTLS(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.TLS = tlsConfig
	}
}

// WithInsecure will create the entrypoint without using TLS.
// Note: as a result you can only make insecure HTTP requests, and no HTTP2
// unless you set WithH2C.
// It is not recommended to use this option as it will result in  unecrypted HTTP traffick.
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

// DisableHTTP2 will prevent the creation of an HTTP2 server on the entrypoint.
func DisableHTTP2() Option {
	return func(c *Config) {
		c.HTTP2 = false
	}
}

// AllowH2C will allow H2C connections on the entrypoint. H2C is HTTP2 without TLS.
// It is not recommended to turn this on.
func AllowH2C() Option {
	return func(c *Config) {
		c.AllowH2C = true
	}
}

// TODO: other options and option comments
