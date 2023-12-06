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

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
)

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

func (t *Transport) String() string {
	return t.name
}

// NeedsCodec is always true for http based transports.
func (t *Transport) NeedsCodec() bool {
	return true
}

// Call does the actual rpc call to the server.
func (t *Transport) Call(ctx context.Context, req *client.Request[any, any], opts *client.CallOptions,
) (*client.RawResponse, error) {
	codec, err := codecs.GetEncoder(opts.ContentType, req.Request())
	if err != nil {
		return nil, orberrors.ErrBadRequest.Wrap(err)
	}

	// Encode the request into a *bytes.Buffer{}.
	buff := bytes.NewBuffer(nil)
	if err := codec.NewEncoder(buff).Encode(req.Request()); err != nil {
		return nil, orberrors.ErrBadRequest.Wrap(err)
	}

	node, err := req.Node(ctx, opts)
	if err != nil {
		return nil, orberrors.From(err)
	}

	// Set the connection timeout
	ctx, cancel := context.WithTimeout(ctx, opts.ConnectionTimeout)
	defer cancel()

	// Create a net/http request.
	hReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s://%s/%s", t.scheme, node.Address, req.Endpoint()),
		buff,
	)
	if err != nil {
		return nil, orberrors.ErrBadRequest.Wrap(err)
	}

	// Set headers.
	hReq.Header.Set("Content-Type", opts.ContentType)
	hReq.Header.Set("Accept", opts.ContentType)

	// Set metadata key=value to request headers.
	// TODO(jochumdev): Should we only allow a list of known headers?
	md, ok := metadata.From(ctx)
	if ok {
		for name, value := range md {
			hReq.Header.Set(name, value)
		}
	}

	return t.call2(node, opts, req, hReq)
}

func (t *Transport) call2(node *registry.Node, opts *client.CallOptions, req *client.Request[any, any], hReq *http.Request,
) (*client.RawResponse, error) {
	// Run the request.
	resp, err := t.hclient.Do(hReq)
	if err != nil {
		return nil, orberrors.From(err)
	}

	buff := bytes.NewBuffer(nil)

	_, err = buff.ReadFrom(resp.Body)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, orberrors.From(err)
	}

	// Close the request body.
	if err := resp.Body.Close(); err != nil {
		return nil, orberrors.From(err)
	}

	// Create a Response{} and fill it.
	res := &client.RawResponse{
		ContentType: resp.Header.Get("Content-Type"),
		Body:        buff,
		Headers:     make(map[string][]string),
	}

	// Copy headers to the RawResponse if wanted.
	if opts.Headers {
		for k, v := range resp.Header {
			res.Headers[k] = v
		}
	}

	if resp.StatusCode != http.StatusOK {
		return res, orberrors.NewHTTP(resp.StatusCode)
	}

	return res, nil
}

// CallNoCodec is a noop for http based transports.
func (t *Transport) CallNoCodec(_ context.Context, _ *client.Request[any, any], _ any, _ *client.CallOptions) error {
	return orberrors.ErrInternalServerError
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
