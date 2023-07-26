// Package http contains the http transport for the orb client.
package http

import (
	"net/http"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"

	"github.com/go-orb/plugins/client/orb/transport/basehttp"
)

// Name is the transports name.
const Name = "http"

func init() {
	orb.Transports.Register(Name, NewTransportHTTP)
}

// NewTransportHTTP creates a new http transport for the orb client.
func NewTransportHTTP(logger log.Logger) (orb.TransportType, error) {
	return basehttp.NewTransport(Name, logger, &http.Client{}, "http")
}
