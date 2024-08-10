package http

// Inspired by and adapted from Traefik
// https://github.com/traefik/traefik/blob/master/pkg/server/server_entrypoint_tcp_http3.go

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-orb/plugins/server/http/router"

	"log/slog"

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

func (s *ServerHTTP) newHTTPServer(router router.Router) (*httpServer, error) {
	var ok bool
	s.handler, ok = router.(http.Handler)

	if !ok {
		return nil, ErrRouterHandlerInterface
	}

	if s.Config.H2C {
		s.handler = h2c.NewHandler(s.handler, &http2.Server{
			MaxConcurrentStreams: uint32(s.Config.MaxConcurrentStreams),
		})
	}

	server := http.Server{
		Handler:           s,
		ReadTimeout:       s.Config.ReadTimeout,
		WriteTimeout:      s.Config.WriteTimeout,
		IdleTimeout:       s.Config.IdleTimeout,
		ReadHeaderTimeout: time.Second * 4,
		// TODO(davincible): do we need to set this? would be nice but doesn't take interface
		// ErrorLog:          httpServerLogger,
	}

	if !s.Config.Insecure && s.Config.TLS != nil {
		server.TLSConfig = s.Config.TLS.Config
	} else if !s.Config.Insecure && s.Config.TLS == nil {
		return nil, ErrNoTLS
	}

	if s.Config.HTTP2 && !strings.Contains(os.Getenv("GODEBUG"), "http2server=0") {
		if s.Config.TLS != nil {
			s.Config.TLS.NextProtos = append([]string{"h2"}, s.Config.TLS.NextProtos...)
		}

		h2 := http2.Server{
			MaxConcurrentStreams: uint32(s.Config.MaxConcurrentStreams),
			NewWriteScheduler:    func() http2.WriteScheduler { return http2.NewPriorityWriteScheduler(nil) },
		}

		if err := http2.ConfigureServer(&server, &h2); err != nil {
			return nil, fmt.Errorf("configure HTTP/2 server: %w", err)
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

func (s *ServerHTTP) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Server", "go-orb")

	// advertise HTTP/3, if enabled
	if s.http3Server != nil {
		// keep track of active requests for QUIC transport purposes
		atomic.AddInt64(&s.activeRequests, 1)
		defer atomic.AddInt64(&s.activeRequests, -1)

		if req.ProtoMajor < 3 {
			err := s.http3Server.SetQUICHeaders(resp.Header())
			if err != nil {
				s.Logger.Error("setting HTTP/3 Alt-Svc header", "error", err)
			}
		}
	}

	// reject very long methods; probably a mistake or an attack
	if len(req.Method) > 32 {
		s.Logger.Warn("rejecting request with long method",
			slog.String("method_trunc", req.Method[:32]),
			slog.String("remote_addr", req.RemoteAddr))
		resp.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	s.handler.ServeHTTP(resp, req)
}
