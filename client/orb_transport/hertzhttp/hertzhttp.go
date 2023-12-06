// Package hertzhttp contains the hertz http transport for the orb client.
package hertzhttp

import (
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"

	hclient "github.com/cloudwego/hertz/pkg/app/client"
	"github.com/go-orb/plugins/client/orb_transport/basehertz"
)

// Name is the transports name.
const Name = "hertzhttp"

func init() {
	orb.Transports.Register(Name, NewTransport)
}

// NewTransport creates a new hertz http transport for the orb client.
func NewTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return basehertz.NewTransport(
		Name,
		logger,
		"http",
		func() (*hclient.Client, error) {
			return hclient.NewClient(
				hclient.WithNoDefaultUserAgentHeader(true),
				hclient.WithMaxConnsPerHost(cfg.PoolSize),
			)
		},
	)
}
