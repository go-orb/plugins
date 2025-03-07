// Package drpc provides the drpc server for go-orb.
package drpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"storj.io/drpc/drpcserver"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	"github.com/google/uuid"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Plugin is the plugin name.
const Plugin = "drpc"

// Server is the drpc Server for go-orb.
type Server struct {
	config   *Config
	logger   log.Logger
	registry registry.Type

	address string

	ctx        context.Context
	cancelFunc context.CancelFunc

	mux    *Mux
	server *drpcserver.Server

	handlers    []orbserver.RegistrationFunc
	middlewares []orbserver.Middleware

	endpoints []string

	// entrypointID is the entrypointID (uuid) of this entrypoint in the registry.
	entrypointID string

	started bool
}

// Start will create the listeners and start the server on the entrypoint.
func (s *Server) Start(_ context.Context) error {
	if s.started {
		return nil
	}

	s.logger.Info("Starting", "address", s.config.Address)

	// create a drpc RPC mux
	s.mux = newMux(s)

	// Register handlers.
	for _, h := range s.handlers {
		h(s)
	}

	s.server = drpcserver.New(s.mux)

	var err error

	listener := s.config.Listener
	if listener == nil {
		listener, err = net.Listen("tcp", s.config.Address)
		if err != nil {
			return err
		}
	}

	s.address = listener.Addr().String()

	s.logger.Info("Got address", "address", s.address)

	s.ctx, s.cancelFunc = context.WithCancel(context.Background())

	go func() {
		if err := s.server.Serve(s.ctx, listener); err != nil {
			s.logger.Error("while starting the dRPC Server", "error", err)
		}
	}()

	s.started = true

	return s.registryRegister()
}

// Stop will stop the dRPC server.
func (s *Server) Stop(_ context.Context) error {
	if !s.started {
		return nil
	}

	// Stops the dRPC Server.
	s.cancelFunc()

	return s.registryDeregister()
}

// AddHandler adds a handler for later registration.
func (s *Server) AddHandler(handler orbserver.RegistrationFunc) {
	s.handlers = append(s.handlers, handler)
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

	return s.address
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

// String returns the entrypoint type.
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
	return orbserver.EntrypointType
}

// Router returns the drpc mux.
func (s *Server) Router() *Mux {
	return s.mux
}

func (s *Server) getEndpoints() []*registry.Endpoint {
	result := make([]*registry.Endpoint, 0, len(s.endpoints))

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

// Provide creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports.
func Provide(
	sections []string,
	configs types.ConfigData,
	logger log.Logger,
	reg registry.Type,
	opts ...orbserver.Option,
) (orbserver.Entrypoint, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(sections, configs, cfg); err != nil {
		return nil, err
	}

	// Configure Middlewares.
	for idx, cfgMw := range cfg.Middlewares {
		pFunc, ok := orbserver.Middlewares.Get(cfgMw.Plugin)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", orbserver.ErrUnknownMiddleware, cfgMw.Plugin)
		}

		mw, err := pFunc(append(sections, "middlewares", strconv.Itoa(idx)), configs, logger)
		if err != nil {
			return nil, err
		}

		cfg.OptMiddlewares = append(cfg.OptMiddlewares, mw)
	}

	// Get handlers.
	for _, k := range cfg.Handlers {
		h, ok := orbserver.Handlers.Get(k)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", orbserver.ErrUnknownHandler, k)
		}

		cfg.OptHandlers = append(cfg.OptHandlers, h)
	}

	return New(cfg, logger, reg)
}

// New creates a dRPC Server from a Config struct.
func New(acfg any, logger log.Logger, reg registry.Type) (orbserver.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("drpc invalid config: %v", cfg)
	}

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("drpc validate addr '%s': %w", cfg.Address, err)
	}

	if err := addr.ValidateAddress(cfg.Address); err != nil {
		return nil, err
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	ctx, cancelFunc := context.WithCancel(context.Background())

	entrypoint := Server{
		config:      cfg,
		logger:      logger,
		registry:    reg,
		handlers:    cfg.OptHandlers,
		middlewares: cfg.OptMiddlewares,
		ctx:         ctx,
		cancelFunc:  cancelFunc,
	}

	return &entrypoint, nil
}
