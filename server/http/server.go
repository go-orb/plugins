// Package http provides an HTTP server implementation.
// It provides an HTTP1, HTTP2, and HTTP3 server, the first two enabled by default.
//
// One server contains multiple entrypoints, with one entrypoint being one
// address to listen on. Each entrypoint with start its own HTTP2 server, and
// optionally also an HTTP3 server. Each entrypoint can be customised individually,
// but default options are provided, and can be tweaked.
//
// The architecture is based on the Traefik server implementation.
package http

import (
	"context"
	"fmt"

	"github.com/go-micro/plugins/server/http/codec"
	"github.com/go-micro/plugins/server/http/entrypoint"
	"github.com/go-micro/plugins/server/http/router/router"

	"http-poc/logger"
)

var _ ServerHTTP = (*Server)(nil)

// TODO: check if we need to add cache

// ServerHTTP implements the HTTP server interface.
type ServerHTTP interface {
	Router() router.Router

	Start() error
	Stop(context.Context) error
	Type() string
	String() string
}

type Server struct {
	codecs codec.Codecs
	logger logger.Logger

	// TODO: check if thread safe to use with multiple servers
	router router.Router
	Config Config

	entrypoints map[string]*entrypoint.Entrypoint
}

func ProvideServerHTTP(router router.Router, codecs codec.Codecs, logger logger.Logger, options ...Option) (*Server, error) {
	s := Server{
		codecs:      codecs,
		logger:      logger,
		router:      router,
		Config:      NewConfig(options...),
		entrypoints: make(map[string]*entrypoint.Entrypoint, 1),
	}

	if err := s.createEntrypoints(); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Server) createEntrypoints() error {
	for _, e := range s.Config.Entrypoints {
		ep, err := entrypoint.NewEntrypoint(s.router, s.logger, s.Config.EntrypointDefaults, e...)
		if err != nil {
			return fmt.Errorf("server create entrypoint: %w", err)
		}

		s.entrypoints[ep.Config.Address] = ep
	}

	return nil
}

// Start will start the HTTP servers on all entrypoints.
func (s *Server) Start() error {
	for addr, entrypoint := range s.entrypoints {
		if err := entrypoint.Start(); err != nil {
			return fmt.Errorf("start entrypoint (%s): %w", addr, err)
		}
	}

	return nil
}

// Stop will stop the HTTP servers on all entrypoints and close the listners.
func (s *Server) Stop(ctx context.Context) error {
	errChan := make(chan error)

	// Stop all servers in parallel to make sure they get equal amount of time
	// to shutdown gracefully.
	for _, e := range s.entrypoints {
		go func(entrypoint *entrypoint.Entrypoint) {
			errChan <- entrypoint.Stop(ctx)
		}(e)
	}

	var err error

	for i := 0; i < len(s.entrypoints); i++ {
		if nerr := <-errChan; nerr != nil {
			err = fmt.Errorf("stop entrypoint: %w", nerr)
		}
	}

	close(errChan)

	return err
}

// Type returns the micro component type.
func (s *Server) Type() string {
	// TODO: abstract this away in a const in the core.
	return "Server"
}

// String returns the server implementation name.
func (s *Server) String() string {
	return "http"
}

// Router returns the HTTP servers' router (mux).
func (s *Server) Router() router.Router {
	return s.router
}
