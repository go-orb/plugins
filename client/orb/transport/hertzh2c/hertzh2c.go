// Package hertzhttp contains the hertz http transport for the orb client.
package hertzh2c

import (
	"context"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/client/orb"

	hclient "github.com/cloudwego/hertz/pkg/app/client"
	"github.com/go-orb/plugins/client/orb/transport/basehertz"
)

// Name is the transports name.
const Name = "hertzh2c"

func init() {
	orb.Transports.Register(Name, NewTransport)
}

// NewTransport creates a new hertz http transport for the orb client.
func NewTransport(logger log.Logger) (orb.TransportType, error) {
	return basehertz.NewTransport(
		Name,
		logger,
		"http",
		func(ctx context.Context, opts *client.CallOptions) (*hclient.Client, error) {
			return hclient.NewClient()
		},
	)
}
