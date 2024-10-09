// Code generated by protoc-gen-go-orb. DO NOT EDIT.
//
// version:
// - protoc-gen-go-orb        v0.0.1
// - protoc                   v5.28.0
//
// Proto source: echo.proto

package proto

import (
	"context"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"

	grpc "google.golang.org/grpc"
)

// HandlerStreams is the name of a service, it's here to static type/reference.
const HandlerStreams = "echo.Streams"

// StreamsClient is the client for echo.Streams
type StreamsClient struct {
	client client.Client
}

// NewStreamsClient creates a new client for echo.Streams
func NewStreamsClient(client client.Client) *StreamsClient {
	return &StreamsClient{client: client}
}

// Call calls Call.
func (c *StreamsClient) Call(ctx context.Context, service string, req *CallRequest, opts ...client.CallOption) (*CallResponse, error) {
	return client.Call[CallResponse](ctx, c.client, service, "echo.Streams/Call", req, opts...)
}

// StreamsHandler is the Handler for echo.Streams
type StreamsHandler interface {
	Call(ctx context.Context, req *CallRequest) (*CallResponse, error)
}

// RegisterStreamsHandler will return a registration function that can be
// provided to entrypoints as a handler registration.
func RegisterStreamsHandler(handler StreamsHandler) server.RegistrationFunc {
	return func(s any) {
		switch srv := s.(type) {

		case grpc.ServiceRegistrar:
			registerStreamsGRPCHandler(srv, handler)
		default:
			log.Warn("No provider for this server found", "proto", "echo.proto", "handler", "Streams", "server", s)
		}
	}
}