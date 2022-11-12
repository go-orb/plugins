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

	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/log"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/types"
	"go-micro.dev/v5/types/component"

	"github.com/go-micro/plugins/server/http/router/router"
	mip "github.com/go-micro/plugins/server/http/utils/ip"
	mtcp "github.com/go-micro/plugins/server/http/utils/tcp"
	mtls "github.com/go-micro/plugins/server/http/utils/tls"
	mudp "github.com/go-micro/plugins/server/http/utils/udp"
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
func ProvideServerHTTP(name string, service types.ServiceName, data types.ConfigData, logger log.Logger, c any, options ...Option) (*ServerHTTP, error) {
	var err error

	cfg, ok := c.(Config)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	cfg.ApplyOptions(options...)

	// Name needs to be explicitly set, as the config may be inherited and contain
	// a different name.
	cfg.Name = name

	cfg, err = parseFileConfig(service, data, cfg)
	if err != nil {
		return nil, err
	}

	if err = mip.ValidateAddress(cfg.Address); err != nil {
		return nil, err
	}

	router, err := cfg.NewRouter()
	if err != nil {
		return nil, fmt.Errorf("http server: create router (%s): %w", cfg.Router, err)
	}

	codecs, err := cfg.NewCodecMap()
	if err != nil {
		return nil, fmt.Errorf("http server: create codec map: %w", err)
	}

	logger, err = logger.WithComponent(server.ComponentType, Plugin, cfg.Logger.Plugin, cfg.Logger.Level)
	if err != nil {
		return nil, fmt.Errorf("create %s (http) component logger: %w", name, err)
	}

	logger = logger.With(slog.String("entrypoint", name))

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

	s.router.Use(s.Config.Middleware...)

	for _, h := range s.Config.RegistrationFuncs {
		h(s)
	}

	s.listenerTCP, err = mtcp.BuildListenerTCP(s.Config.Address, s.Config.TLS)
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

type stopper interface {
	Stop(context.Context) error
}

// Stop will stop the HTTP server(s).
func (e *ServerHTTP) Stop(ctx context.Context) error {
	if !e.started {
		return nil
	}

	errChan := make(chan error)
	defer close(errChan)

	e.Logger.Debug("Stopping all HTTP entrypoints")

	c := 1
	if e.Config.HTTP3 {
		c++

		go func() {
			errChan <- e.http3Server.Stop(ctx)

			// Listener most likely already closed, just as a double check.
			_ = e.listenerUDP.Close() //nolint:errcheck
		}()
	}

	go func(srv stopper, l net.Listener) {
		errChan <- srv.Stop(ctx)

		// Listener most likely already closed, just as a double check.
		_ = l.Close() //nolint:errcheck
	}(e.httpServer, e.listenerTCP)

	var err error

	for i := 0; i < c; i++ {
		if nerr := <-errChan; nerr != nil {
			err = nerr
		}
	}

	e.started = false

	return err
}

func (e *ServerHTTP) Register(register server.RegistrationFunc) {
	register(e)
}

func (e *ServerHTTP) String() string {
	return Plugin
}

func (e *ServerHTTP) Name() string {
	return e.Config.Name
}

func (e *ServerHTTP) Type() component.Type {
	return server.ComponentType
}

func (e *ServerHTTP) Router() router.Router {
	return e.router
}

func (e *ServerHTTP) setupTLS() (*tls.Config, error) {
	var (
		config *tls.Config
		err    error
	)

	// TLS already provided
	if e.Config.TLS != nil {
		return e.Config.TLS, nil
	}

	// Generate self signed cert
	if !e.Config.Insecure && e.Config.TLS == nil {
		config, err = mtls.GenTlSConfig(e.Config.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to generate self signed certificate: %w", err)
		}
	}

	return config, nil
}
