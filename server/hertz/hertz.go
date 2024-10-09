// Package hertz contains a hertz server for go-orb.
package hertz

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"log/slog"

	"github.com/cloudwego/hertz/pkg/app/server"
	hconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	"github.com/go-orb/plugins/server/hertz/internal/orblog"

	"github.com/hertz-contrib/http2/factory"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Server is the hertz Server for go-orb.
type Server struct {
	config   *Config
	logger   log.Logger
	registry registry.Type

	address string
	hServer *server.Hertz

	// entrypointID is the entrypointID (uuid) of this entrypoint in the registry.
	entrypointID string

	codecs map[string]codecs.Marshaler

	started bool
}

// Start will create the listeners and start the server on the entrypoint.
func (s *Server) Start() error {
	if s.started {
		return nil
	}

	s.logger.Info("Starting", "address", s.config.Address)

	// Listen and close on that address, to see which port we get.
	l, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return err
	}

	s.address = l.Addr().String()

	if err := l.Close(); err != nil {
		return err
	}

	s.logger.Info("Got address", "address", s.address)

	hlog.SetLogger(orblog.NewLogger(s.logger))

	hopts := []hconfig.Option{server.WithHostPorts(s.address)}
	if s.config.H2C {
		hopts = append(hopts, server.WithH2C(true))
	}

	s.hServer = server.Default(hopts...)

	// Register handlers.
	for _, h := range s.config.OptHandlers {
		h(s)
	}

	if s.config.H2C || s.config.HTTP2 {
		// register http2 server factory
		s.hServer.AddProtocol("h2", factory.NewServerFactory())
	}

	errCh := make(chan error)
	go func(h *server.Hertz, errCh chan error) {
		errCh <- h.Run()
	}(s.hServer, errCh)

	if err := s.registryRegister(); err != nil {
		return fmt.Errorf("failed to register the hertz server: %w", err)
	}

	s.started = true

	return nil
}

// Stop will stop the Hertz server(s).
func (s *Server) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	errChan := make(chan error)
	defer close(errChan)

	s.logger.Debug("Stopping")

	if err := s.registryDeregister(); err != nil {
		return err
	}

	stopCtx, cancel := context.WithTimeoutCause(ctx, s.config.StopTimeout, errors.New("timeout while stopping the hertz server"))
	defer cancel()

	s.started = false

	return s.hServer.Shutdown(stopCtx)
}

// Register executes a registration function on the entrypoint.
func (s *Server) Register(register orbserver.RegistrationFunc) {
	register(s)
}

// Address returns the address the entrypoint is listening on.
func (s *Server) Address() string {
	return s.address
}

// Transport returns the client transport to use.
func (s *Server) Transport() string {
	if s.config.H2C {
		return "hertzh2c"
	} else if !s.config.Insecure {
		return "hertzhttps"
	}

	return "hertzhttp"
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

// Router returns the hertz server.
func (s *Server) Router() *server.Hertz {
	return s.hServer
}

func (s *Server) getEndpoints() []*registry.Endpoint {
	routes := s.hServer.Routes()

	result := make([]*registry.Endpoint, len(routes))

	for _, r := range routes {
		s.logger.Trace("found endpoint", slog.String("name", r.Path[1:]))

		result = append(result, &registry.Endpoint{
			Name:     r.Path[1:],
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
// multiple entrypoints for multiple addresses and ports. One entrypoint
// can serve a HTTP1 and HTTP2/H2C server.
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

// New creates a hertz server by options.
func New(acfg any, logger log.Logger, reg registry.Type) (orbserver.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("hertz invalid config: %v", cfg)
	}

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("hertz validate addr '%s': %w", cfg.Address, err)
	}

	if err := addr.ValidateAddress(cfg.Address); err != nil {
		return nil, err
	}

	codecs, err := cfg.NewCodecMap()
	if err != nil {
		return nil, fmt.Errorf("create codec map: %w", err)
	}

	logger = logger.With(slog.String("component", orbserver.ComponentType), slog.String("plugin", Plugin), slog.String("entrypoint", cfg.Name))

	entrypoint := Server{
		config:   cfg,
		logger:   logger,
		registry: reg,
		codecs:   codecs,
	}

	return &entrypoint, nil
}
