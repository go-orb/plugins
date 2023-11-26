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

	"golang.org/x/exp/slog"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	mtls "github.com/go-orb/go-orb/util/tls"
	"github.com/google/uuid"

	"github.com/go-orb/plugins/server/http/router"
	mtcp "github.com/go-orb/plugins/server/http/utils/tcp"
	mudp "github.com/go-orb/plugins/server/http/utils/udp"
)

var _ server.Entrypoint = (*ServerHTTP)(nil)

// Plugin is the plugin name.
const Plugin = "http"

// ServerHTTP represents a listener on one address. You can create multiple
// entrypoints for multiple addresses and ports. This is e.g. useful if you
// want to listen on multiple interfaces, or multiple ports in parallel, even
// with the same handler.
type ServerHTTP struct {
	Config   Config
	Logger   log.Logger
	Registry registry.Type

	// entrypointID is the entrypointID (uuid) of this entrypoint in the registry.
	entrypointID string

	// router is not exported as you can't change the router after server creation.
	// The router here is merely a reference to the router that is used in the servers
	// themselves. You can fetch the router with the getter, and register handlers,
	// or mount other routers.
	router  router.Router
	handler http.Handler
	codecs  map[string]codecs.Marshaler

	httpServer  *httpServer
	http3Server *http3server

	listenerUDP net.PacketConn
	listenerTCP net.Listener

	started bool

	activeRequests int64 // accessed atomically
}

// ProvideServerHTTP creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports. One entrypoint
// can serve a HTTP1, HTTP2 and HTTP3 server. If you enable HTTP3 it will listen
// on both TCP and UDP on the same port.
func ProvideServerHTTP(
	_ types.ServiceName,
	logger log.Logger,
	reg registry.Type,
	cfg Config,
	options ...Option,
) (*ServerHTTP, error) {
	cfg.ApplyOptions(options...)

	var err error

	cfg.Address, err = addr.GetAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("http validate addr '%s': %w", cfg.Address, err)
	}

	if err := addr.ValidateAddress(cfg.Address); err != nil {
		return nil, err
	}

	router, err := cfg.NewRouter()
	if err != nil {
		return nil, fmt.Errorf("create router (%s): %w", cfg.Router, err)
	}

	codecs, err := cfg.NewCodecMap()
	if err != nil {
		return nil, fmt.Errorf("create codec map: %w", err)
	}

	logger = logger.With(slog.String("component", server.ComponentType), slog.String("plugin", Plugin), slog.String("entrypoint", cfg.Name))

	entrypoint := ServerHTTP{
		Config:   cfg,
		Logger:   logger,
		Registry: reg,
		codecs:   codecs,
		router:   router,
	}

	entrypoint.Config.TLS, err = entrypoint.setupTLS()
	if err != nil {
		return nil, err
	}

	entrypoint.httpServer, err = entrypoint.newHTTPServer(router)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP server: %w", err)
	}

	if entrypoint.Config.HTTP3 {
		entrypoint.http3Server = entrypoint.newHTTP3Server()
	}

	return &entrypoint, nil
}

// Start will create the listeners and start the server on the entrypoint.
func (s *ServerHTTP) Start() error {
	if s.started {
		return nil
	}

	var err error

	s.Logger.Debug("Starting all HTTP entrypoints")

	for _, middleware := range s.Config.Middleware {
		s.router.Use(middleware)
	}

	for _, h := range s.Config.HandlerRegistrations {
		h(s)
	}

	var tlsConfig *tls.Config

	if s.Config.TLS != nil {
		tlsConfig = s.Config.TLS.Config
	}

	s.listenerTCP, err = mtcp.BuildListenerTCP(s.Config.Address, tlsConfig)
	if err != nil {
		return err
	}

	go func() {
		if err = s.httpServer.Start(s.listenerTCP); err != nil {
			s.Logger.Error("failed to start HTTP server: %w", err)
		}
	}()

	if !s.Config.HTTP3 {
		if err := s.register(); err != nil {
			return fmt.Errorf("failed to register the HTTP server: %w", err)
		}

		s.started = true

		return nil
	}

	// Listen on the same UDP port as TCP for HTTP3
	s.listenerUDP, err = mudp.BuildListenerUDP(s.Config.Address)
	if err != nil {
		return fmt.Errorf("failed to start UDP listener: %w", err)
	}

	go func() {
		if err := s.http3Server.Start(); err != nil {
			s.Logger.Error("failed to start HTTP3 server", "error", err)
		}
	}()

	if err := s.register(); err != nil {
		return fmt.Errorf("failed to register the HTTP server: %w", err)
	}

	s.started = true

	return nil
}

// Stop will stop the HTTP server(s).
func (s *ServerHTTP) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	errChan := make(chan error)
	defer close(errChan)

	s.Logger.Debug("Stopping all HTTP entrypoints")

	if err := s.deregister(); err != nil {
		return err
	}

	c := 1
	if s.Config.HTTP3 {
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

// Register executes a registration function on the entrypoint.
func (s *ServerHTTP) Register(register server.RegistrationFunc) {
	register(s)
}

// Address returns the address the entrypoint is listening on.
func (s *ServerHTTP) Address() string {
	if s.listenerTCP != nil {
		return s.listenerTCP.Addr().String()
	}

	return s.Config.Address
}

// Transport returns the client transport to use.
func (s *ServerHTTP) Transport() string {
	//nolint:gocritic
	if s.Config.H2C {
		return "h2c"
	} else if s.Config.HTTP3 {
		return "http3"
	} else if !s.Config.Insecure {
		return "https"
	}

	return "http"
}

// EntrypointID returns the id (uuid) of this entrypoint in the registry.
func (s *ServerHTTP) EntrypointID() string {
	if s.entrypointID != "" {
		return s.entrypointID
	}

	s.entrypointID = fmt.Sprintf("%s-%s", s.Registry.ServiceName(), uuid.New().String())

	return s.entrypointID
}

// String returns the entrypoint type; http.
func (s *ServerHTTP) String() string {
	return Plugin
}

// Name returns the entrypoint name.
func (s *ServerHTTP) Name() string {
	return s.Config.Name
}

// Type returns the component type.
func (s *ServerHTTP) Type() string {
	return server.ComponentType
}

// Router returns the router used by the HTTP server.
// You can use this to register extra handlers, or mount additional routers.
func (s *ServerHTTP) Router() router.Router {
	return s.router
}

func (s *ServerHTTP) setupTLS() (*mtls.Config, error) {
	// TLS already provided or not needed.
	if s.Config.TLS != nil || s.Config.Insecure {
		return s.Config.TLS, nil
	}

	var (
		config *tls.Config
		err    error
	)

	// Generate self signed cert.
	if s.Config.TLS == nil {
		config, err = mtls.GenTLSConfig(s.Config.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to generate self signed certificate: %w", err)
		}
	}

	return &mtls.Config{Config: config}, nil
}

func (s *ServerHTTP) getEndpoints() ([]*registry.Endpoint, error) {
	router, ok := s.router.(router.Routes)
	if !ok {
		return nil, errors.New("incompatible router")
	}

	routes := router.Routes()
	result := make([]*registry.Endpoint, len(routes))

	for _, r := range routes {
		s.Logger.Trace("found endpoint", slog.String("name", r.Pattern[1:]))

		result = append(result, &registry.Endpoint{
			Name:     r.Pattern[1:],
			Metadata: map[string]string{"stream": "true"},
		})

		if len(r.SubRoutes) > 0 {
			for _, sr := range r.SubRoutes {
				s.Logger.Trace("found sub endpoint", slog.String("name", sr.Pattern[1:]))

				result = append(result, &registry.Endpoint{
					Name:     sr.Pattern[1:],
					Metadata: map[string]string{"stream": "true"},
				})
			}
		}
	}

	return result, nil
}

func (s *ServerHTTP) registryService() (*registry.Service, error) {
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
		Name:      s.Registry.ServiceName(),
		Version:   s.Registry.ServiceVersion(),
		Nodes:     []*registry.Node{node},
		Endpoints: eps,
	}, nil
}

func (s *ServerHTTP) register() error {
	rService, err := s.registryService()
	if err != nil {
		return err
	}

	return s.Registry.Register(rService)
}

func (s *ServerHTTP) deregister() error {
	rService, err := s.registryService()
	if err != nil {
		return err
	}

	return s.Registry.Deregister(rService)
}
