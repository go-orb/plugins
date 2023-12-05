// Package hertz contains a hertz server for go-orb.
package hertz

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/cloudwego/hertz/pkg/app/server"
	hconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	"github.com/go-orb/plugins/server/hertz/internal/orblog"

	"github.com/hertz-contrib/http2/factory"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Name is the plugin name.
const Name = "hertz"

// Server is the hertz Server for go-orb.
type Server struct {
	Config   Config
	Logger   log.Logger
	Registry registry.Type

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

	s.Logger.Debug("Starting")

	// for _, middleware := range s.Config.Middleware {
	// 	s.router.Use(middleware)
	// }

	hlog.SetLogger(orblog.NewLogger(s.Logger))

	hopts := []hconfig.Option{server.WithHostPorts(s.Config.Address)}
	if s.Config.H2C {
		hopts = append(hopts, server.WithH2C(true))
	}

	s.hServer = server.Default(hopts...)

	// Register handlers.
	for _, h := range s.Config.HandlerRegistrations {
		h(s)
	}

	if s.Config.H2C || s.Config.HTTP2 {
		// register http2 server factory
		s.hServer.AddProtocol("h2", factory.NewServerFactory())
	}

	errCh := make(chan error)
	go func(h *server.Hertz, errCh chan error) {
		errCh <- h.Run()
	}(s.hServer, errCh)

	if err := s.registryRegister(); err != nil {
		return fmt.Errorf("failed to register the HTTP server: %w", err)
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

	s.Logger.Debug("Stopping")

	if err := s.registryDeregister(); err != nil {
		return err
	}

	stopCtx, cancel := context.WithTimeoutCause(ctx, s.Config.StopTimeout, fmt.Errorf("timeout while stopping the hertz server"))
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
	return s.Config.Address
}

// Transport returns the client transport to use.
func (s *Server) Transport() string {
	if s.Config.H2C {
		return "hertzh2c"
	} else if !s.Config.Insecure {
		return "hertzhttps"
	}

	return "hertzhttp"
}

// EntrypointID returns the id (uuid) of this entrypoint in the registry.
func (s *Server) EntrypointID() string {
	if s.entrypointID != "" {
		return s.entrypointID
	}

	s.entrypointID = fmt.Sprintf("%s-%s", s.Registry.ServiceName(), uuid.New().String())

	return s.entrypointID
}

// String returns the entrypoint type; http.
func (s *Server) String() string {
	return Name
}

// Name returns the entrypoint name.
func (s *Server) Name() string {
	return s.Config.Name
}

// Type returns the component type.
func (s *Server) Type() string {
	return orbserver.ComponentType
}

// Router returns the hertz server.
func (s *Server) Router() *server.Hertz {
	return s.hServer
}

func (s *Server) getEndpoints() []*registry.Endpoint {
	routes := s.hServer.Routes()

	result := make([]*registry.Endpoint, len(routes))

	for _, r := range routes {
		s.Logger.Trace("found endpoint", slog.String("name", r.Path[1:]))

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
		Name:      s.Registry.ServiceName(),
		Version:   s.Registry.ServiceVersion(),
		Nodes:     []*registry.Node{node},
		Endpoints: s.getEndpoints(),
	}
}

func (s *Server) registryRegister() error {
	rService := s.registryService()

	return s.Registry.Register(rService)
}

func (s *Server) registryDeregister() error {
	rService := s.registryService()

	return s.Registry.Deregister(rService)
}

// ProvideServer creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports. One entrypoint
// can serve a HTTP1 and HTTP2/H2C server.
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

	codecs, err := cfg.NewCodecMap()
	if err != nil {
		return nil, fmt.Errorf("create codec map: %w", err)
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	entrypoint := Server{
		Config:   cfg,
		Logger:   logger,
		Registry: reg,
		codecs:   codecs,
	}

	return &entrypoint, nil
}
