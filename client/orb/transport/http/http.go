// Package http contains the http transport for the orb client.
package http

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-orb/go-orb/client"
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
	return basehttp.NewTransport(
		Name,
		logger,
		"http",
		func(ctx context.Context, opts *client.CallOptions) (*http.Client, error) {
			return &http.Client{
				Timeout: opts.ConnectionTimeout,
				Transport: &http.Transport{
					MaxIdleConns:          opts.PoolSize,
					MaxIdleConnsPerHost:   opts.PoolHosts + 1,
					MaxConnsPerHost:       opts.PoolHosts,
					IdleConnTimeout:       opts.PoolTTL,
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     false,
					DisableKeepAlives:     false,
					DialContext: (&net.Dialer{
						Timeout:   opts.DialTimeout,
						KeepAlive: 30 * time.Second,
						DualStack: false,
					}).DialContext,
				},
			}, nil
		},
	)
}
