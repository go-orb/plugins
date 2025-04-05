package http

// Inspired by and adapted from Traefik
// https://github.com/traefik/traefik/blob/master/pkg/server/server_entrypoint_tcp_http3.go

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"log/slog"
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

func (s *Server) newHTTPServer(router *Router) (*httpServer, error) {
	s.handler = router

	server := http.Server{
		Handler:           s,
		ReadTimeout:       time.Duration(s.config.ReadTimeout),
		WriteTimeout:      time.Duration(s.config.WriteTimeout),
		IdleTimeout:       time.Duration(s.config.IdleTimeout),
		ReadHeaderTimeout: time.Second * 4,

		// TODO(davincible): do we need to set this? would be nice but doesn't take interface
		// ErrorLog:          httpServerLogger,
	}

	if !s.config.Insecure && s.config.TLS != nil {
		server.TLSConfig = s.config.TLS.Config
	} else if !s.config.Insecure && s.config.TLS == nil {
		return nil, ErrNoTLS
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

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Server", "go-orb")

	// advertise HTTP/3, if enabled
	if s.http3Server != nil {
		// keep track of active requests for QUIC transport purposes
		atomic.AddInt64(&s.activeRequests, 1)
		defer atomic.AddInt64(&s.activeRequests, -1)

		if req.ProtoMajor < 3 {
			err := s.http3Server.SetQUICHeaders(resp.Header())
			if err != nil {
				s.logger.Error("setting HTTP/3 Alt-Svc header", "error", err)
			}
		}
	}

	// reject very long methods; probably a mistake or an attack
	if len(req.Method) > 32 {
		s.logger.Warn("rejecting request with long method",
			slog.String("method_trunc", req.Method[:32]),
			slog.String("remote_addr", req.RemoteAddr))
		resp.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	s.handler.ServeHTTP(resp, req)
}
