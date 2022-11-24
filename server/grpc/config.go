package grpc

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"go-micro.dev/v5/util/matcher"
)

const (
	DefaultNetwork        = "tcp"
	DefaultAddress        = ":0"
	DefaultTimeout        = time.Second * 5
	DefaultgRPCReflection = true
	DefaultHealthService  = true
)

// ServerOption is gRPC server option.
type Option func(o *Config)

type Config struct {
	Name               string `json:"name" yaml:"name"`
	Address            string
	Network            string
	Timeout            time.Duration
	TLSConfig          *tls.Config
	UnaryInterceptors  matcher.Matcher[grpc.UnaryServerInterceptor]
	StreamInterceptors matcher.Matcher[grpc.StreamServerInterceptor]
	GRPCOptions        []grpc.ServerOption
	HealthService      bool
	GRPCreflection     bool
}

func NewConfig() Config {
	return Config{
		Name:               "grpc-" + uuid.NewString(),
		Address:            DefaultAddress,
		Network:            DefaultNetwork,
		Timeout:            DefaultTimeout,
		HealthService:      DefaultHealthService,
		GRPCreflection:     DefaultgRPCReflection,
		UnaryInterceptors:  matcher.NewMatcher(UnaryInterceptors),
		StreamInterceptors: matcher.NewMatcher(StreamInterceptors),
	}
}

// ApplyOptions applies a set of options.
func (c *Config) ApplyOptions(opts ...Option) {
	for _, o := range opts {
		o(c)
	}
}

// Network with server network.
func Network(network string) Option {
	return func(c *Config) {
		c.Network = network
	}
}

// Address with server address.
func Address(addr string) Option {
	return func(c *Config) {
		c.Address = addr
	}
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) Option {
	return func(s *Config) {
		s.Timeout = timeout
	}
}

// Logger with server logger.
// Deprecated: use global logger instead.
func Logger(logger log.Logger) Option {
	return func(s *Config) {}
}

// Middleware with server middleware.
// func Middleware(m ...middleware.Middleware) Option {
// 	return func(s *Config) {
// 		s.middleware.Use(m...)
// 	}
// }

// TLSConfig with TLS config.
func TLSConfig(c *tls.Config) Option {
	return func(s *Config) {
		s.TLSConfig = c
	}
}

// Listener with server lis
// func Listener(lis net.Listener) Option {
// 	return func(s *Config) {
// 		s.lis = lis
// 	}
// }

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
func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor, selector ...string) Option {
	return func(s *Config) {
		if len(selector) > 0 {
			s.UnaryInterceptors.Add(selector[0], interceptor)
			return
		}

		s.UnaryInterceptors.Use(interceptor)
	}
}

// StreamInterceptor sets a middleware for streaming gRPC calls.
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
func StreamInterceptor(interceptor grpc.StreamServerInterceptor, selector ...string) Option {
	return func(s *Config) {
		if len(selector) > 0 {
			s.StreamInterceptors.Add(selector[0], interceptor)
			return
		}

		s.StreamInterceptors.Use(interceptor)
	}
}

// Options with grpc options.
func Options(opts ...grpc.ServerOption) Option {
	return func(s *Config) {
		s.GRPCOptions = opts
	}
}
