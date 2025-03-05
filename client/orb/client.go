// Package orb provides the default client for go-orb.
package orb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"log/slog"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
)

var _ (client.Client) = (*Client)(nil)

// Client is the microservices client for go-orb.
type Client struct {
	config Config

	logger   log.Logger
	registry registry.Registry

	middlewares []client.Middleware

	transports *container.SafeMap[string, Transport]
}

// Start starts the client.
func (c *Client) Start(_ context.Context) error {
	return nil
}

// Stop stops the client.
func (c *Client) Stop(ctx context.Context) error {
	hasError := false

	c.transports.Range(func(_ string, t Transport) bool {
		if err := t.Stop(ctx); err != nil {
			c.logger.Error("failed to stop a transport", "error", err)

			hasError = true
		}

		return true
	})

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

// Config returns the internal config, this is for tests.
func (c *Client) Config() client.Config {
	return c.config.Config
}

// With closes all transports and configures the client with the given options.
func (c *Client) With(opts ...client.Option) error {
	err := c.Stop(context.Background())

	for _, o := range opts {
		o(&c.config)
	}

	return err
}

// ResolveService resolves a servicename to a Node with the help of the registry.
func (c *Client) ResolveService(
	_ context.Context,
	service string,
	preferredTransports ...string,
) (client.NodeMap, error) {
	if service == "" {
		return nil, client.ErrServiceArgumentEmpty
	}

	svc, err := c.registry.GetService(service)
	if err != nil {
		return nil, err
	}

	rNodes := make(client.NodeMap)

	// Find nodes to query
	for _, service := range svc {
		for _, node := range service.Nodes {
			tNodes, ok := rNodes[node.Transport]
			if !ok {
				tNodes = []*registry.Node{}
			}

			tNodes = append(tNodes, node)
			rNodes[node.Transport] = tNodes
		}
	}

	// Not one node found.
	if len(rNodes) == 0 {
		return nil, fmt.Errorf("%w: requested transports was: %s", client.ErrNoNodeFound, preferredTransports)
	}

	return rNodes, nil
}

// NeedsCodec returns whetever the underlying transport requires a codec to translate the communication with the server.
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
		PoolSize:            c.config.Config.PoolSize,
		PoolTTL:             c.config.Config.PoolTTL,
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
		TLSConfig:           c.config.Config.TLSConfig,
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
	transport, ok := c.transports.Get(node.Transport)

	if !ok {
		// Failed to get it from the registry, try to create a new one.
		tcreator, ok := Transports.Get(node.Transport)
		if !ok {
			c.logger.Error("Failed to create a transport", slog.String("service", req.Service()), slog.String("transport", node.Transport))
			return nil, fmt.Errorf("%w: %w (%s)", orberrors.ErrInternalServerError, client.ErrFailedToCreateTransport, node.Transport)
		}

		transport, err = tcreator(c.logger.With("transport", node.Transport), &c.config)
		if err != nil {
			c.logger.Error(
				"Failed to create a transport",
				slog.String("service", req.Service()),
				slog.String("transport", node.Transport),
				slog.Any("error", err),
			)

			return nil, fmt.Errorf("%w: %w (%s)", orberrors.ErrInternalServerError, client.ErrFailedToCreateTransport, node.Transport)
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
	_ any,
	opts ...client.CallOption,
) (resp *client.RawResponse, err error) {
	callOptions := c.makeOptions(opts...)

	// Add metadata to the context.
	ctx, _ = metadata.WithOutgoing(ctx)

	transport, err := c.transportForReq(ctx, req, callOptions)
	if err != nil {
		return nil, err
	}

	// Wrap middlewares
	call := func(ctx context.Context, req *client.Request[any, any], opts *client.CallOptions) (*client.RawResponse, error) {
		result, err := transport.Call(ctx, req, opts)

		// Retry logic.
		if err != nil && callOptions.Retry != nil && callOptions.Retries > 0 {
			var retryCount int
			for retryCount < callOptions.Retries {
				retryCount++

				shouldRetry, rErr := callOptions.Retry(ctx, err, callOptions)
				if !shouldRetry || rErr != nil {
					break
				}

				result, err = transport.Call(ctx, req, callOptions)
			}
		}

		return result, err
	}
	for _, m := range c.middlewares {
		call = m.Call(call)
	}

	// The actual call.
	return call(ctx, req, callOptions)
}

// CallNoCodec does the actual call without codecs.
func (c *Client) CallNoCodec(
	ctx context.Context,
	req *client.Request[any, any],
	result any,
	opts ...client.CallOption,
) error {
	callOptions := c.makeOptions(opts...)

	// Add metadata to the context.
	ctx, _ = metadata.WithOutgoing(ctx)

	transport, err := c.transportForReq(ctx, req, callOptions)
	if err != nil {
		return err
	}

	// Wrap middlewares
	call := func(ctx context.Context, req *client.Request[any, any], result any, opts *client.CallOptions) error {
		err := transport.CallNoCodec(ctx, req, result, opts)

		// Retry logic.
		if err != nil && callOptions.Retry != nil && callOptions.Retries > 0 {
			var retryCount int
			for retryCount < callOptions.Retries {
				retryCount++

				shouldRetry, rErr := callOptions.Retry(ctx, err, callOptions)
				if !shouldRetry || rErr != nil {
					break
				}

				err = transport.CallNoCodec(ctx, req, result, opts)
			}
		}

		return err
	}
	for _, m := range c.middlewares {
		call = m.CallNoCodec(call)
	}

	// The actual call.
	return call(ctx, req, result, callOptions)
}

// New creates a new orb client. This functions should rarely be called manually.
// To create a new client use ProvideClientOrb.
func New(cfg Config, log log.Logger, registry registry.Type) *Client {
	// Filter out unknown preferred transports from config.
	nPTransports := []string{}

	for _, pt := range cfg.PreferredTransports {
		if _, ok := Transports.Get(pt); ok {
			nPTransports = append(nPTransports, pt)
		}
	}

	// To keep the client working when no transports match,
	// we use all transports in any order as preferred ones.
	if len(nPTransports) == 0 {
		Transports.Range(func(name string, _ TransportFactory) bool {
			nPTransports = append(nPTransports, name)
			return true
		})
	}

	cfg.PreferredTransports = nPTransports

	return &Client{
		config:     cfg,
		logger:     log,
		registry:   registry,
		transports: container.NewSafeMap[string, Transport](),
	}
}

// Provide is the wire provider for client.
//
//nolint:gocognit,gocyclo
func Provide(
	name types.ServiceName,
	data types.ConfigData,
	components *types.Components,
	logger log.Logger,
	registry registry.Type,
	opts ...client.Option,
) (client.Type, error) {
	cfg, err := NewConfig(name, data, opts...)
	if err != nil {
		return client.Type{}, err
	}

	sections := types.SplitServiceName(name)
	sections = append(sections, client.DefaultConfigSection)

	if err := config.Parse(sections, data, &cfg); err != nil {
		return client.Type{}, err
	}

	newClient := New(cfg, logger, registry)

	//nolint:nestif
	if config.HasKey[[]any](sections, "middlewares", data) || len(cfg.Middleware) > 0 {
		// Get and factory them all.
		middlewares := []client.Middleware{}

		for i := 0; ; i++ {
			mCfg := &client.MiddlewareConfig{}
			if err := config.Parse(append(sections, "middlewares", strconv.Itoa(i)), data, mCfg); err != nil || mCfg.Name == "" {
				if errors.Is(err, config.ErrNotExistent) || mCfg.Name == "" {
					break
				}

				return client.Type{}, err
			}

			fac, ok := client.Middlewares.Get(mCfg.Name)
			if !ok {
				return client.Type{}, fmt.Errorf("Client middleware '%s' not found, did you import it?", mCfg.Name)
			}

			m, err := fac(append(sections, "middlewares", strconv.Itoa(i)), data, client.Type{Client: newClient}, logger)
			if err != nil {
				return client.Type{}, err
			}

			middlewares = append(middlewares, m)
		}

		for _, m := range cfg.Middleware {
			fac, ok := client.Middlewares.Get(m.Name)
			if !ok {
				return client.Type{}, fmt.Errorf("Client middleware '%s' not found, did you import it?", m.Name)
			}

			myData, err := config.ParseStruct([]string{}, m)
			if err != nil {
				return client.Type{}, err
			}

			m, err := fac([]string{}, types.ConfigData{myData}, client.Type{Client: newClient}, logger)
			if err != nil {
				return client.Type{}, err
			}

			middlewares = append(middlewares, m)
		}

		// Apply them to the client.
		if len(middlewares) > 0 {
			newClient.middlewares = middlewares
		}
	}

	instance := client.Type{Client: newClient}

	// Register the client as a component.
	err = components.Add(&instance, types.PriorityClient)
	if err != nil {
		logger.Warn("while registering client/orb as a component", "error", err)
	}

	return instance, nil
}
