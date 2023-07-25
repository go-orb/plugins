package grpc

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"

	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/util/matcher"
	mtls "github.com/go-orb/go-orb/util/tls"
)

var _ (server.EntrypointConfig) = (*Config)(nil)

const (
	// DefaultAddress to listen on. By default a random port will be selected
	// with a preferably private network interface. Otherwise a public interface.
	DefaultAddress = ":0"

	// DefaultInsecure is set to false to make sure the network traffic is entrypted.
	DefaultInsecure = false

	// DefaultgRPCReflection enables reflection by default.
	DefaultgRPCReflection = true

	// DefaultHealthService enables the health service by default.
	DefaultHealthService = true

	// DefaultTimeout is set to 5s.
	DefaultTimeout = time.Second * 5
)

// Option is a functional option to provide custom values to the config.
type Option func(o *Config)

// Config provides options to the gRPC entrypoint.
type Config struct {
	// Name is the entrypoint name.
	//
	// The default name is 'grpc-<random uuid>'
	Name string `json:"name" yaml:"name"`

	// Address to listen on.
	//
	// If no IP is provided, an interface will be selected automatically. Private
	// interfaces are preferred, if none are found a public interface will be used.
	//
	// If no port is provided, a random port will be selected. To listen on a
	// specific interface, but with a random port, you can use '<IP>:0'.
	Address string

	// Insecure will start the gRPC server without TLS.
	// If set to false, and no TLS certifiate is provided, a self-signed
	// certificate will be generated.
	//
	// WARNING: don't use this in production, unless you really know what you are
	// doing. this will result in unencrypted traffic. Really, it is even advised
	// against using this in testing environments.
	Insecure bool

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

	// HandlerRegistrations are all handler registration functions that will be
	// registered to the server upon startup.
	//
	// You can statically add handlers by using the fuctional server options.
	// Optionally, you can dynamically add handlers by registering them to the
	// Handlers global, and setting them explicitly in the config.
	HandlerRegistrations server.HandlerRegistrations `json:"handlers" yaml:"handlers"`

	// UnaryInterceptors are middlware for unary gRPC calls. These are all
	// request handlers that don't use streaming.
	//
	// Optionally selectors can be provided to the middleware to limit which
	// requests the middleware is called on.
	UnaryInterceptors matcher.Matcher[grpc.UnaryServerInterceptor] `json:"middleware" yaml:"middleware"`

	// StreamInterceptors are middlware for streaming gRPC calls.
	//
	// Optionally selectors can be provided to the middleware to limit which
	// requests the middleware is called on.
	StreamInterceptors matcher.Matcher[grpc.StreamServerInterceptor] `json:"streamMiddleware" yaml:"streamMiddleware"`

	// GRPCOptions are options provided by the grpc package, and will be directly
	// passed ot the gRPC server.
	GRPCOptions []grpc.ServerOption `json:"-" yaml:"-"`

	// HealthService dictates whether the gRPC health check protocol should be
	// implemented. This is an implementation provided by the grpc package.
	// Defaults to true.
	//
	// This is useful for healthprobes, such as in Kubernetes (>=1.24).
	HealthService bool `json:"health" yaml:"health"`

	// Reflection dictates whether the server should implementent gRPC
	// reflection. This is used by e.g. the gRPC proxy. Defaults to true.
	Reflection bool `json:"reflection" yaml:"reflection"`

	// Timeout adds a timeout to the request context on the request handler.
	//
	// The handler still needs to respect the context timeout for this to have
	// any effect.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// Listener is a custom listener. If none provided one will be created with
	// the address and TLS config.
	Listener net.Listener `json:"-" yaml:"-"`

	// Logger allows you to dynamically change the log level and plugin for a
	// specific entrypoint.
	Logger struct {
		Level  slog.Level `json:"level,omitempty" yaml:"level,omitempty"` // TODO: change with custom level
		Plugin string     `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	} `json:"logger" yaml:"logger"`
}

// NewConfig will create a new default config for the entrypoint.
func NewConfig() *Config {
	return &Config{
		Name:                 "grpc-" + uuid.NewString(),
		Address:              DefaultAddress,
		Timeout:              DefaultTimeout,
		HealthService:        DefaultHealthService,
		Reflection:           DefaultgRPCReflection,
		Insecure:             DefaultInsecure,
		UnaryInterceptors:    matcher.NewMatcher(UnaryInterceptors),
		StreamInterceptors:   matcher.NewMatcher(StreamInterceptors),
		HandlerRegistrations: make(server.HandlerRegistrations),
	}
}

// ApplyOptions applies a set of options.
func (c *Config) ApplyOptions(opts ...Option) {
	for _, o := range opts {
		o(c)
	}
}

// Copy creates a copy of the config.
func (c Config) Copy() server.EntrypointConfig {
	return &c
}

// GetAddress returns the entrypoint address.
func (c Config) GetAddress() string {
	return c.Address
}

// WithName sets the entrypoint name. The default name is in the format of
// 'grpc-<uuid>'.
// Setting a custom name allows you to dynamically reference the entrypoint in
// the file config, and makes it easier to attribute the logs.
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithAddress sets the address to listen on.
//
// If no IP is provided, an interface will be selected automatically. Private
// interfaces are preferred, if none are found a public interface will be used.
// To listen on all interfaces explicitly set '0.0.0.0:<port>'.
//
// If no port is provided, a random port will be selected. To listen on a
// specific interface, but with a random port, you can use '<IP>:0'.
func WithAddress(addr string) Option {
	return func(c *Config) {
		c.Address = addr
	}
}

// WithInsecure will create the entrypoint without using TLS.
//
// WARNING: don't use this in production, unless you really know what you are
// doing. this will result in unencrypted traffic. Really, it is even advised
// against using this in testing environments.
func WithInsecure(insecure bool) Option {
	return func(c *Config) {
		c.Insecure = insecure
	}
}

// WithTimeout sets the request context timeout for requests.
//
// The handler still needs to respect the context timeout for this to have
// any effect.
func WithTimeout(timeout time.Duration) Option {
	return func(o *Config) {
		o.Timeout = timeout
	}
}

// WithTLS sets the TLS config.
func WithTLS(config *tls.Config) Option {
	return func(s *Config) {
		s.TLS = &mtls.Config{Config: config}
	}
}

// Listener sets a custom listener to pass to the server.
func Listener(listener net.Listener) Option {
	return func(s *Config) {
		s.Listener = listener
	}
}

// WithUnaryInterceptor sets a middleware for unary (simple non-streaming) calls.
//
// Optionally, a selctor regex can be provided to limit the scope on which the
// middleware should be called.
//
// Selector example:
//   - /*  > special case, will be replaced with '.*'
//   - .*
//   - /myPkg.myService/*
//   - /myPkg.myService/Echo
//   - /myPkg.myService/Echo[1-9]
//   - Echo$
func WithUnaryInterceptor(name string, interceptor grpc.UnaryServerInterceptor, selector ...string) Option {
	UnaryInterceptors.Set(name, interceptor)

	return func(s *Config) {
		if len(selector) > 0 {
			s.UnaryInterceptors.Add(selector[0], name, interceptor)
			return
		}

		s.UnaryInterceptors.Use(name, interceptor)
	}
}

// WithStreamInterceptor sets a middleware for streaming gRPC calls.
//
// Optionally, a selctor regex can be provided to limit the scope on which the
// middleware should be called.
//
// Selector example:
//   - /*  > special case, will be replaced with '.*'
//   - .*
//   - /myPkg.myService/*
//   - /myPkg.myService/Echo
//   - /myPkg.myService/Echo[1-9]
//   - Echo$
func WithStreamInterceptor(name string, interceptor grpc.StreamServerInterceptor, selector ...string) Option {
	return func(s *Config) {
		if len(selector) > 0 {
			s.StreamInterceptors.Add(selector[0], name, interceptor)
			return
		}

		s.StreamInterceptors.Use(name, interceptor)
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

// WithGRPCOptions with grpc options.
func WithGRPCOptions(opts ...grpc.ServerOption) Option {
	return func(s *Config) {
		s.GRPCOptions = opts
	}
}

// WithHealthService dictates whether the gRPC health check protocol should be
// implemented. This is an implementation provided by the grpc package.
// Defaults to true.
//
// This is useful for healthprobes, such as in Kubernetes (>=1.24).
func WithHealthService(health bool) Option {
	return func(s *Config) {
		s.HealthService = health
	}
}

// WithGRPCReflection dictates whether the server should implementent gRPC
// reflection. This is used by e.g. the gRPC proxy. Defaults to true.
func WithGRPCReflection(reflection bool) Option {
	return func(s *Config) {
		s.Reflection = reflection
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

// WithDefaults sets default options to use on the creation of new gRPC entrypoints.
func WithDefaults(options ...Option) server.Option {
	return func(c *server.Config) {
		cfg, ok := c.Defaults[Plugin].(*Config)
		if !ok {
			// Should never happen.
			panic(fmt.Errorf("http.WithDefaults received invalid type, not *grpc.Config, but '%T'", c.Defaults[Plugin]))
		}

		cfg.ApplyOptions(options...)

		c.Defaults[Plugin] = cfg
	}
}

// WithEntrypoint adds a gRPC entrypoint with the provided options.
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
