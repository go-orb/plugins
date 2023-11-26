// Package basehttp contains the base http transport for the orb client,
// every http transport uses this as base.!
package basehttp

import (
	"bytes"
	"context"
	"fmt"
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

// TransportClientCreator is a factory for a client transport.
type TransportClientCreator func(ctx context.Context, opts *client.CallOptions) (*http.Client, error)

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	name          string
	logger        log.Logger
	clientCreator TransportClientCreator
	hclient       *http.Client
	scheme        string
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
	codec, err := codecs.GetMime(opts.ContentType)
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

	t.logger.Trace(
		"Making a request", "url", fmt.Sprintf("%s://%s/%s", node.Transport, node.Address, req.Endpoint()), "content-type", opts.ContentType,
	)

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

	// Get the client
	if t.hclient == nil {
		hclient, err := t.clientCreator(ctx, opts)
		if err != nil {
			return nil, err
		}

		t.hclient = hclient
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

	// Read the whole body into a []byte slice.
	buff := bytes.NewBuffer(nil)
	_, err = buff.ReadFrom(resp.Body)

	if err != nil {
		return nil, orberrors.From(err)
	}

	// Tell the client and server we are done reading.
	if err = resp.Body.Close(); err != nil {
		return nil, orberrors.From(err)
	}

	// Create a Response{} and fill it.
	res := &client.RawResponse{
		ContentType: resp.Header.Get("Content-Type"),
		Body:        buff,
		Headers:     make(map[string][]string),
	}

	t.logger.Trace(
		"Got a result", "url", fmt.Sprintf("%s://%s/%s", node.Transport, node.Address, req.Endpoint()), "content-type", res.ContentType,
	)

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
func NewTransport(name string, logger log.Logger, scheme string, clientCreator TransportClientCreator,
) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		name:          name,
		logger:        logger,
		scheme:        scheme,
		clientCreator: clientCreator,
	}}, nil
}
