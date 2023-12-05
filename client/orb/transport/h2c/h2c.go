// Package h2c contains the h2c transport for the orb client.
package h2c

import (
	"net"
	"net/http"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb/transport/basehttp"
)

// Name is the transports name.
const Name = "h2c"

func init() {
	orb.Transports.Register(Name, NewTransport)
}

// NewTransport creates a new https transport for the orb client.
func NewTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return basehttp.NewTransport(
		Name,
		logger,
		"http",
		&http.Client{
			Timeout: cfg.ConnectionTimeout,
			Transport: &http.Transport{
				MaxIdleConns:          cfg.PoolHosts * cfg.PoolSize,
				MaxIdleConnsPerHost:   cfg.PoolSize,
				MaxConnsPerHost:       cfg.PoolSize + 1,
				IdleConnTimeout:       cfg.PoolTTL,
				ExpectContinueTimeout: 1 * time.Second,
				ForceAttemptHTTP2:     false,
				DisableKeepAlives:     false,
				Dial: (&net.Dialer{
					Timeout:   cfg.DialTimeout,
					KeepAlive: 15 * time.Second,
					DualStack: false,
				}).Dial,
			},
		},
	)
}
