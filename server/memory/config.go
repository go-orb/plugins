package memory

import (
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
)

const (
	// DefaultMaxConcurrentStreams for memory.
	DefaultMaxConcurrentStreams = 256
)

// Config provides options to the entrypoint.
type Config struct {
	server.EntrypointConfig `yaml:",inline"`

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
			Name:    Name,
			Plugin:  Name,
			Enabled: true,
		},
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
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
