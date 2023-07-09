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
	"fmt"
	"net"

	"golang.org/x/exp/slog"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/addr"
	mtls "github.com/go-orb/go-orb/util/tls"

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
	Config Config
	Logger log.Logger

	// router is not exported as you can't change the router after server creation.
	// The router here is merely a reference to the router that is used in the servers
	// themselves. You can fetch the router with the getter, and register handlers,
	// or mount other routers.
	router router.Router
	codecs map[string]codecs.Marshaler

	httpServer  *httpServer
	http3Server *http3server

	listenerUDP net.PacketConn
	listenerTCP net.Listener

	started bool
}

// ProvideServerHTTP creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports. One entrypoint
// can serve a HTTP1, HTTP2 and HTTP3 server. If you enable HTTP3 it will listen
// on both TCP and UDP on the same port.
func ProvideServerHTTP(
	_ types.ServiceName,
	logger log.Logger,
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

	logger, err = logger.WithComponent(server.ComponentType, Plugin, cfg.Logger.Plugin, cfg.Logger.Level)
	if err != nil {
		return nil, fmt.Errorf("create %s (http) component logger: %w", cfg.Name, err)
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	entrypoint := ServerHTTP{
		Config: cfg,
		Logger: logger,
		codecs: codecs,
		router: router,
	}

	entrypoint.Config.TLS, err = entrypoint.setupTLS()
	if err != nil {
		return nil, err
	}

	entrypoint.httpServer, err = entrypoint.newHTTPServer(router)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP server: %w", err)
	}

	if !entrypoint.Config.HTTP3 {
		return &entrypoint, nil
	}

	entrypoint.http3Server, err = entrypoint.newHTTP3Server()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP3 server: %w", err)
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
			s.Logger.Error("Failed to start HTTP server", err)
		}
	}()

	if !s.Config.HTTP3 {
		s.started = true
		return nil
	}

	// Listen on the same UDP port as TCP for HTTP3
	s.listenerUDP, err = mudp.BuildListenerUDP(s.Config.Address)
	if err != nil {
		return fmt.Errorf("failed to start UDP listener: %w", err)
	}

	go func() {
		if err := s.http3Server.Start(s.listenerUDP); err != nil {
			s.Logger.Error("Failed to start HTTP3 server", err)
		}
	}()

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
		Stop(context.Context) error
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
