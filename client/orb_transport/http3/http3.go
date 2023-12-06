// Package http3 contains the http3 transport for the orb client.
package http3

import (
	"crypto/tls"
	"net/http"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb_transport/basehttp"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// Name is the transports name.
const Name = "http3"

func init() {
	orb.Transports.Register(Name, NewTransport)
}

// NewTransport creates a new https transport for the orb client.
func NewTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return basehttp.NewTransport(
		Name,
		logger,
		"https",
		&http.Client{
			Timeout: cfg.ConnectionTimeout,
			Transport: &http3.RoundTripper{
				QuicConfig: &quic.Config{
					MaxIncomingStreams:         int64(cfg.PoolSize),
					MaxIncomingUniStreams:      int64(cfg.PoolSize),
					MaxStreamReceiveWindow:     3 * (1 << 20),   // 3 MB
					MaxConnectionReceiveWindow: 4.5 * (1 << 20), // 4.5 MB
				},
				TLSClientConfig: &tls.Config{
					//nolint:gosec
					InsecureSkipVerify: true,
				},
			},
		},
	)
}
