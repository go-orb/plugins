// Package basehttp contains the base http transport for the orb client,
// every http transport uses this as base.!
package basehttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
)

//nolint:gochecknoglobals
var stdHeaders = []string{"Content-Length", "Content-Type", "Date", "Server"}

var _ (orb.Transport) = (*Transport)(nil)

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	name    string
	logger  log.Logger
	hclient *http.Client
	scheme  string
}

// Start starts the transport.
func (t *Transport) Start() error {
	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	if t.hclient != nil {
		t.hclient.CloseIdleConnections()
	}

	return nil
}

// Name returns the name of this transport.
func (t *Transport) Name() string {
	return t.name
}

// Request does the actual rpc call to the server.
//
//nolint:funlen,gocyclo
func (t *Transport) Request(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
	codec, err := codecs.GetEncoder(opts.ContentType, req.Req())
	if err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	// Encode the request into a *bytes.Buffer{}.
	buff := bytes.NewBuffer(nil)
	if err := codec.NewEncoder(buff).Encode(req.Req()); err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	node, err := req.Node(ctx, opts)
	if err != nil {
		return orberrors.From(err)
	}

	// Set the connection timeout
	ctx, cancel := context.WithTimeout(ctx, opts.ConnectionTimeout)
	defer cancel()

	// Create a net/http request.
	hReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s://%s%s", t.scheme, node.Address, req.Endpoint()),
		buff,
	)
	if err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	// Set headers.
	hReq.Header.Set("Content-Type", opts.ContentType)
	hReq.Header.Set("Accept", opts.ContentType)

	// Set metadata key=value to request headers.
	md, ok := metadata.Outgoing(ctx)
	if ok {
		for name, value := range md {
			hReq.Header.Set(name, value)
		}
	}

	// Run the request.
	resp, err := t.hclient.Do(hReq)
	if err != nil {
		return orberrors.From(err)
	}

	buff = bytes.NewBuffer(nil)

	_, err = buff.ReadFrom(resp.Body)
	if err != nil && !errors.Is(err, io.EOF) {
		return orberrors.From(err)
	}

	// Close the request body.
	if err := resp.Body.Close(); err != nil {
		return orberrors.From(err)
	}

	if opts.ResponseMetadata != nil {
		md := opts.ResponseMetadata

		// Copy headers to opts.Header
		for k, v := range resp.Header {
			// Skip std headers.
			if slices.Contains(stdHeaders, k) {
				continue
			}

			if len(v) == 1 {
				md[strings.ToLower(k)] = v[0]
			} else {
				md[strings.ToLower(k)] = v[0]

				for i := 1; i < len(v); i++ {
					md[strings.ToLower(k)+"-"+strconv.Itoa(i)] = v[i]
				}
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		return orberrors.HTTP(resp.StatusCode)
	}

	// Decode the response into `result`.
	err = codec.NewDecoder(buff).Decode(result)
	if err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	return nil
}

// NewTransport creates a Transport with a custom http.Client.
func NewTransport(
	name string, logger log.Logger, scheme string, hclient *http.Client,
) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		name:    name,
		logger:  logger,
		scheme:  scheme,
		hclient: hclient,
	},
	}, nil
}
