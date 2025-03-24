// Package memory provides the memory RPC server for go-orb.
package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/util/metadata"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Name is the plugin name.
const Name = "memory"

// Server is the memory Server for go-orb.
type Server struct {
	serviceName    string
	serviceVersion string

	config   *Config
	logger   log.Logger
	registry registry.Type

	ctx        context.Context
	cancelFunc context.CancelFunc

	mux *Mux

	handlers    []orbserver.RegistrationFunc
	middlewares []orbserver.Middleware

	endpoints []string

	started bool
}

// Start registers the memory server with the client package.
func (s *Server) Start(ctx context.Context) error {
	if s.started {
		return nil
	}

	s.logger.Info("Starting memory server")

	// create a memory RPC mux
	s.mux = newMux(s)

	// Register handlers.
	for _, h := range s.handlers {
		h(s)
	}

	s.ctx, s.cancelFunc = context.WithCancel(ctx)

	// Register the memory server with the client package
	client.RegisterMemoryServer(s.serviceName, s)

	s.started = true

	return nil
}

// Stop will unregister the memory server from the client package.
func (s *Server) Stop(_ context.Context) error {
	if !s.started {
		return nil
	}

	s.logger.Info("Stopping memory server")

	// Cancel any ongoing operations
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}

	// Unregister from the client package
	client.UnregisterMemoryServer(s.serviceName)

	// Clean up resources
	s.started = false

	// Deregister from registry
	return nil
}

// AddHandler adds a handler for later registration.
func (s *Server) AddHandler(handler orbserver.RegistrationFunc) {
	s.handlers = append(s.handlers, handler)
}

// Register executes a registration function on the entrypoint.
func (s *Server) Register(register orbserver.RegistrationFunc) {
	if register == nil {
		s.logger.Warn("Nil register function")
		return
	}

	register(s)
}

// AddEndpoint adds an endpoint to the internal list.
// This is used by the Register() callback function.
func (s *Server) AddEndpoint(name string) {
	s.endpoints = append(s.endpoints, name)
}

// Address returns an empty string as memory server doesn't have a network address.
func (s *Server) Address() string {
	return ""
}

// Transport returns the client transport to use: "memory".
func (s *Server) Transport() string {
	return "memory"
}

// String returns the entrypoint type.
func (s *Server) String() string {
	return s.Type()
}

// Enabled returns if this entrypoint has been enabled in config.
func (s *Server) Enabled() bool {
	return true
}

// Name returns the entrypoint name.
func (s *Server) Name() string {
	return Name
}

// Type returns the component type.
func (s *Server) Type() string {
	return "server"
}

// Router returns the memory mux.
func (s *Server) Router() *Mux {
	return s.mux
}

// Request implements the client.MemoryServer interface.
func (s *Server) Request(ctx context.Context, infos client.RequestInfos, req any, result any, opts *client.CallOptions) error {
	// Extract the service and method from the request
	service := infos.Service
	endpoint := infos.Endpoint

	// Get a reference to the actual request data, making sure it's not nil
	requestData := req
	if requestData == nil {
		// If Req() returns nil, use the entire req object as the request
		// This ensures we always have something to work with
		requestData = req
	}

	// Add metadata to context
	ctx, reqMd := metadata.WithIncoming(ctx)
	ctx, _ = metadata.WithOutgoing(ctx)

	reqMd[metadata.Service] = service
	reqMd[metadata.Method] = endpoint

	// Add request infos to context
	ctx = context.WithValue(ctx, client.RequestInfosKey{}, &infos)

	// Create a new memory stream
	cStream, sStream := CreateClientServerPair(ctx, endpoint)
	cStream.responseMd = opts.ResponseMetadata

	if err := cStream.Send(requestData); err != nil {
		return err
	}

	// Create the server stream handler
	go func() {
		if err := s.mux.HandleRPC(sStream, endpoint); err != nil {
			select {
			case cStream.errCh <- err:
			default:
			}
		}

		_ = sStream.Close() //nolint:errcheck
	}()

	if err := cStream.CloseSend(); err != nil {
		return err
	}

	if err := cStream.Recv(result); err != nil {
		return err
	}

	return nil
}

// Provide creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports.
func Provide(
	serviceName string,
	serviceVersion string,
	configs map[string]any,
	logger log.Logger,
	reg registry.Type,
	opts ...orbserver.Option,
) (orbserver.Entrypoint, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, "", configs, cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return nil, err
	}

	// Configure Middlewares.
	for idx, cfgMw := range cfg.Middlewares {
		pFunc, ok := orbserver.Middlewares.Get(cfgMw.Plugin)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", orbserver.ErrUnknownMiddleware, cfgMw.Plugin)
		}

		mw, err := pFunc([]string{"middlewares"}, strconv.Itoa(idx), configs, logger)
		if err != nil {
			return nil, err
		}

		cfg.OptMiddlewares = append(cfg.OptMiddlewares, mw)
	}

	// Get handlers.
	for _, k := range cfg.Handlers {
		h, ok := orbserver.Handlers.Get(k)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", orbserver.ErrUnknownHandler, k)
		}

		cfg.OptHandlers = append(cfg.OptHandlers, h)
	}

	return New(serviceName, serviceVersion, cfg, logger, reg)
}

// New creates a memory Server from a Config struct.
func New(
	serviceName string,
	serviceVersion string,
	acfg any,
	logger log.Logger,
	reg registry.Type,
) (orbserver.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("memory invalid config: %v", cfg)
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	ctx, cancelFunc := context.WithCancel(context.Background())

	entrypoint := Server{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		config:         cfg,
		logger:         logger,
		registry:       reg,
		handlers:       cfg.OptHandlers,
		middlewares:    cfg.OptMiddlewares,
		ctx:            ctx,
		cancelFunc:     cancelFunc,
	}

	return &entrypoint, nil
}
