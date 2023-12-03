// Package basehertz contains a base transport which is used by hertz transports.
package basehertz

import (
	"bytes"
	"context"
	"fmt"

	hclient "github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

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
type TransportClientCreator func(ctx context.Context, opts *client.CallOptions) (*hclient.Client, error)

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	name          string
	logger        log.Logger
	clientCreator TransportClientCreator
	hclient       *hclient.Client
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

	t.logger.Trace(
		"Making a request",
		"url", fmt.Sprintf("%s://%s/%s", t.scheme, node.Address, req.Endpoint()),
		"content-type", opts.ContentType,
	)

	// Create a hertz request.
	hReq := &protocol.Request{}
	hReq.SetMethod(consts.MethodPost)
	hReq.SetBodyStream(buff, buff.Len())
	hReq.Header.SetContentTypeBytes([]byte(opts.ContentType))
	hReq.Header.Set("Accept", opts.ContentType)
	hReq.SetRequestURI(fmt.Sprintf("%s://%s/%s", t.scheme, node.Address, req.Endpoint()))

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

	return t.call2(ctx, node, opts, req, hReq)
}

type hresBodyCloserWrapper struct {
	buff *bytes.Buffer
}

func (h *hresBodyCloserWrapper) Read(p []byte) (n int, err error) {
	return h.buff.Read(p)
}

func (h *hresBodyCloserWrapper) Close() error {
	return nil
}

func (t *Transport) call2(
	ctx context.Context,
	node *registry.Node,
	opts *client.CallOptions,
	req *client.Request[any, any],
	hReq *protocol.Request,
) (*client.RawResponse, error) {
	// Run the request.
	hRes := &protocol.Response{}

	err := t.hclient.DoTimeout(ctx, hReq, hRes, opts.RequestTimeout)
	if err != nil {
		return nil, orberrors.From(err)
	}

	// Read into a bytes.Buffer.
	buff := bytes.NewBuffer(hRes.Body())

	// Create a Response{} and fill it.
	res := &client.RawResponse{
		ContentType: hRes.Header.Get("Content-Type"),
		Body:        &hresBodyCloserWrapper{buff: buff},
		Headers:     make(map[string][]string),
	}

	t.logger.Trace(
		"Got a result",
		"url", fmt.Sprintf("%s://%s/%s", t.scheme, node.Address, req.Endpoint()),
		"content-type", res.ContentType,
		"size", hRes.Header.Get(consts.HeaderContentLength),
	)

	// Copy headers to the RawResponse if wanted.
	// TODO(jochumdev): Implement me.
	// if opts.Headers {
	// 	for k, v := range hRes.Header.Al {
	// 		res.Headers[k] = v
	// 	}
	// }

	if hRes.StatusCode() != consts.StatusOK {
		return res, orberrors.NewHTTP(hRes.StatusCode())
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
