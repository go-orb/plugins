// Package https contains the h2c transport for the orb client.
package https

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb/transport/basehttp"
)

// Name is the transports name.
const Name = "https"

func init() {
	orb.Transports.Register(Name, NewTransportHTTPS)
}

// NewTransportH2C creates a new h2c transport for the orb. client.
func NewTransportHTTPS(logger log.Logger) (orb.TransportType, error) {
	return basehttp.NewTransport(
		Name,
		logger,
		"https",
		func(ctx context.Context, opts *client.CallOptions) (*http.Client, error) {
			return &http.Client{
				Timeout: opts.RequestTimeout,
				Transport: &http.Transport{
					MaxIdleConns:        opts.PoolHosts * opts.PoolSize,
					MaxIdleConnsPerHost: opts.PoolSize,
					MaxConnsPerHost:     opts.PoolHosts,
					IdleConnTimeout:     opts.PoolTTL,
					Dial: (&net.Dialer{
						Timeout: opts.DialTimeout,
					}).Dial,
					TLSHandshakeTimeout: opts.DialTimeout,
					TLSClientConfig: &tls.Config{
						//nolint:gosec
						InsecureSkipVerify: true,
					},
				},
			}, nil
		},
	)
}
