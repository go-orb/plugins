package drpc

import (
	"fmt"
	"net"
	"time"

	"log/slog"

	"github.com/go-orb/go-orb/server"
	"github.com/google/uuid"
)

const (
	// DefaultAddress to use for new dRPC servers.
	// If set to "random", the default, a random address will be selected,
	// preferably on a private interface (XX subet). TODO: implement.
	DefaultAddress = "[::]:8381"

	// DefaultMaxConcurrentStreams for HTTP2.
	DefaultMaxConcurrentStreams = 512

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
	DefaultConfigSection = Name
)

// DefaultCodecWhitelist is the default allowed list of codecs to be used for
// HTTP request encoding/decoding. This means that if any of these plugins are
// registered, they will be included in the server's available codecs.
// If they are not registered, the server will not be able to handle these formats.
func DefaultCodecWhitelist() []string {
	return []string{"proto", "jsonpb", "form", "xml"}
}

// Option is a functional option to provide custom values to the config.
type Option func(*Config)

// Config provides options to the entrypoint.
type Config struct {
	// Name is the entrypoint name.
	//
	// The default name is 'http-<random uuid>'
	Name string `json:"name" yaml:"name"`

	// Listener can be used to provide your own Listener, when in use `Address` is obsolete.
	Listener net.Listener `json:"-" yaml:"-"`

	// Address to listen on.
	// TODO(davincible): implement this, and the address method.
	// If no IP is provided, an interface will be selected automatically. Private
	// interfaces are preferred, if none are found a public interface will be used.
	//
	// If no port is provided, a random port will be selected. To listen on a
	// specific interface, but with a random port, you can use '<IP>:0'.
	Address string `json:"address" yaml:"address"`

	// HandlerRegistrations are all handler registration functions that will be
	// registered to the server upon startup.
	//
	// You can statically add handlers by using the fuctional server options.
	// Optionally, you can dynamically add handlers by registering them to the
	// Handlers global, and setting them explicitly in the config.
	HandlerRegistrations server.HandlerRegistrations `json:"handlers" yaml:"handlers"`

	// Logger allows you to dynamically change the log level and plugin for a
	// specific entrypoint.
	Logger struct {
		Level  slog.Level `json:"level,omitempty" yaml:"level,omitempty"` // TODO(davincible): change with custom level
		Plugin string     `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	} `json:"logger" yaml:"logger"`
}

// NewConfig will create a new default config for the entrypoint.
func NewConfig(options ...Option) *Config {
	cfg := Config{
		Name:                 "dprc-" + uuid.NewString(),
		Address:              DefaultAddress,
		HandlerRegistrations: make(server.HandlerRegistrations),
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

// WithListener sets the entrypoints listener. This overwrites `Address`.
func WithListener(l net.Listener) Option {
	return func(c *Config) {
		c.Listener = l
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

// WithConfig will set replace the server config with config provided as argument.
// Warning: any options applied previous to this option will be overwritten by
// the contents of the config provided here.
func WithConfig(config Config) Option {
	return func(c *Config) {
		*c = config
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

// WithLogLevel changes the log level from the inherited logger.
func WithLogLevel(level slog.Level) Option {
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
		cfg, ok := c.Defaults[Name].(*Config)
		if !ok {
			// Should never happen.
			panic(fmt.Errorf("http.WithDefaults received invalid type, not *server.Config, but '%T'", cfg))
		}

		cfg.ApplyOptions(options...)

		c.Defaults[Name] = cfg
	}
}

// WithEntrypoint adds an HTTP entrypoint with the provided options.
func WithEntrypoint(options ...Option) server.Option {
	return func(c *server.Config) {
		cfgAny, ok := c.Defaults[Name]
		if !ok {
			// Should never happen, but just in case.
			panic("no defaults for http entrypoint found")
		}

		cfg := cfgAny.Copy().(*Config) //nolint:errcheck

		cfg.ApplyOptions(options...)

		c.Templates[cfg.Name] = server.EntrypointTemplate{
			Enabled: true,
			Type:    Name,
			Config:  cfg,
		}
	}
}
