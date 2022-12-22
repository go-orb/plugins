// Package grpc provides a gRPC entrypoint for go-micro.
package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"golang.org/x/exp/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/types"
	"go-micro.dev/v5/util/addr"
	mnet "go-micro.dev/v5/util/net"
	mtls "go-micro.dev/v5/util/tls"
)

var _ server.Entrypoint = (*ServerGRPC)(nil)

// Plugin name.
const Plugin = "grpc"

// ServerGRPC is an entrypoint with a gRPC server.
type ServerGRPC struct {
	server *grpc.Server

	config Config

	logger log.Logger

	lis net.Listener

	// health server implements the gRPC health protocol.
	health *health.Server

	// Cache the middleware count to prevent evaluation on every request.
	unaryMiddleware int

	started bool
}

// ProvideServerGRPC creates a gRPC server by options.
func ProvideServerGRPC(
	_ types.ServiceName,
	logger log.Logger,
	cfg Config,
	opts ...Option,
) (*ServerGRPC, error) {
	var err error

	cfg.ApplyOptions(opts...)

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("grpc validate address '%s': %w", cfg.Address, err)
	}

	logger, err = logger.WithComponent(server.ComponentType, Plugin, cfg.Logger.Plugin, cfg.Logger.Level)
	if err != nil {
		return nil, fmt.Errorf("create %s (http) component logger: %w", cfg.Name, err)
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	srv := ServerGRPC{
		config: cfg,
		logger: logger,
	}

	srv.setupgRPCServer()

	return &srv, nil
}

func (s *ServerGRPC) setupgRPCServer() {
	grpcOpts := []grpc.ServerOption{}

	s.unaryMiddleware = s.config.UnaryInterceptors.Len()
	if s.unaryMiddleware > 0 || s.config.Timeout > 0 {
		grpcOpts = append(grpcOpts, grpc.UnaryInterceptor(s.unaryServerInterceptor()))
	}

	if s.config.StreamInterceptors.Len() > 0 {
		grpcOpts = append(grpcOpts, grpc.StreamInterceptor(s.streamServerInterceptor()))
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
func (s *ServerGRPC) Start() error {
	if s.started {
		return nil
	}

	// Register handlers.
	for _, f := range s.config.HandlerRegistrations {
		s.Register(f)
	}

	if s.lis == nil {
		if err := s.listen(); err != nil {
			return fmt.Errorf("create listener (%s): %w", s.config.Address, err)
		}
	}

	s.logger.Info("gRPC server listening on: " + s.lis.Addr().String())

	go func() {
		if err := s.server.Serve(s.lis); err != nil {
			// TODO: propagate error here
			s.logger.Error("failed to start gRPC server", err)
		}
	}()

	if s.health != nil {
		s.health.Resume()
	}

	s.started = true

	return nil
}

// Stop stop the gRPC server.
func (s *ServerGRPC) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	s.logger.Info("gRPC server shutting down: " + s.lis.Addr().String())

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
//
// Note that this a copy and you cannot mutate it.
// Some values such as arrays are pointers, but mutating them either results
// in undefined behavior, or no change, as they are already processed.
func (s *ServerGRPC) Config() Config {
	return s.config
}

// Address returns the address the entypoint listens on.
func (s *ServerGRPC) Address() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}

	return s.config.Address
}

// Register executes a registration function on the entrypoint.
func (s *ServerGRPC) Register(register server.RegistrationFunc) {
	register(s.server)
}

// String returns the entrypoint type; http.
func (s *ServerGRPC) String() string {
	return Plugin
}

// Name returns the entrypoint name.
func (s *ServerGRPC) Name() string {
	return s.config.Name
}

// Type returns the component type.
func (s *ServerGRPC) Type() string {
	return server.ComponentType
}

// listen creates a listener.
func (s *ServerGRPC) listen() error {
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
		tlsConfig = s.config.TLS.Config
	}

	lis, err := mnet.Listen("tcp", s.config.Address, tlsConfig)
	if err != nil {
		return err
	}

	s.lis = lis

	return nil
}
