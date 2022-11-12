package http

// Copied and adapted from Traefik
// https://github.com/traefik/traefik/blob/master/pkg/server/server_entrypoint_tcp_http3.go

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"

	mip "github.com/go-micro/plugins/server/http/utils/ip"

	"github.com/lucas-clemente/quic-go/http3"
)

// Errors returned by the HTTP3 server.
var (
	ErrNoTLSConfig = errors.New("no tls config")
)

type http3server struct {
	*http3.Server

	lock   sync.RWMutex
	getter func(info *tls.ClientHelloInfo) (*tls.Config, error)
}

func (s *ServerHTTP) newHTTP3Server() (*http3server, error) {
	port, err := mip.ParsePort(s.Config.Address)
	if err != nil {
		return nil, err
	}

	h3 := http3server{
		getter: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			return s.Config.TLS, nil
			// return nil, ErrNoTLSConfig
		},
	}

	h2 := s.httpServer.Server

	h3.Server = &http3.Server{
		Addr:      s.Config.Address,
		Port:      port,
		Handler:   h2.Handler,
		TLSConfig: h3.prepareTLSConfig(s.Config.TLS),
	}

	previousHandler := h2.Handler

	h2.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := h3.SetQuicHeaders(rw.Header()); err != nil {
			s.Logger.Error("Failed to set HTTP3 headers", err)
		}

		previousHandler.ServeHTTP(rw, req)
	})

	return &h3, nil
}

func (h3 *http3server) Start(l net.PacketConn) error {
	if err := h3.Serve(l); err != nil && errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (h3 *http3server) Stop(_ context.Context) error {
	// TODO: use e.Server.CloseGracefully() when available.
	return h3.Close()
}

func (h3 *http3server) getGetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	h3.lock.RLock()
	defer h3.lock.RUnlock()

	return h3.getter(info)
}

func (h3 *http3server) prepareTLSConfig(c *tls.Config) *tls.Config {
	if c == nil {
		c = new(tls.Config)
	}

	c.GetConfigForClient = h3.getGetConfigForClient

	return c
}
