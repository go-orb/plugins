// Package https contains the h2c transport for the orb client.
package https

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb_transport/basehttp"
)

// Name is the transports name.
const Name = "https"

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
			Transport: &http.Transport{
				MaxIdleConns:        cfg.PoolHosts * cfg.PoolSize,
				MaxIdleConnsPerHost: cfg.PoolSize,
				MaxConnsPerHost:     cfg.PoolHosts,
				IdleConnTimeout:     cfg.PoolTTL,
				ForceAttemptHTTP2:   false,
				DisableKeepAlives:   false,
				DialContext: (&net.Dialer{
					Timeout:   cfg.DialTimeout,
					KeepAlive: 15 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: cfg.DialTimeout,
				TLSClientConfig: &tls.Config{
					//nolint:gosec
					InsecureSkipVerify: true,
				},
			},
		},
	)
}
