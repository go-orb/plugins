package memory

import (
	"context"
	"io"
	"maps"
	"reflect"
	"sync"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"storj.io/drpc"
)

// Stream creates a new bidirectional stream for memory-based RPC communication.
func (s *Server) Stream(
	ctx context.Context,
	infos client.RequestInfos,
	opts *client.CallOptions,
) (client.StreamIface[any, any], error) {
	if s.mux == nil {
		return nil, orberrors.ErrBadRequest.WrapNew("server not configured with a mux")
	}

	// Extract the service and method from the request
	service := infos.Service
	endpoint := infos.Endpoint

	// Add metadata to context
	ctx, reqMd := metadata.WithIncoming(ctx)
	ctx, outMd := metadata.WithOutgoing(ctx)

	reqMd[metadata.Service] = service
	reqMd[metadata.Method] = endpoint

	// Add request infos to context
	ctx = context.WithValue(ctx, client.RequestInfosKey{}, &infos)

	// Create a new memory stream
	stream := &MStream{
		ctx:      ctx,
		endpoint: endpoint,
		sendCh:   make(chan any, 10),
		recvCh:   make(chan any, 10),
		errCh:    make(chan error, 1),
		doneCh:   make(chan struct{}),
		mu:       sync.Mutex{},
	}

	// Create the server stream handler
	go func() {
		if err := s.mux.HandleRPC(stream, endpoint); err != nil {
			select {
			case stream.errCh <- err:
			default:
			}
		}

		_ = stream.Close() //nolint:errcheck
	}()

	// Copy response metadata if needed
	if opts.ResponseMetadata != nil {
		maps.Copy(opts.ResponseMetadata, outMd)
	}

	return stream, nil
}

// MStream implements the client.StreamIface and the drpc.Stream interface for memory-based streaming.
type MStream struct {
	ctx      context.Context
	endpoint string
	closed   bool
	mu       sync.Mutex

	sendCh chan any
	recvCh chan any
	errCh  chan error
	doneCh chan struct{}
}

// Context returns the stream's context.
func (s *MStream) Context() context.Context {
	return s.ctx
}

// Send sends a message through the stream.
func (s *MStream) Send(msg any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return io.EOF
	}

	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.doneCh:
		return io.EOF
	case s.sendCh <- msg:
		return nil
	default:
		// This will happen if sendCh is closed
		return io.ErrClosedPipe
	}
}

// Recv receives a message from the stream.
func (s *MStream) Recv(msg any) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.doneCh:
		return io.EOF
	case err := <-s.errCh:
		return err
	case received, ok := <-s.recvCh:
		if !ok {
			return io.EOF
		}

		// Try to copy the received message to the provided message pointer
		rv := reflect.ValueOf(msg)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return orberrors.ErrBadRequest.WrapNew("message must be a non-nil pointer")
		}

		// Clone the received message into the destination
		sourceVal := reflect.ValueOf(received)
		destVal := rv.Elem()

		if sourceVal.Type().AssignableTo(destVal.Type()) {
			destVal.Set(sourceVal)
			return nil
		}

		return orberrors.ErrBadRequest.WrapNew("message types are not compatible")
	}
}

// MsgSend sends a message through the stream.
func (s *MStream) MsgSend(msg drpc.Message, _ drpc.Encoding) error {
	return s.Send(msg)
}

// MsgRecv receives a message from the stream.
func (s *MStream) MsgRecv(msg drpc.Message, _ drpc.Encoding) error {
	return s.Recv(msg)
}

// Close closes the stream.
func (s *MStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.doneCh)

	return nil
}

// CloseSend closes the send side of the stream but keeps the receive side open.
func (s *MStream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	// Close only the send channel
	close(s.sendCh)

	return nil
}
