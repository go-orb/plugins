package drpc

import (
	"net"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	"github.com/google/uuid"
)

const (
	// DefaultAddress to use for new dRPC servers.
	DefaultAddress = ":0"

	// DefaultMaxConcurrentStreams for dRPC.
	DefaultMaxConcurrentStreams = 256
)

// Config provides options to the entrypoint.
type Config struct {
	server.EntrypointConfig `yaml:",inline"`

	// Listener can be used to provide your own Listener, when in use `Address` is obsolete.
	Listener net.Listener `json:"-" yaml:"-"`

	// Address to listen on.
	// If no port is provided, a random port will be selected. To listen on a
	// specific interface, but with a random port, you can use '<IP>:0'.
	Address string `json:"address" yaml:"address"`

	// MaxConcurrentStreams is the worker pool size.
	MaxConcurrentStreams int `json:"maxConcurrentStreams" yaml:"maxConcurrentStreams"`

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
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
}

// WithName sets the entrypoint name. The default name is in the format of
// 'drpc-<uuid>'.
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

// WithListener sets the entrypoints listener. This overwrites `Address`.
func WithListener(l net.Listener) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Listener = l
		}
	}
}

// WithAddress specifies the address to listen on.
// If you want to listen on all interfaces use the format "[::]:8080"
// If you want to listen on a specific interface/address use the full IP.
func WithAddress(address string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Address = address
		}
	}
}

// WithMaxConcurrentStreams sets the worker pool size.
func WithMaxConcurrentStreams(n int) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.MaxConcurrentStreams = n
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
