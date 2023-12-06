package http

// Copied and adapted from Traefik
// https://github.com/traefik/traefik/blob/master/pkg/server/server_entrypoint_tcp_http3.go

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// Errors returned by the HTTP3 server.
var (
	ErrNoTLSConfig = errors.New("no tls config")
)

type http3server struct {
	*http3.Server

	s *ServerHTTP
}

func (s *ServerHTTP) newHTTP3Server() *http3server {
	h3 := http3server{
		s: s,
	}

	h3.Server = &http3.Server{
		Handler:        s,
		TLSConfig:      s.Config.TLS.Config,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,

		// TODO(davincible): remove this config when draft versions are no longer supported (we have no need to support drafts)
		QuicConfig: &quic.Config{
			MaxIncomingStreams: int64(s.Config.MaxConcurrentStreams),
			Versions:           []quic.VersionNumber{quic.Version1, quic.Version2},
		},
	}

	return &h3
}

func (h3 *http3server) Start() error {
	h3ln, err := quic.ListenEarly(h3.s.listenerUDP, http3.ConfigureTLSConfig(h3.s.Config.TLS.Config), &quic.Config{
		Allow0RTT: true,
		RequireAddressValidation: func(clientAddr net.Addr) bool {
			return atomic.LoadInt64(&h3.s.activeRequests) > int64(h3.s.Config.MaxConcurrentStreams)
		},
	})
	if err != nil {
		return err
	}

	if err := h3.ServeListener(h3ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (h3 *http3server) Stop(_ context.Context) error {
	// TODO(davincible): use h3.CloseGracefully() when available.
	return h3.Close()
}
