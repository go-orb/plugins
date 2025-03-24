package memory

import (
	"context"
	"io"
	"maps"
	"reflect"
	"sync"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"storj.io/drpc"
)

// convertReceived converts a received message to the expected result type.
func convertReceived(msg any, received any) error {
	reqValue := reflect.ValueOf(received)

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
		if origReqValue.Type().AssignableTo(reflect.PointerTo(msgValue.Type())) {
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
	b, err := codec.Marshal(received)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	// Decode the bytes into the message using the codec
	if err := codec.Unmarshal(b, msg); err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return nil
}

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
	ctx, _ = metadata.WithOutgoing(ctx)

	reqMd[metadata.Service] = service
	reqMd[metadata.Method] = endpoint

	// Add request infos to context
	ctx = context.WithValue(ctx, client.RequestInfosKey{}, &infos)

	// Create a new memory stream
	cStream, sStream := CreateClientServerPair(ctx, endpoint)
	cStream.responseMd = opts.ResponseMetadata

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

	return cStream, nil
}

// Stream implements the client.StreamIface and the drpc.Stream interface for memory-based streaming.
type Stream struct {
	ctx      context.Context
	endpoint string
	closed   bool
	mu       sync.Mutex

	// For client->server communication
	clientToServer chan any
	// For server->client communication
	serverToClient chan any

	errCh  chan error
	doneCh chan struct{}

	// Indicates if this is the client or server side of the stream
	isClient bool

	// Response metadata
	responseMd map[string]string
}

// Context returns the stream's context.
func (s *Stream) Context() context.Context {
	return s.ctx
}

// Send sends a message through the stream.
func (s *Stream) Send(msg any) error {
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
	default:
		// Choose the right channel based on whether this is client or server
		if s.isClient {
			// Client sends to clientToServer channel
			select {
			case s.clientToServer <- msg:
				return nil
			default:
				return io.ErrClosedPipe
			}
		} else {
			// Server sends to serverToClient channel
			select {
			case s.serverToClient <- msg:
				return nil
			default:
				return io.ErrClosedPipe
			}
		}
	}
}

// Recv receives a message from the stream.
func (s *Stream) Recv(msg any) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.doneCh:
		return io.EOF
	case err := <-s.errCh:
		return err
	default:
		// Choose the right channel based on whether this is client or server
		var (
			received any
			chanOk   bool
		)

		if s.isClient {
			// Client receives from serverToClient channel
			select {
			case received, chanOk = <-s.serverToClient:
				// Copy response metadata if needed
				if outMD, mdok := metadata.Outgoing(s.ctx); mdok {
					if s.responseMd != nil {
						maps.Copy(s.responseMd, outMD)
					}
				}
			case err := <-s.errCh:
				return err
			case <-s.doneCh:
				return io.EOF
			case <-s.ctx.Done():
				return s.ctx.Err()
			}
		} else {
			// Server receives from clientToServer channel
			select {
			case received, chanOk = <-s.clientToServer:
			case <-s.doneCh:
				return io.EOF
			case <-s.ctx.Done():
				return s.ctx.Err()
			}
		}

		if !chanOk {
			return io.EOF
		}

		return convertReceived(msg, received)
	}
}

// MsgSend sends a message through the stream.
func (s *Stream) MsgSend(msg drpc.Message, _ drpc.Encoding) error {
	return s.Send(msg)
}

// MsgRecv receives a message from the stream.
func (s *Stream) MsgRecv(msg drpc.Message, _ drpc.Encoding) error {
	return s.Recv(msg)
}

// Close closes the stream.
func (s *Stream) Close() error {
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
func (s *Stream) CloseSend() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	// Close only the appropriate send channel
	if s.isClient {
		close(s.clientToServer)
	}

	return nil
}

// CreateClientServerPair creates a pair of connected streams for client and server.
func CreateClientServerPair(ctx context.Context, endpoint string) (*Stream, *Stream) {
	clientToServer := make(chan any, 10)
	serverToClient := make(chan any, 10)

	clientStream := &Stream{
		ctx:            ctx,
		endpoint:       endpoint,
		clientToServer: clientToServer,
		serverToClient: serverToClient,
		errCh:          make(chan error, 1),
		doneCh:         make(chan struct{}),
		isClient:       true,
	}

	serverStream := &Stream{
		ctx:            ctx,
		endpoint:       endpoint,
		clientToServer: clientToServer,
		serverToClient: serverToClient,
		errCh:          make(chan error, 1),
		doneCh:         make(chan struct{}),
		isClient:       false,
	}

	return clientStream, serverStream
}
