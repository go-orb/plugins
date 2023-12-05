// Package drpc provides the drpc server for go-orb.
package drpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	"github.com/google/uuid"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Name is the plugin name.
const Name = "drpc"

// Server is the drpc Server for go-orb.
type Server struct {
	config   Config
	logger   log.Logger
	registry registry.Type

	ctx        context.Context
	cancelFunc context.CancelFunc

	mux    *drpcmux.Mux
	server *drpcserver.Server

	endpoints []string

	// entrypointID is the entrypointID (uuid) of this entrypoint in the registry.
	entrypointID string

	started bool
}

// Start will create the listeners and start the server on the entrypoint.
func (s *Server) Start() error {
	if s.started {
		return nil
	}

	s.logger.Debug("Starting")

	// create a drpc RPC mux
	s.mux = drpcmux.New()

	s.server = drpcserver.New(s.mux)

	var err error

	listener := s.config.Listener
	if listener == nil {
		listener, err = net.Listen("tcp", s.config.Address)
		if err != nil {
			return err
		}
	}

	go func(s *Server, listener net.Listener) {
		err := s.server.Serve(s.ctx, listener)
		s.logger.Error("While serving", "error", err)
	}(s, listener)

	s.started = true

	return s.registryRegister()
}

// Stop will stop the Hertz server(s).
func (s *Server) Stop(_ context.Context) error {
	if !s.started {
		return nil
	}

	// Stops the dRPC Server.
	s.cancelFunc()

	return s.registryDeregister()
}

// Register executes a registration function on the entrypoint.
func (s *Server) Register(register orbserver.RegistrationFunc) {
	if !s.started {
		return
	}

	register(s.mux)
}

// AddEndpoint add's an endpoint to the internal list.
// This is used by the Register() callback function.
func (s *Server) AddEndpoint(name string) {
	s.endpoints = append(s.endpoints, name)
}

// Address returns the address the entrypoint is listening on, for example: [::]:8381.
func (s *Server) Address() string {
	if !s.started {
		return ""
	}

	return s.config.Address
}

// Transport returns the client transport to use: "drpc".
func (s *Server) Transport() string {
	return "drpc"
}

// EntrypointID returns the id (uuid) of this entrypoint in the registry.
func (s *Server) EntrypointID() string {
	if s.entrypointID != "" {
		return s.entrypointID
	}

	s.entrypointID = fmt.Sprintf("%s-%s", s.registry.ServiceName(), uuid.New().String())

	return s.entrypointID
}

// String returns the entrypoint type; http.
func (s *Server) String() string {
	return Name
}

// Name returns the entrypoint name.
func (s *Server) Name() string {
	return s.config.Name
}

// Type returns the component type.
func (s *Server) Type() string {
	return orbserver.ComponentType
}

// Router returns the drpc mux.
func (s *Server) Router() *drpcmux.Mux {
	return s.mux
}

func (s *Server) getEndpoints() []*registry.Endpoint {
	result := make([]*registry.Endpoint, len(s.endpoints))

	for _, r := range s.endpoints {
		s.logger.Trace("found endpoint", "name", r[1:])

		result = append(result, &registry.Endpoint{
			Name:     r,
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
		Metadata:  make(map[string]string),
	}

	return &registry.Service{
		Name:      s.registry.ServiceName(),
		Version:   s.registry.ServiceVersion(),
		Nodes:     []*registry.Node{node},
		Endpoints: s.getEndpoints(),
	}
}

func (s *Server) registryRegister() error {
	rService := s.registryService()

	return s.registry.Register(rService)
}

func (s *Server) registryDeregister() error {
	rService := s.registryService()

	return s.registry.Deregister(rService)
}

// ProvideServer creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports.
func ProvideServer(
	_ types.ServiceName,
	logger log.Logger,
	reg registry.Type,
	cfg Config,
	options ...Option,
) (*Server, error) {
	cfg.ApplyOptions(options...)

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("http validate addr '%s': %w", cfg.Address, err)
	}

	if err := addr.ValidateAddress(cfg.Address); err != nil {
		return nil, err
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	ctx, cancelFunc := context.WithCancel(context.Background())

	entrypoint := Server{
		config:     cfg,
		logger:     logger,
		registry:   reg,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}

	return &entrypoint, nil
}
