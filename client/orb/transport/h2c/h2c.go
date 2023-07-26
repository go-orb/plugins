// Package h2c contains the h2c transport for the orb client.
package h2c

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb/transport/basehttp"
	"golang.org/x/net/http2"
)

// Name is the transports name.
const Name = "h2c"

func init() {
	orb.Transports.Register(Name, NewTransportH2C)
}

// NewTransportH2C creates a new h2c transport for the orb. client.
func NewTransportH2C(logger log.Logger) (orb.TransportType, error) {
	return basehttp.NewTransport(
		Name,
		logger,
		&http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		},
		"http",
	)
}
