// Package orb provides the default client for go-orb.
package orb

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"sync"
	"time"

	"log/slog"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/go-orb/util/orberrors"
)

var _ (client.Client) = (*Client)(nil)

// Client is the microservices client for go-orb.
type Client struct {
	config Config

	logger   log.Logger
	registry registry.Registry

	middlewares []client.Middleware

	transportLock sync.Mutex
	transports    *container.Map[string, Transport]
}

// Logger returns the logger.
func (c *Client) Logger() log.Logger {
	return c.logger
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

// SelectService selects a service node.
func (c *Client) SelectService(ctx context.Context, service string, opts ...client.CallOption) (string, string, error) {
	options := c.makeOptions(opts...)

	return c.selectNode(ctx, service, options)
}

// selectNode returns a node for the given service.
func (c *Client) selectNode(ctx context.Context, service string, opts *client.CallOptions) (string, string, error) {
	if opts.URL != "" {
		myURL, err := url.Parse(opts.URL)
		if err != nil {
			return "", "", orberrors.ErrBadRequest.Wrap(err)
		}

		return myURL.Host, myURL.Scheme, nil
	}

	// Resolve the service to a list of nodes in a per transport map.
	nodes, err := c.resolveService(ctx, service, opts.PreferredTransports...)
	if err != nil {
		c.Logger().Error("Failed to resolve service", "error", err, "service", service)
		return "", "", err
	}

	// Run the configured Selector to get a node from the resolved nodes.
	node, err := opts.Selector(ctx, service, nodes, opts.PreferredTransports, opts.AnyTransport)
	if err != nil {
		c.Logger().Error("Failed to resolve service", "error", err, "service", service)
		return "", "", err
	}

	return node.Address, node.Transport, nil
}

// resolveService resolves a servicename to a Node with the help of the registry.
func (c *Client) resolveService(
	_ context.Context,
	service string,
	preferredTransports ...string,
) (client.NodeMap, error) {
	if service == "" {
		return nil, client.ErrServiceArgumentEmpty
	}

	// Try to resolve the service with retries
	var (
		services []*registry.Service
		err      error
	)

	// Retry up to 3 times with a small delay between attempts
	for retries := 0; retries < 1000; retries++ {
		if _, err := client.ResolveMemoryServer(service); err == nil {
			rNodes := make(client.NodeMap)
			rNodes["memory"] = []*registry.Node{
				{
					ID:        "memory",
					Address:   "",
					Transport: "memory",
				},
			}

			return rNodes, nil
		}

		services, err = c.registry.GetService(service)
		if err == nil && len(services) > 0 {
			c.logger.Debug("service resolution successful", "service", service)
			break // Service found, exit retry loop
		}

		c.logger.Debug("service resolution failed, retrying", "service", service, "attempt", retries+1, "error", err)
		time.Sleep(time.Duration(math.Pow(float64(retries+1), math.E)) * time.Millisecond * 100) // Increasing backoff
	}

	if err != nil {
		c.logger.Debug("service resolution failed after retries", "service", service, "error", err)
		return nil, err
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no instances found for service: %s", service)
	}

	rNodes := make(client.NodeMap)

	// Find nodes to query
	for _, service := range services {
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

func (c *Client) makeOptions(opts ...client.CallOption) *client.CallOptions {
	// Construct CallOptions, use the client's config as base.
	co := &client.CallOptions{
		ContentType:         c.config.Config.ContentType,
		PreferredTransports: c.config.Config.PreferredTransports,
		AnyTransport:        c.config.Config.AnyTransport,
		Selector:            c.config.Config.Selector,
		DialTimeout:         time.Duration(c.config.Config.DialTimeout),
		ConnectionTimeout:   time.Duration(c.config.Config.ConnectionTimeout),
		RequestTimeout:      time.Duration(c.config.Config.RequestTimeout),
		StreamTimeout:       time.Duration(c.config.Config.StreamTimeout),
		ConnClose:           false,
		TLSConfig:           c.config.Config.TLSConfig,

		Metadata:         map[string]string{},
		ResponseMetadata: map[string]string{},

		RetryFunc:          client.DefaultCallOptionsRetryFunc,
		Retries:            client.DefaultCallOptionsRetries,
		MaxCallRecvMsgSize: client.DefaultMaxCallRecvMsgSize,
		MaxCallSendMsgSize: client.DefaultMaxCallSendMsgSize,
	}

	// Apply options.
	for _, o := range opts {
		o(co)
	}

	return co
}

func (c *Client) transport(transport string) (Transport, error) {
	c.transportLock.Lock()
	defer c.transportLock.Unlock()

	// Try to fetch the transport from the internal registry.
	transportInstance, ok := c.transports.Get(transport)
	if ok {
		return transportInstance, nil
	}

	// Failed to get it from the registry, try to create a new one.
	tcreator, ok := Transports.Get(transport)
	if !ok {
		c.logger.Error("Failed to create a transport", slog.String("transport", transport))

		return nil, orberrors.ErrInternalServerError.Wrap(
			fmt.Errorf("%w: %s", client.ErrFailedToCreateTransport, transport),
		)
	}

	transportInstance, err := tcreator(c.logger.With("transport", transport), &c.config)
	if err != nil {
		c.logger.Error(
			"Failed to create a transport",
			slog.String("transport", transport),
			slog.Any("error", err),
		)

		return nil, orberrors.ErrInternalServerError.Wrap(
			fmt.Errorf("%w: %s", client.ErrFailedToCreateTransport, transport),
		)
	}

	if err := transportInstance.Start(); err != nil {
		return nil, orberrors.From(err)
	}

	// Store the transport for later use.
	c.transports.Set(transport, transportInstance)

	return transportInstance, nil
}

// Request does the actual call.
func (c *Client) Request(
	ctx context.Context,
	service string,
	endpoint string,
	req any,
	result any,
	opts ...client.CallOption,
) error {
	options := c.makeOptions(opts...)

	address, transport, err := c.selectNode(ctx, service, options)
	if err != nil {
		return err
	}

	t, err := c.transport(transport)
	if err != nil {
		return err
	}

	infos := client.RequestInfos{
		Service:   service,
		Endpoint:  endpoint,
		Transport: transport,
		Address:   address,
	}

	// Add request infos to context
	ctx = context.WithValue(ctx, client.RequestInfosKey{}, &infos)

	err = t.Request(ctx, infos, req, result, options)
	if err != nil {
		return err
	}

	return nil
}

// Stream opens a bidirectional stream to the service endpoint.
func (c *Client) Stream(
	ctx context.Context,
	service string,
	endpoint string,
	opts ...client.CallOption,
) (client.StreamIface[any, any], error) {
	options := c.makeOptions(opts...)

	address, transport, err := c.selectNode(ctx, service, options)
	if err != nil {
		return nil, err
	}

	t, err := c.transport(transport)
	if err != nil {
		return nil, err
	}

	infos := client.RequestInfos{
		Service:   service,
		Endpoint:  endpoint,
		Transport: transport,
		Address:   address,
	}

	// Add request infos to context
	ctx = context.WithValue(ctx, client.RequestInfosKey{}, &infos)

	stream, err := t.Stream(ctx, infos, options)
	if err != nil {
		// Don't cancel here - the context is owned by the caller
		c.logger.Error("stream failed", "error", err, "address", address, "transport", transport)

		return nil, err
	}

	return stream, nil
}

// New creates a new orb client. This functions should rarely be called manually.
// To create a new client use ProvideClientOrb.
func New(cfg Config, log log.Logger, registry registry.Type) *Client {
	// Filter out unknown preferred transports from config.
	nPTransports := []string{}

	for _, pt := range cfg.Config.PreferredTransports {
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

	cfg.Config.PreferredTransports = nPTransports

	return &Client{
		config:     cfg,
		logger:     log,
		registry:   registry,
		transports: container.NewMap[string, Transport](),
	}
}

// Provide is the wire provider for client.
//
//nolint:gocognit,gocyclo
func Provide(
	configData map[string]any,
	components *types.Components,
	logger log.Logger,
	registry registry.Type,
	opts ...client.Option,
) (client.Type, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, client.DefaultConfigSection, configData, &cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return client.Type{}, err
	}

	newClient := New(cfg, logger, registry)

	//nolint:nestif
	if config.HasKey[[]any]([]string{client.DefaultConfigSection}, "middlewares", configData) || len(cfg.Config.Middleware) > 0 {
		// Get and factory them all.
		middlewares := []client.Middleware{}

		for i := 0; ; i++ {
			mCfg := &client.MiddlewareConfig{}

			sections := []string{client.DefaultConfigSection, "middlewares"}

			err := config.Parse(sections, strconv.Itoa(i), configData, mCfg)
			if err != nil && !errors.Is(err, config.ErrNoSuchKey) || mCfg.Name == "" {
				logger.Warn("Unable to parse middleware config", "section", sections, "key", strconv.Itoa(i))
				break
			}

			fac, ok := client.Middlewares.Get(mCfg.Name)
			if !ok {
				return client.Type{}, fmt.Errorf("Client middleware '%s' not found, did you import it?", mCfg.Name)
			}

			mConfig, err := config.WalkMap([]string{client.DefaultConfigSection, "middlewares", strconv.Itoa(i)}, configData)
			if err != nil && !errors.Is(err, config.ErrNoSuchKey) {
				return client.Type{}, err
			}

			m, err := fac(mConfig, client.Type{Client: newClient}, logger)
			if err != nil {
				return client.Type{}, err
			}

			middlewares = append(middlewares, m)
		}

		for _, m := range cfg.Config.Middleware {
			fac, ok := client.Middlewares.Get(m.Name)
			if !ok {
				return client.Type{}, fmt.Errorf("Client middleware '%s' not found, did you import it?", m.Name)
			}

			mConfig, err := config.ParseStruct([]string{}, m)
			if err != nil && !errors.Is(err, config.ErrNoSuchKey) {
				return client.Type{}, err
			}

			m, err := fac(mConfig, client.Type{Client: newClient}, logger)
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
	err := components.Add(&instance, types.PriorityClient)
	if err != nil {
		logger.Warn("while registering client/orb as a component", "error", err)
	}

	return instance, nil
}
