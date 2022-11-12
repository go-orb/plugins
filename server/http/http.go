package http

// Inspired by and adapted from Traefik
// https://github.com/traefik/traefik/blob/master/pkg/server/server_entrypoint_tcp_http3.go

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-micro/plugins/server/http/router/router"

	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Errors.
var (
	ErrRouterHandlerInterface = errors.New("router does not implement http.Handler interface")
	ErrNoTLS                  = errors.New("no TLS config provided")
)

// httpServer is a server with Start and Stop methods.
type httpServer struct {
	Server *http.Server
}

func (e *ServerHTTP) newHTTPServer(router router.Router) (*httpServer, error) {
	handler, ok := router.(http.Handler)
	if !ok {
		return nil, ErrRouterHandlerInterface
	}

	if e.Config.H2C {
		handler = h2c.NewHandler(handler, &http2.Server{
			MaxConcurrentStreams: uint32(e.Config.MaxConcurrentStreams),
		})
	}

	server := http.Server{
		Handler:           handler,
		ReadTimeout:       e.Config.ReadTimeout,
		WriteTimeout:      e.Config.WriteTimeout,
		IdleTimeout:       e.Config.IdleTimeout,
		ReadHeaderTimeout: time.Second * 4,
		// TODO: do we need to set this? would be nice but doesn't take interface
		// ErrorLog:          httpServerLogger,
	}

	if !e.Config.Insecure && e.Config.TLS != nil {
		server.TLSConfig = e.Config.TLS
	} else if !e.Config.Insecure && e.Config.TLS == nil {
		return nil, ErrNoTLS
	}

	if e.Config.HTTP2 && !strings.Contains(os.Getenv("GODEBUG"), "http2server=0") {
		if e.Config.TLS != nil {
			e.Config.TLS.NextProtos = append([]string{"h2"}, e.Config.TLS.NextProtos...)
		}

		h2 := http2.Server{
			MaxConcurrentStreams: uint32(e.Config.MaxConcurrentStreams),
			NewWriteScheduler:    func() http2.WriteScheduler { return http2.NewPriorityWriteScheduler(nil) },
		}

		if err := http2.ConfigureServer(&server, &h2); err != nil {
			return nil, errors.Wrap(err, "configure HTTP/2 server")
		}
	}

	return &httpServer{Server: &server}, nil
}

func (s *httpServer) Start(l net.Listener) error {
	if err := s.Server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *httpServer) Stop(ctx context.Context) error {
	if err := s.Server.Shutdown(ctx); err != nil && errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return nil
}
