// Package memory provides the memory RPC server for go-orb.
package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"strconv"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	orbserver "github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/google/uuid"
	"storj.io/drpc"
)

var _ orbserver.Entrypoint = (*Server)(nil)

// Name is the plugin name.
const Name = "memory"

// Server is the memory Server for go-orb.
type Server struct {
	config   *Config
	logger   log.Logger
	registry registry.Type

	ctx        context.Context
	cancelFunc context.CancelFunc

	mux *Mux

	handlers    []orbserver.RegistrationFunc
	middlewares []orbserver.Middleware

	endpoints []string

	// entrypointID is the entrypointID (uuid) of this entrypoint in the registry.
	entrypointID string

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
	client.RegisterMemoryServer(string(s.registry.ServiceName()), s)

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
	client.UnregisterMemoryServer(s.registry.ServiceName())

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

// EntrypointID returns the id (uuid) of this entrypoint in the registry.
func (s *Server) EntrypointID() string {
	if s.entrypointID == "" {
		s.entrypointID = uuid.New().String()
	}

	return s.entrypointID
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

// Stream is a simple adapter that implements a minimal subset of the drpc.Stream interface
// for in-memory communication. This allows us to reuse the existing RPC handling logic.
type Stream struct {
	ctx     context.Context
	request any
	result  any
}

// Context returns the context for this stream.
func (m *Stream) Context() context.Context {
	return m.ctx
}

// Close implements the drpc.Stream interface.
func (m *Stream) Close() error {
	return nil
}

// CloseSend closes the send direction of the stream.
func (m *Stream) CloseSend() error {
	return nil
}

// MsgSend implements a simplified version of the drpc.Stream interface for memory-only communication.
func (m *Stream) MsgSend(msg drpc.Message, _ drpc.Encoding) error {
	// For memory calls, we directly copy the message to the result without serialization
	if m.result == nil {
		// No result to set, but not an error - just log and continue
		return nil
	}

	// Get result value for reflection
	resValue := reflect.ValueOf(m.result)
	if resValue.Kind() != reflect.Ptr || resValue.IsNil() {
		// Result must be a non-nil pointer
		return nil
	}

	// Get the value that the result pointer points to
	resElemValue := resValue.Elem()

	// Get message value for reflection
	msgValue := reflect.ValueOf(msg)
	origMsgValue := msgValue

	// 1. If message is a pointer, try to use its element value
	if msgValue.Kind() == reflect.Ptr && !msgValue.IsNil() {
		msgValue = msgValue.Elem()

		// Try direct assignment if types match
		if resElemValue.Type() == msgValue.Type() {
			resElemValue.Set(msgValue)
			return nil
		}

		// Try assignable types
		if msgValue.Type().AssignableTo(resElemValue.Type()) {
			resElemValue.Set(msgValue)
			return nil
		}
	}

	// 2. Try to use the original message value directly
	if origMsgValue.Type().AssignableTo(resElemValue.Type()) {
		resElemValue.Set(origMsgValue)
		return nil
	}

	// Last resort: use JSON codec as intermediary for type conversion
	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	// Encode the message to bytes using the codec
	b, err := codec.Marshal(msg)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	// Decode the bytes into the result using the codec
	if err := codec.Unmarshal(b, m.result); err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return nil
}

// MsgRecv implements a simplified version of the drpc.Stream interface for memory-only communication.
func (m *Stream) MsgRecv(msg drpc.Message, _ drpc.Encoding) error {
	// For in-memory calls, we directly set the received message to the request
	// data without any serialization/deserialization
	if m.request == nil {
		// If request is nil, we can't do anything meaningful
		return errors.New("memory stream has nil request")
	}

	reqValue := reflect.ValueOf(m.request)

	// Handle case where request is a pointer
	origReqValue := reqValue

	if reqValue.Kind() == reflect.Ptr && !reqValue.IsNil() {
		// For pointers, we might need to use the dereferenced value
		reqValue = reqValue.Elem()
	}

	msgValue := reflect.ValueOf(msg)
	if msgValue.Kind() == reflect.Ptr && !msgValue.IsNil() {
		msgValue = msgValue.Elem()

		// Try direct assignment if types match exactly
		if msgValue.Type() == reqValue.Type() {
			msgValue.Set(reqValue)
			return nil
		}

		// Try assignable types
		if reqValue.Type().AssignableTo(msgValue.Type()) {
			msgValue.Set(reqValue)
			return nil
		}

		// Try to convert between pointer and value
		if origReqValue.Type().AssignableTo(reflect.PtrTo(msgValue.Type())) {
			// Handle pointer compatibility
			msgValue.Set(origReqValue.Elem())
			return nil
		}
	}

	// Last resort: use JSON codec as intermediary for type conversion
	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	// Encode the request to bytes using the codec
	b, err := codec.Marshal(m.request)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	// Decode the bytes into the message using the codec
	if err := codec.Unmarshal(b, msg); err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return nil
}

// Request implements the client.MemoryServer interface.
func (s *Server) Request(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
	// For memory server, we directly handle the request without serialization
	// Since we're in the same process, we can just call the appropriate handler

	// Extract the service and method from the request
	service := req.Service()
	endpoint := req.Endpoint()

	// Get a reference to the actual request data, making sure it's not nil
	requestData := req.Req()
	if requestData == nil {
		// If Req() returns nil, use the entire req object as the request
		// This ensures we always have something to work with
		requestData = req
	}

	// Add metadata to context
	ctx, reqMd := metadata.WithIncoming(ctx)
	ctx, outMd := metadata.WithOutgoing(ctx)

	reqMd[metadata.Service] = service
	reqMd[metadata.Method] = endpoint

	// Create a memory stream to handle the request/response
	stream := &Stream{
		ctx:     ctx,
		request: requestData,
		result:  result,
	}

	// Execute the RPC handler through the mux
	err := s.mux.HandleRPC(stream, endpoint)

	maps.Copy(opts.ResponseMetadata, outMd)

	if err != nil {
		return err
	}

	return nil
}

// Provide creates a new entrypoint for a single address. You can create
// multiple entrypoints for multiple addresses and ports.
func Provide(
	sections []string,
	configs types.ConfigData,
	logger log.Logger,
	reg registry.Type,
	opts ...orbserver.Option,
) (orbserver.Entrypoint, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(sections, configs, cfg); err != nil {
		return nil, err
	}

	// Configure Middlewares.
	for idx, cfgMw := range cfg.Middlewares {
		pFunc, ok := orbserver.Middlewares.Get(cfgMw.Plugin)
		if !ok {
			return nil, fmt.Errorf("%w: '%s', did you register it?", orbserver.ErrUnknownMiddleware, cfgMw.Plugin)
		}

		mw, err := pFunc(append(sections, "middlewares", strconv.Itoa(idx)), configs, logger)
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

	return New(cfg, logger, reg)
}

// New creates a memory Server from a Config struct.
func New(acfg any, logger log.Logger, reg registry.Type) (orbserver.Entrypoint, error) {
	cfg, ok := acfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("memory invalid config: %v", cfg)
	}

	logger = logger.With(slog.String("entrypoint", cfg.Name))

	ctx, cancelFunc := context.WithCancel(context.Background())

	entrypoint := Server{
		config:      cfg,
		logger:      logger,
		registry:    reg,
		handlers:    cfg.OptHandlers,
		middlewares: cfg.OptMiddlewares,
		ctx:         ctx,
		cancelFunc:  cancelFunc,
	}

	return &entrypoint, nil
}
