// Package https contains the h2c transport for the orb client.
package https

import (
	"crypto/tls"
	"net/http"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb/transport/basehttp"
)

// Name is the transports name.
const Name = "https"

func init() {
	orb.Transports.Register(Name, NewTransportHTTPS)
}

// NewTransportHTTPS creates a new https transport for the orb. client.
func NewTransportHTTPS(logger log.Logger) (orb.TransportType, error) {
	return basehttp.NewTransport(
		Name,
		logger,
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					//nolint:gosec
					InsecureSkipVerify: true,
				},
			},
		},
		"https",
	)
}
