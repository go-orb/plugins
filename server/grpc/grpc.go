package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/types/component"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc/reflection"
)

var _ server.Entrypoint = (*ServerGRPC)(nil)

const Plugin = "grpc"

// ServerGRPC is a proto RPC server, such as gRPC or dRPC.
type ServerGRPC struct {
	server *grpc.Server

	config Config

	logger log.Logger

	lis net.Listener

	health *health.Server

	// Cache the middleware count to prevent evaluation on every request.
	unaryMiddleware int

	started bool
}

// ProvideServerGRPC creates a gRPC server by options.
func ProvideServerGRPC(opts ...Option) *ServerGRPC {
	cfg := NewConfig()
	cfg.ApplyOptions(opts...)

	srv := &ServerGRPC{
		config: cfg,
	}

	srv.setupgRPCServer()

	return srv
}

func (s *ServerGRPC) setupgRPCServer() {
	grpcOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(s.streamServerInterceptor()),
	}

	if len(s.config.GRPCOptions) > 0 {
		grpcOpts = append(grpcOpts, s.config.GRPCOptions)
	}

	s.unaryMiddleware = s.config.UnaryInterceptors.Len()
	if s.unaryMiddleware > 0 || s.config.Timeout > 0 {
		grpcOpts = append(grpcOpts, grpc.UnaryInterceptor(s.unaryServerInterceptor()))
	}

	if s.config.UnaryInterceptors.Len() > 0 {
		grpcOpts = append(grpcOpts, grpc.UnaryInterceptor(s.unaryServerInterceptor()))
	}

	s.server = grpc.NewServer(grpcOpts...)

	if s.config.HealthService {
		s.health = health.NewServer()
		grpc_health_v1.RegisterHealthServer(s.server, s.health)
	}

	if s.config.GRPCreflection {
		reflection.Register(s.server)
	}
}

// Use uses a service middleware with selector.
// selector:
//   - '/*'
//   - '/helloworld.v1.Greeter/*'
//   - '/helloworld.v1.Greeter/SayHello'
// func (s *Server) Use(selector string, m ...middleware.Middleware) {
// 	s.middleware.Add(selector, m...)
// }

// Start start the gRPC server.
func (s *ServerGRPC) Start() error {
	if s.started {
		return nil
	}

	s.logger.Info("gRPC server listening on: " + s.lis.Addr().String())

	if err := s.listen(); err != nil {
		return fmt.Errorf("create listener (%s %s): %w", s.config.Network, s.config.Address, err)
	}

	go func() {
		if err := s.server.Serve(s.lis); err != nil {
			// TODO: propagate error here
			s.logger.Error("failed to start gRPC server", err)
		}
	}()

	if s.health != nil {
		s.health.Resume()
	}

	return nil
}

// Stop stop the gRPC server.
func (s *ServerGRPC) Stop(ctx context.Context) error {
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

	return nil
}

// Register executes a registration function on the entrypoint.
func (s *ServerGRPC) Register(register server.RegistrationFunc) {
	register(s)
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
func (s *ServerGRPC) Type() component.Type {
	return server.ComponentType
}

func (s *ServerGRPC) listen() error {
	if s.lis != nil {
		return nil
	}

	if s.config.TLSConfig != nil {
		lis, err := tls.Listen(s.config.Network, s.config.Address, s.config.TLSConfig)
		if err != nil {
			return err
		}

		s.lis = lis

		return nil
	}

	lis, err := net.Listen(s.config.Network, s.config.Address)
	if err != nil {
		return err
	}

	s.lis = lis

	return nil
}
