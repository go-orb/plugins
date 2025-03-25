package drpc

import (
	"net"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
)

const (
	// DefaultAddress to use for new dRPC servers.
	DefaultAddress = ":0"
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
		Address: DefaultAddress,
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
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
