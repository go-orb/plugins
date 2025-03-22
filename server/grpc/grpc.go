// Package grpc provides a gRPC entrypoint for go-orb.
package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"

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
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	mnet "github.com/go-orb/go-orb/util/net"
	mtls "github.com/go-orb/go-orb/util/tls"
)

// Interface guard.
var _ server.Entrypoint = (*Server)(nil)

// Server is an entrypoint with a gRPC server.
type Server struct {
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
	configData map[string]any,
	logger log.Logger,
	reg registry.Type,
	opts ...server.Option,
) (server.Entrypoint, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, "", configData, cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return nil, err
	}

	// Configure Middlewares.
	for idx, cfgMw := range cfg.Middlewares {
		pFunc, ok := server.Middlewares.Get(cfgMw.Plugin)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", server.ErrUnknownMiddleware, cfgMw.Plugin)
		}

		mw, err := pFunc([]string{"middlewares"}, strconv.Itoa(idx), configData, logger)
		if err != nil {
			return nil, err
		}

		cfg.OptMiddlewares = append(cfg.OptMiddlewares, mw)
	}

	return New(cfg, logger, reg)
}

// New creates a gRPC server by options.
func New(acfg any, logger log.Logger, reg registry.Type) (server.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("grpc invalid config: %v", cfg)
	}

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("grpc validate address '%s': %w", cfg.Address, err)
	}

	logger = logger.With(slog.String("component", server.ComponentType), slog.String("plugin", Plugin), slog.String("entrypoint", cfg.Name))

	// Get handlers.
	for _, k := range cfg.Handlers {
		h, ok := server.Handlers.Get(k)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", server.ErrUnknownHandler, k)
		}

		cfg.OptHandlers = append(cfg.OptHandlers, h)
	}

	srv := Server{
		config:   cfg,
		logger:   logger,
		registry: reg,
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
func (s *Server) Start(_ context.Context) error {
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
	if err := s.register(); err != nil {
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

// Metadata returns the metadata of this entrypoint.
func (s *Server) Metadata() map[string]string {
	return s.config.Metadata
}

// EntrypointID returns the id of this entrypoint (node) in the registry.
func (s *Server) EntrypointID() string {
	return s.registry.ServiceName() + types.DefaultSeparator + s.config.Name
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
	return s.config.Name
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

func (s *Server) getEndpoints() []*registry.Endpoint {
	sInfo := s.server.GetServiceInfo()

	result := make([]*registry.Endpoint, 0, len(sInfo))

	for k := range sInfo {
		s.logger.Trace("found endpoint", slog.String("name", k))

		result = append(result, &registry.Endpoint{
			Name:     k,
			Metadata: map[string]string{"stream": "true"},
		})
	}

	return result
}

func (s *Server) registryService() *registry.Service {
	node := &registry.Node{
		ID:        s.EntrypointID(),
		Address:   s.Address(),
		Transport: s.Transport(),
		Metadata:  s.Metadata(),
	}

	eps := s.getEndpoints()

	return &registry.Service{
		Name:      s.registry.ServiceName(),
		Version:   s.registry.ServiceVersion(),
		Nodes:     []*registry.Node{node},
		Endpoints: eps,
	}
}

func (s *Server) register() error {
	return s.registry.Register(s.registryService())
}

func (s *Server) deregister() error {
	return s.registry.Deregister(s.registryService())
}
