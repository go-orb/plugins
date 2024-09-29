// Package grpc provides a gRPC entrypoint for go-orb.
package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	mnet "github.com/go-orb/go-orb/util/net"
	mtls "github.com/go-orb/go-orb/util/tls"
	"github.com/google/uuid"
)

// Interface guard.
var _ server.Entrypoint = (*ServerGRPC)(nil)

// ServerGRPC is an entrypoint with a gRPC server.
type ServerGRPC struct {
	server *grpc.Server

	config Config

	logger log.Logger

	registry registry.Type

	// entrypointID is the entrypointID (uuid) of this entrypoint in the registry.
	entrypointID string

	lis net.Listener

	// health server implements the gRPC health protocol.
	health *health.Server

	// Cache the middleware count to prevent evaluation on every request.
	unaryMiddleware int

	started bool
}

// Provide creates a gRPC server by options.
func Provide(
	_ types.ServiceName,
	logger log.Logger,
	reg registry.Type,
	cfg Config,
	opts ...Option,
) (*ServerGRPC, error) {
	var err error

	cfg.ApplyOptions(opts...)

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("grpc validate address '%s': %w", cfg.Address, err)
	}

	logger = logger.With(slog.String("component", server.ComponentType), slog.String("plugin", Plugin), slog.String("entrypoint", cfg.Name))

	srv := ServerGRPC{
		config:   cfg,
		logger:   logger,
		registry: reg,
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
			// TODO(davincible): propagate error here
			s.logger.Error("failed to start gRPC server", "err", err)
		}
	}()

	if s.health != nil {
		s.health.Resume()
	}

	if err := s.register(); err != nil {
		return err
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

	if err := s.deregister(); err != nil {
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
//
// Note that this a copy and you cannot mutate it.
// Some values such as arrays are pointers, but mutating them either results
// in undefined behavior, or no change, as they are already processed.
func (s *ServerGRPC) Config() Config {
	return s.config
}

// Address returns the address the entypoint listens on,
// for example: 127.0.0.1:8000 .
func (s *ServerGRPC) Address() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}

	return s.config.Address
}

// Transport returns the client transport to use.
func (s *ServerGRPC) Transport() string {
	return "grpc"
}

// EntrypointID returns the id (uuid) of this entrypoint in the registry.
func (s *ServerGRPC) EntrypointID() string {
	if s.entrypointID != "" {
		return s.entrypointID
	}

	s.entrypointID = fmt.Sprintf("%s-%s", s.registry.ServiceName(), uuid.New().String())

	return s.entrypointID
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

func (s *ServerGRPC) getEndpoints() []*registry.Endpoint {
	sInfo := s.server.GetServiceInfo()

	result := make([]*registry.Endpoint, len(sInfo))

	for k := range sInfo {
		s.logger.Trace("found endpoint", slog.String("name", k))

		result = append(result, &registry.Endpoint{
			Name:     k,
			Metadata: map[string]string{"stream": "true"},
		})
	}

	return result
}

func (s *ServerGRPC) registryService() *registry.Service {
	node := &registry.Node{
		ID:        s.EntrypointID(),
		Address:   s.Address(),
		Transport: s.Transport(),
		Metadata:  make(map[string]string),
	}

	eps := s.getEndpoints()

	return &registry.Service{
		Name:      s.registry.ServiceName(),
		Version:   s.registry.ServiceVersion(),
		Nodes:     []*registry.Node{node},
		Endpoints: eps,
	}
}

func (s *ServerGRPC) register() error {
	return s.registry.Register(s.registryService())
}

func (s *ServerGRPC) deregister() error {
	return s.registry.Deregister(s.registryService())
}
