package grpc

import (
	"crypto/tls"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	mtls "github.com/go-orb/go-orb/util/tls"
)

const (
	// DefaultAddress to listen on. By default a random port will be selected.
	DefaultAddress = ":0"

	// DefaultInsecure is set to false to make sure the network traffic is entrypted.
	DefaultInsecure = false

	// DefaultgRPCReflection enables reflection by default.
	DefaultgRPCReflection = false

	// DefaultHealthService enables the health service by default.
	DefaultHealthService = false

	// DefaultTimeout is set to 5s.
	DefaultTimeout = time.Second * 5
)

// Config provides options to the gRPC entrypoint.
type Config struct {
	server.EntrypointConfig `yaml:",inline"`

	// Address to listen on.
	//
	// If no IP is provided, an interface will be selected automatically. Private
	// interfaces are preferred, if none are found a public interface will be used.
	//
	// If no port is provided, a random port will be selected. To listen on a
	// specific interface, but with a random port, you can use '<IP>:0'.
	Address string `json:"address" yaml:"address"`

	// Insecure will start the gRPC server without TLS.
	// If set to false, and no TLS certifiate is provided, a self-signed
	// certificate will be generated.
	//
	// WARNING: don't use this in production, unless you really know what you are
	// doing. this will result in unencrypted traffic. Really, it is even advised
	// against using this in testing environments.
	Insecure bool `json:"insecure,omitempty" yaml:"insecure,omitempty"`

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
	Timeout config.Duration `json:"timeout" yaml:"timeout"`

	// Listener is a custom listener. If none provided one will be created with
	// the address and TLS config.
	Listener net.Listener `json:"-" yaml:"-"`

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
		Address:       DefaultAddress,
		Timeout:       config.Duration(DefaultTimeout),
		HealthService: DefaultHealthService,
		Reflection:    DefaultgRPCReflection,
		Insecure:      DefaultInsecure,
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
}

// WithAddress sets the address to listen on.
//
// If no IP is provided, an interface will be selected automatically. Private
// interfaces are preferred, if none are found a public interface will be used.
// To listen on all interfaces explicitly set '0.0.0.0:<port>'.
//
// If no port is provided, a random port will be selected. To listen on a
// specific interface, but with a random port, you can use '<IP>:0'.
func WithAddress(addr string) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Address = addr
		}
	}
}

// WithInsecure will create the entrypoint without using TLS.
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

// WithTimeout sets the request context timeout for requests.
//
// The handler still needs to respect the context timeout for this to have
// any effect.
func WithTimeout(timeout time.Duration) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Timeout = config.Duration(timeout)
		}
	}
}

// WithTLS sets the TLS config.
func WithTLS(config *tls.Config) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.TLS = &mtls.Config{Config: config}
		}
	}
}

// Listener sets a custom listener to pass to the server.
func Listener(listener net.Listener) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Listener = listener
		}
	}
}

// WithGRPCOptions with grpc options.
func WithGRPCOptions(opts ...grpc.ServerOption) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.GRPCOptions = append(cfg.GRPCOptions, opts...)
		}
	}
}

// WithHealthService dictates whether the gRPC health check protocol should be
// implemented. This is an implementation provided by the grpc package.
// Defaults to false.
//
// This is useful for healthprobes, such as in Kubernetes (>=1.24).
func WithHealthService(health bool) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.HealthService = health
		}
	}
}

// WithReflection dictates whether the server should implementent gRPC
// reflection. This is used by e.g. the gRPC proxy. Defaults to false.
func WithReflection(reflection bool) server.Option {
	return func(c server.EntrypointConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Reflection = reflection
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
