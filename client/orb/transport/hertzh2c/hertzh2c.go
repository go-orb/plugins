// Package hertzh2c contains the hertz h2c transport for the orb client.
package hertzh2c

import (
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"

	hclient "github.com/cloudwego/hertz/pkg/app/client"
	"github.com/go-orb/plugins/client/orb/transport/basehertz"

	"github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
)

// Name is the transports name.
const Name = "hertzh2c"

func init() {
	orb.RegisterTransport(Name, NewTransport)
}

// NewTransport creates a new hertz http transport for the orb client.
func NewTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return basehertz.NewTransport(
		Name,
		logger,
		"http",
		func() (*hclient.Client, error) {
			c, err := hclient.NewClient(
				hclient.WithNoDefaultUserAgentHeader(true),
				hclient.WithMaxConnsPerHost(cfg.PoolSize),
			)
			if err != nil {
				return nil, err
			}

			c.SetClientFactory(factory.NewClientFactory(config.WithAllowHTTP(true)))

			return c, nil
		},
	)
}
