// Package orb provides the default client for go-orb.
package orb

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

var _ (client.Client) = (*Client)(nil)

// Client is the microservices client for go-orb.
type Client struct {
	config Config

	logger   log.Logger
	registry registry.Registry

	middlewares []client.Middleware

	transports container.SafeMap[Transport]
}

// Start starts the client.
func (c *Client) Start() error {
	return nil
}

// Stop stops the client.
func (c *Client) Stop(ctx context.Context) error {
	hasError := false

	for _, t := range c.transports.All() {
		if err := t.Stop(ctx); err != nil {
			c.logger.Error("failed to stop a transport", "error", err)

			hasError = true
		}
	}

	if hasError {
		return errors.New("there has been an error stopping the orb client, see the logs")
	}

	return nil
}

func (c *Client) String() string {
	return Name
}

// Type returns the clients componenttype "client".
func (c *Client) Type() string {
	return client.ComponentType
}

// Config returns a pointer to the clients config.
func (c *Client) Config() *client.Config {
	return &c.config.Config
}

// ResolveService resolves a servicename to a Node with the help of the registry.
func (c *Client) ResolveService(
	_ context.Context,
	service string,
	preferredTransports ...string,
) (*container.Map[[]*registry.Node], error) {
	if service == "" {
		return nil, client.ErrServiceArgumentEmpty
	}

	svc, err := c.registry.GetService(service)
	if err != nil {
		return nil, err
	}

	rNodes := container.NewMap[[]*registry.Node]()

	// Find nodes to query
	for _, service := range svc {
		for _, node := range service.Nodes {
			tNodes, err := rNodes.Get(node.Transport)
			if err != nil {
				tNodes = []*registry.Node{}
			}

			tNodes = append(tNodes, node)
			rNodes.Set(node.Transport, tNodes)
		}
	}

	// Not one node found.
	if rNodes.Len() == 0 {
		return nil, fmt.Errorf("%w: requested transports was: %s", client.ErrNoNodeFound, preferredTransports)
	}

	return rNodes, nil
}

func (c *Client) NeedsCodec(ctx context.Context, req *client.Request[any, any], opts ...client.CallOption) bool {
	co := c.makeOptions(opts...)

	transport, err := c.transportForReq(ctx, req, co)
	if err != nil {
		return false
	}

	return transport.NeedsCodec()
}

func (c *Client) makeOptions(opts ...client.CallOption) *client.CallOptions {
	// Construct CallOptions, use the client's config as base.
	co := &client.CallOptions{
		ContentType:         c.config.Config.ContentType,
		PreferredTransports: c.config.Config.PreferredTransports,
		AnyTransport:        c.config.Config.AnyTransport,
		Selector:            c.config.Config.Selector,
		Backoff:             c.config.Config.Backoff,
		Retry:               c.config.Config.Retry,
		Retries:             c.config.Config.Retries,
		DialTimeout:         c.config.Config.DialTimeout,
		ConnectionTimeout:   c.config.Config.ConnectionTimeout,
		RequestTimeout:      c.config.Config.RequestTimeout,
		StreamTimeout:       c.config.Config.StreamTimeout,
		ConnClose:           false,
		TlsConfig:           c.config.Config.TlsConfig,
	}

	// Apply options.
	for _, o := range opts {
		o(co)
	}

	return co
}

func (c *Client) transportForReq(ctx context.Context, req *client.Request[any, any], opts *client.CallOptions) (Transport, error) {
	node, err := req.Node(ctx, opts)
	if err != nil {
		return nil, orberrors.ErrInternalServerError.Wrap(err)
	}

	// Try to fetch the transport from the internal registry.
	transport, err := c.transports.Get(node.Transport)

	if err != nil {
		err = nil

		// Failed to get it from the registry, try to create a new one.
		tcreator, err := Transports.Get(node.Transport)
		if err != nil {
			c.logger.Error("Failed to create a transport", slog.String("service", req.Service()), slog.String("transport", node.Transport))
			return nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("%w (%s)", client.ErrFailedToCreateTransport, node.Transport))
		}

		transport, err = tcreator(c.logger.With("transport", node.Transport))
		if err != nil {
			c.logger.Error(
				"Failed to create a transport",
				slog.String("service", req.Service()),
				slog.String("transport", node.Transport),
				slog.Any("error", err),
			)

			return nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("%w (%s)", client.ErrFailedToCreateTransport, node.Transport))
		}

		if err := transport.Start(); err != nil {
			return nil, orberrors.From(err)
		}

		// Store the transport for later use.
		c.transports.Set(node.Transport, transport)
	}

	return transport, err
}

// Call does the actual call.
func (c *Client) Call(
	ctx context.Context,
	req *client.Request[any, any],
	result any,
	opts ...client.CallOption,
) (resp *client.RawResponse, err error) {

	co := c.makeOptions(opts...)

	// Wrap middlewares
	call := c.call
	for _, m := range c.middlewares {
		call = m.Call(call)
	}

	return call(ctx, req, co)
}

func (c *Client) call(ctx context.Context, req *client.Request[any, any], opts *client.CallOptions) (resp *client.RawResponse, err error) {
	transport, err := c.transportForReq(ctx, req, opts)
	if err != nil {
		return nil, err
	}

	// Add metadata to the context.
	ctx = metadata.Ensure(ctx)

	return transport.Call(ctx, req, opts)
}

// Call does the actual call.
func (c *Client) CallNoCodec(
	ctx context.Context,
	req *client.Request[any, any],
	result any,
	opts ...client.CallOption,
) error {

	co := c.makeOptions(opts...)

	// Wrap middlewares
	call := c.callNoCodec
	for _, m := range c.middlewares {
		call = m.CallNoCodec(call)
	}

	// Add metadata to the context.
	ctx = metadata.Ensure(ctx)

	return call(ctx, req, result, co)
}

func (c *Client) callNoCodec(ctx context.Context, req *client.Request[any, any], result any, opts *client.CallOptions) (err error) {
	transport, err := c.transportForReq(ctx, req, opts)
	if err != nil {
		return err
	}

	return transport.CallNoCodec(ctx, req, result, opts)
}

// New creates a new orb client. This functions should rarely be called manually.
// To create a new client use ProvideClientOrb.
func New(cfg Config, log log.Logger, registry registry.Type) *Client {
	// Filter out unknown preferred transports from config.
	nPTransports := []string{}
	allTransports := Transports.Keys()

	for _, pt := range cfg.PreferredTransports {
		if slices.Contains(allTransports, pt) {
			nPTransports = append(nPTransports, pt)
		}
	}

	cfg.PreferredTransports = nPTransports

	// To keep the client working when no transports match,
	// we use all transports in any order as preferred ones.
	if len(cfg.PreferredTransports) == 0 {
		cfg.PreferredTransports = allTransports
	}

	return &Client{
		config:      cfg,
		logger:      log,
		registry:    registry,
		middlewares: []client.Middleware{},
		transports:  *container.NewSafeMap[Transport](),
	}
}

// ProvideClientOrb is the wire provider for client.
func ProvideClientOrb(
	name types.ServiceName,
	data types.ConfigData,
	logger log.Logger,
	registry registry.Type,
	opts ...client.Option,
) (client.Type, error) {
	cfg, err := NewConfig(name, data, opts...)
	if err != nil {
		return client.Type{}, err
	}

	sections := types.SplitServiceName(name)
	if err := config.Parse(append(sections, client.DefaultConfigSection), data, cfg); err != nil {
		return client.Type{}, err
	}

	c := New(cfg, logger, registry)

	return client.Type{Client: c}, nil
}
