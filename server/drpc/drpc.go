// Package drpc provides the drpc server for go-orb.
package drpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"storj.io/drpc/drpcserver"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/util/addr"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Plugin is the plugin name.
const Plugin = "drpc"

// Server is the drpc Server for go-orb.
type Server struct {
	serviceName    string
	serviceVersion string
	epName         string

	config   *Config
	logger   log.Logger
	registry registry.Type

	address string

	ctx        context.Context
	cancelFunc context.CancelFunc

	mux    *Mux
	server *drpcserver.Server

	handlers    []orbserver.RegistrationFunc
	middlewares []orbserver.Middleware

	endpoints []string

	started bool
}

// Start will create the listeners and start the server on the entrypoint.
func (s *Server) Start(ctx context.Context) error {
	if s.started {
		return nil
	}

	s.logger.Info("Starting", "address", s.config.Address)

	// create a drpc RPC mux
	s.mux = newMux(s)

	// Register handlers.
	for _, h := range s.handlers {
		h(s)
	}

	s.server = drpcserver.New(s.mux)

	var err error

	listener := s.config.Listener
	if listener == nil {
		if s.config.Network == "unix" {
			if err := os.MkdirAll(filepath.Dir(s.config.Address), 0o700); err != nil {
				return fmt.Errorf("while creating the directory for %s: %w", s.config.Address, err)
			}
		}

		listener, err = net.Listen(s.config.Network, s.config.Address)
		if err != nil {
			return err
		}
	}

	s.address = listener.Addr().String()

	s.logger = s.logger.With(slog.String("transport", s.Transport()), slog.String("address", s.address))

	s.logger.Info("dRPC server listening")

	s.ctx, s.cancelFunc = context.WithCancel(ctx)

	go func() {
		if err := s.server.Serve(s.ctx, listener); err != nil {
			s.logger.Error("while starting the dRPC Server", "error", err)
		}
	}()

	s.started = true

	return s.registryRegister(ctx)
}

// Stop will stop the dRPC server.
func (s *Server) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	// Stops the dRPC Server.
	s.cancelFunc()

	return s.registryDeregister(ctx)
}

// AddHandler adds a handler for later registration.
func (s *Server) AddHandler(handler orbserver.RegistrationFunc) {
	s.handlers = append(s.handlers, handler)
}

// Register executes a registration function on the entrypoint.
func (s *Server) Register(register orbserver.RegistrationFunc) {
	if !s.started {
		return
	}

	register(s.mux)
}

// AddEndpoint add's an endpoint to the internal list.
// This is used by the Register() callback function.
func (s *Server) AddEndpoint(name string) {
	s.endpoints = append(s.endpoints, name)
}

// Network returns the network the entrypoint is listening on.
func (s *Server) Network() string {
	return s.config.Network
}

// Address returns the address the entrypoint is listening on, for example: [::]:8381.
func (s *Server) Address() string {
	if !s.started {
		return ""
	}

	return s.address
}

// Transport returns the client transport to use: "drpc".
func (s *Server) Transport() string {
	if s.config.Network == "unix" {
		return "unix+drpc"
	}

	return "drpc"
}

// String returns the entrypoint type.
func (s *Server) String() string {
	return Plugin
}

// Enabled returns if this entrypoint has been enbaled in config.
func (s *Server) Enabled() bool {
	return s.config.Enabled
}

// Name returns the entrypoint name.
func (s *Server) Name() string {
	return s.epName
}

// Type returns the component type.
func (s *Server) Type() string {
	return orbserver.EntrypointType
}

// Router returns the drpc mux.
func (s *Server) Router() *Mux {
	return s.mux
}

func (s *Server) registryService() registry.ServiceNode {
	return registry.ServiceNode{
		Name:     s.serviceName,
		Version:  s.serviceVersion,
		Address:  s.Address(),
		Node:     s.Name(),
		Network:  s.Network(),
		Scheme:   s.Transport(),
		Metadata: make(map[string]string),
	}
}

func (s *Server) registryRegister(ctx context.Context) error {
	return s.registry.Register(ctx, s.registryService())
}

func (s *Server) registryDeregister(ctx context.Context) error {
	return s.registry.Deregister(ctx, s.registryService())
}

// Provide creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports.
func Provide(
	name string,
	version string,
	epName string,
	configData map[string]any,
	logger log.Logger,
	reg registry.Type,
	opts ...orbserver.Option,
) (orbserver.Entrypoint, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(nil, "", configData, cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return nil, err
	}

	return New(name, version, epName, cfg, logger, reg)
}

// New creates a dRPC Server from a Config struct.
func New(name string, version string, epName string, acfg any, logger log.Logger, reg registry.Type) (orbserver.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("drpc invalid config: %v", cfg)
	}

	var err error

	if cfg.Network == "tcp" {
		cfg.Address, err = addr.GetAddress(cfg.Address)
		if err != nil {
			return nil, fmt.Errorf("drpc validate addr '%s': %w", cfg.Address, err)
		}

		if err := addr.ValidateAddress(cfg.Address); err != nil {
			return nil, err
		}
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	entrypoint := Server{
		serviceName:    name,
		serviceVersion: version,
		epName:         epName,
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
