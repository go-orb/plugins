// Package grpc provides a gRPC entrypoint for go-orb.
package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/util/addr"
	mnet "github.com/go-orb/go-orb/util/net"
	mtls "github.com/go-orb/go-orb/util/tls"
)

// Interface guard.
var _ server.Entrypoint = (*Server)(nil)

// Server is an entrypoint with a gRPC server.
type Server struct {
	serviceName    string
	serviceVersion string
	epName         string

	server *grpc.Server

	config *Config

	logger log.Logger

	registry registry.Type

	lis net.Listener

	// health server implements the gRPC health protocol.
	health *health.Server

	started bool
}

// Provide provides a gRPC server by config.
func Provide(
	serviceName string,
	serviceVersion string,
	epName string,
	configData map[string]any,
	logger log.Logger,
	reg registry.Type,
	opts ...server.Option,
) (server.Entrypoint, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, "", configData, cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return nil, err
	}

	return New(serviceName, serviceVersion, epName, cfg, logger, reg)
}

// New creates a gRPC server by options.
func New(
	serviceName string,
	serviceVersion string,
	epName string,
	acfg any,
	logger log.Logger,
	reg registry.Type,
) (server.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("grpc invalid config: %v", cfg)
	}

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("grpc validate address '%s': %w", cfg.Address, err)
	}

	srv := Server{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		epName:         epName,
		config:         cfg,
		logger:         logger,
		registry:       reg,
	}

	srv.setupgRPCServer()

	return &srv, nil
}

func (s *Server) setupgRPCServer() {
	grpcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(s.unaryServerInterceptor()),
		grpc.StreamInterceptor(s.streamServerInterceptor()),
	}

	if len(s.config.GRPCOptions) > 0 {
		grpcOpts = append(grpcOpts, s.config.GRPCOptions...)
	}

	s.server = grpc.NewServer(grpcOpts...)

	if s.config.HealthService {
		s.health = health.NewServer()
		grpc_health_v1.RegisterHealthServer(s.server, s.health)
	}

	if s.config.Reflection {
		reflection.Register(s.server)
	}
}

// Start start the gRPC server.
func (s *Server) Start(ctx context.Context) error {
	if s.started {
		return nil
	}

	if encoding.GetCodec("json") == nil {
		codec, err := codecs.GetMime(codecs.MimeJSON)
		if err != nil {
			return err
		}

		encoding.RegisterCodec(codec)
	}

	// Register handlers.
	for _, f := range s.config.OptHandlers {
		s.Register(f)
	}

	if s.lis == nil {
		if err := s.listen(); err != nil {
			return fmt.Errorf("create listener (%s): %w", s.config.Address, err)
		}
	}

	s.logger = s.logger.With(slog.String("transport", s.Transport()), slog.String("address", s.Address()))

	s.logger.Info("gRPC server listening")

	go func() {
		if err := s.server.Serve(s.lis); err != nil {
			// TODO(davincible): propagate error here
			s.logger.Error("failed to start gRPC server", "err", err)
		}
	}()

	if s.health != nil {
		s.health.Resume()
	}

	// Register with registry.
	if err := s.registryRegister(ctx); err != nil {
		return err
	}

	s.started = true

	return nil
}

// Stop stop the gRPC server.
func (s *Server) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	s.logger.Info("gRPC server shutting down", "address", s.lis.Addr().String())

	if err := s.registryDeregister(ctx); err != nil {
		return err
	}

	done := make(chan any)

	go func() {
		if s.health != nil {
			s.health.Shutdown()
		}

		s.server.GracefulStop()

		done <- nil
	}()

	select {
	case <-ctx.Done():
		s.server.Stop()
	case <-done:
	}

	s.started = false

	return nil
}

// Config returns the server config.
func (s *Server) Config() *Config {
	return s.config
}

// Address returns the address the entypoint listens on,
// for example: 127.0.0.1:8000 .
func (s *Server) Address() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}

	return s.config.Address
}

// Transport returns the client transport to use.
func (s *Server) Transport() string {
	if !s.config.Insecure {
		return "grpcs"
	}

	return "grpc"
}

// AddHandler adds a handler for later registration.
func (s *Server) AddHandler(handler server.RegistrationFunc) {
	s.config.OptHandlers = append(s.config.OptHandlers, handler)
}

// Register executes a registration function on the entrypoint.
func (s *Server) Register(register server.RegistrationFunc) {
	register(s.server)
}

// String returns the entrypoint type; http.
func (s *Server) String() string {
	return Plugin
}

// Enabled returns if this entrypoint has been enbaled in config.
func (s *Server) Enabled() bool {
	return s.config.Enabled
}

// Name returns the entrypoint name.
func (s *Server) Name() string {
	return s.epName
}

// Type returns the component type.
func (s *Server) Type() string {
	return server.EntrypointType
}

// listen creates a listener.
func (s *Server) listen() error {
	if s.lis != nil {
		return nil
	}

	if !s.config.Insecure && s.config.TLS == nil {
		config, err := mtls.GenTLSConfig(s.config.Address)
		if err != nil {
			return fmt.Errorf("failed to generate self signed certificate: %w", err)
		}

		s.config.TLS = &mtls.Config{Config: config}
	}

	var tlsConfig *tls.Config

	if s.config.TLS != nil && s.config.TLS.Config != nil {
		s.logger.Debug("TLS config found", "config", s.config.TLS)
		tlsConfig = s.config.TLS.Config
		tlsConfig.NextProtos = []string{"h2"}
	}

	lis, err := mnet.Listen("tcp", s.config.Address, tlsConfig)
	if err != nil {
		return err
	}

	s.lis = lis

	return nil
}

func (s *Server) registryService() registry.ServiceNode {
	return registry.ServiceNode{
		Name:     s.serviceName,
		Version:  s.serviceVersion,
		Node:     s.Name(),
		Address:  s.Address(),
		Scheme:   s.Transport(),
		Metadata: make(map[string]string),
	}
}

func (s *Server) registryRegister(ctx context.Context) error {
	return s.registry.Register(ctx, s.registryService())
}

func (s *Server) registryDeregister(ctx context.Context) error {
	return s.registry.Deregister(ctx, s.registryService())
}
