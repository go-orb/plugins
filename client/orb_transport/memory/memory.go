// Package memory implements a go-orb/plugins/client/orb compatible memory transport.
package memory

import (
	"context"
	"maps"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"

	"github.com/go-orb/plugins/client/orb"
)

// Name is the name of this transport.
const Name = "memory"

func init() {
	orb.RegisterTransport(Name, NewTransport)
}

// Transport is a go-orb/plugins/client/orb compatible memory transport.
type Transport struct {
	logger log.Logger
}

// Start starts the transport.
func (t *Transport) Start() error {
	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	return nil
}

// Name returns the name of this transport.
func (t *Transport) Name() string {
	return Name
}

// Request does the actual rpc request to the server.
func (t *Transport) Request(ctx context.Context, infos client.RequestInfos, req any, result any, opts *client.CallOptions) error {
	server, err := client.ResolveMemoryServer(infos.Service)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	md := opts.Metadata
	if md == nil {
		md = map[string]string{}
	}

	// Set the connection timeout
	ctx, cancel := context.WithTimeout(ctx, opts.ConnectionTimeout)
	defer cancel()

	ctx, outMd := metadata.WithOutgoing(ctx)
	ctx, inMd := metadata.WithIncoming(ctx)

	maps.Copy(inMd, md)
	inMd["Content-Type"] = opts.ContentType

	err = server.Request(ctx, infos, req, result, opts)

	// Retrieve metadata from memory.
	if opts.ResponseMetadata != nil {
		for k, v := range outMd {
			opts.ResponseMetadata[k] = v
		}
	}

	return err
}

// Stream opens a bidirectional stream to the memory server.
func (t *Transport) Stream(
	ctx context.Context, infos client.RequestInfos,
	opts *client.CallOptions,
) (client.StreamIface[any, any], error) {
	server, err := client.ResolveMemoryServer(infos.Service)
	if err != nil {
		return nil, orberrors.ErrInternalServerError.Wrap(err)
	}

	md := opts.Metadata
	if md == nil {
		md = map[string]string{}
	}

	// Set the connection timeout
	ctx, _ = context.WithTimeout(ctx, opts.ConnectionTimeout)

	ctx, outMd := metadata.WithOutgoing(ctx)
	ctx, inMd := metadata.WithIncoming(ctx)

	maps.Copy(inMd, md)
	inMd["Content-Type"] = opts.ContentType

	stream, err := server.Stream(ctx, infos, opts)

	// Retrieve metadata from memory.
	if opts.ResponseMetadata != nil {
		for k, v := range outMd {
			opts.ResponseMetadata[k] = v
		}
	}

	return stream, err
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger, _ *orb.Config) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		logger: logger,
	}}, nil
}
