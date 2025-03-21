// Package http provides an HTTP server implementation.
// It provides an HTTP1, HTTP2, and HTTP3 server, the first two enabled by default.
//
// One server contains multiple entrypoints, with one entrypoint being one
// address to listen on. Each entrypoint with start its own HTTP2 server, and
// optionally also an HTTP3 server. Each entrypoint can be customized individually,
// but default options are provided, and can be tweaked.
//
// The architecture is based on the Traefik server implementation.
package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"log/slog"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/util/addr"
	mtls "github.com/go-orb/go-orb/util/tls"

	mtcp "github.com/go-orb/plugins/server/http/utils/tcp"
	mudp "github.com/go-orb/plugins/server/http/utils/udp"
)

var _ server.Entrypoint = (*Server)(nil)

// Plugin is the plugin name.
const Plugin = "http"

// Server represents a listener on one address. You can create multiple
// entrypoints for multiple addresses and ports. This is e.g. useful if you
// want to listen on multiple interfaces, or multiple ports in parallel, even
// with the same handler.
type Server struct {
	config   *Config
	logger   log.Logger
	registry registry.Type

	// router is not exported as you can't change the router after server creation.
	// The router here is merely a reference to the router that is used in the servers
	// themselves. You can fetch the router with the getter, and register handlers,
	// or mount other routers.
	router  *Router
	handler http.Handler

	httpServer  *httpServer
	http3Server *http3server

	listenerUDP net.PacketConn
	listenerTCP net.Listener

	started bool

	activeRequests int64 // accessed atomically
}

// Provide creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports. One entrypoint
// can serve a HTTP1, HTTP2 and HTTP3 server. If you enable HTTP3 it will listen
// on both TCP and UDP on the same port.
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

	// Get handlers.
	for _, k := range cfg.Handlers {
		h, ok := server.Handlers.Get(k)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", server.ErrUnknownHandler, k)
		}

		cfg.OptHandlers = append(cfg.OptHandlers, h)
	}

	return New(cfg, logger, reg)
}

// New creates a http server by options.
func New(acfg any, logger log.Logger, reg registry.Type) (server.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("http invalid config: %v", cfg)
	}

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("http validate addr '%s': %w", cfg.Address, err)
	}

	if err := addr.ValidateAddress(cfg.Address); err != nil {
		return nil, err
	}

	router := NewRouter(logger)

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	entrypoint := Server{
		config:   cfg,
		logger:   logger,
		registry: reg,
		router:   router,
	}

	entrypoint.config.TLS, err = entrypoint.setupTLS()
	if err != nil {
		return nil, err
	}

	entrypoint.httpServer, err = entrypoint.newHTTPServer(router)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP server: %w", err)
	}

	if entrypoint.config.HTTP3 {
		entrypoint.http3Server = entrypoint.newHTTP3Server()
	}

	return &entrypoint, nil
}

// Start will create the listeners and start the server on the entrypoint.
func (s *Server) Start(_ context.Context) error {
	if s.started {
		return nil
	}

	var err error

	s.logger.Info("Starting", "address", s.config.Address)

	// Register handlers.
	for _, f := range s.config.OptHandlers {
		s.Register(f)
	}

	var tlsConfig *tls.Config

	if s.config.TLS != nil {
		tlsConfig = s.config.TLS.Config
	}

	s.listenerTCP, err = mtcp.BuildListenerTCP(s.config.Address, tlsConfig)
	if err != nil {
		return err
	}

	s.logger = s.logger.With(slog.String("transport", s.Transport()), slog.String("address", s.Address()))

	s.logger.Info("HTTP server listening")

	go func() {
		if err = s.httpServer.Start(s.listenerTCP); err != nil {
			s.logger.Error("failed to start HTTP server: %w", "err", err)
		}
	}()

	if !s.config.HTTP3 {
		if err := s.register(); err != nil {
			return fmt.Errorf("failed to register the HTTP server: %w", err)
		}

		s.started = true

		return nil
	}

	// Listen on the same UDP port as TCP for HTTP3
	s.listenerUDP, err = mudp.BuildListenerUDP(s.listenerTCP.Addr().String())
	if err != nil {
		return fmt.Errorf("failed to start UDP listener: %w", err)
	}

	go func() {
		if err := s.http3Server.Start(); err != nil {
			s.logger.Error("failed to start HTTP3 server", "error", err)
		}
	}()

	if err := s.register(); err != nil {
		return fmt.Errorf("failed to register the HTTP server: %w", err)
	}

	s.started = true

	return nil
}

// Stop will stop the HTTP server(s).
func (s *Server) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	errChan := make(chan error)
	defer close(errChan)

	s.logger.Debug("Stopping")

	if err := s.deregister(); err != nil {
		return err
	}

	c := 1
	if s.config.HTTP3 {
		c++

		go func() {
			errChan <- s.http3Server.Stop(ctx)

			// Listener most likely already closed, just as a double check.
			_ = s.listenerUDP.Close() //nolint:errcheck
		}()
	}

	type stopper interface {
		Stop(ctx context.Context) error
	}

	go func(srv stopper, l net.Listener) {
		errChan <- srv.Stop(ctx)

		// Listener most likely already closed, just as a double check.
		_ = l.Close() //nolint:errcheck
	}(s.httpServer, s.listenerTCP)

	var err error

	for i := 0; i < c; i++ {
		if nerr := <-errChan; nerr != nil {
			err = nerr
		}
	}

	s.started = false

	return err
}

// Config returns a copy of the internal config.
func (s *Server) Config() Config {
	return *s.config
}

// AddHandler adds a handler for later registration.
func (s *Server) AddHandler(handler server.RegistrationFunc) {
	s.config.OptHandlers = append(s.config.OptHandlers, handler)
}

// Register executes a registration function on the entrypoint.
func (s *Server) Register(register server.RegistrationFunc) {
	register(s)
}

// Address returns the address the entrypoint is listening on.
func (s *Server) Address() string {
	if s.listenerTCP != nil {
		return s.listenerTCP.Addr().String()
	}

	return s.config.Address
}

// Transport returns the client transport to use.
func (s *Server) Transport() string {
	//nolint:gocritic
	if s.config.H2C {
		return "h2c"
	} else if s.config.HTTP3 {
		return "http3"
	} else if !s.config.Insecure {
		return "https"
	}

	return "http"
}

// EntrypointID returns the id (configured name) of this entrypoint in the registry.
func (s *Server) EntrypointID() string {
	return s.config.Name
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

// Router returns the router used by the HTTP server.
// You can use this to register extra handlers, or mount additional routers.
func (s *Server) Router() *Router {
	return s.router
}

func (s *Server) setupTLS() (*mtls.Config, error) {
	// TLS already provided or not needed.
	if s.config.TLS != nil || s.config.Insecure {
		return s.config.TLS, nil
	}

	var (
		config *tls.Config
		err    error
	)

	// Generate self signed cert.
	if s.config.TLS == nil {
		config, err = mtls.GenTLSConfig(s.config.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to generate self signed certificate: %w", err)
		}
	}

	return &mtls.Config{Config: config}, nil
}

//nolint:unparam
func (s *Server) getEndpoints() ([]*registry.Endpoint, error) {
	routes := s.router.Routes()
	result := make([]*registry.Endpoint, len(routes))

	for _, r := range routes {
		s.logger.Trace("found endpoint", slog.String("name", r))

		result = append(result, &registry.Endpoint{
			Name:     r,
			Metadata: map[string]string{"stream": "true"},
		})
	}

	return result, nil
}

func (s *Server) registryService() (*registry.Service, error) {
	node := &registry.Node{
		ID:        s.EntrypointID(),
		Address:   s.Address(),
		Transport: s.Transport(),
		Metadata:  make(map[string]string),
	}

	eps, err := s.getEndpoints()
	if err != nil {
		return nil, err
	}

	return &registry.Service{
		Name:      s.registry.ServiceName(),
		Version:   s.registry.ServiceVersion(),
		Nodes:     []*registry.Node{node},
		Endpoints: eps,
	}, nil
}

func (s *Server) register() error {
	rService, err := s.registryService()
	if err != nil {
		return err
	}

	return s.registry.Register(rService)
}

func (s *Server) deregister() error {
	rService, err := s.registryService()
	if err != nil {
		return err
	}

	return s.registry.Deregister(rService)
}
