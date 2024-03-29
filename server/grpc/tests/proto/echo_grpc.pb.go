// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.7
// source: echo.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// StreamsClient is the client API for Streams service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StreamsClient interface {
	Call(ctx context.Context, in *CallRequest, opts ...grpc.CallOption) (*CallResponse, error)
	Stream(ctx context.Context, opts ...grpc.CallOption) (Streams_StreamClient, error)
}

type streamsClient struct {
	cc grpc.ClientConnInterface
}

func NewStreamsClient(cc grpc.ClientConnInterface) StreamsClient {
	return &streamsClient{cc}
}

func (c *streamsClient) Call(ctx context.Context, in *CallRequest, opts ...grpc.CallOption) (*CallResponse, error) {
	out := new(CallResponse)
	err := c.cc.Invoke(ctx, "/echo.Streams/Call", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *streamsClient) Stream(ctx context.Context, opts ...grpc.CallOption) (Streams_StreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &Streams_ServiceDesc.Streams[0], "/echo.Streams/Stream", opts...)
	if err != nil {
		return nil, err
	}
	x := &streamsStreamClient{stream}
	return x, nil
}

type Streams_StreamClient interface {
	Send(*CallRequest) error
	Recv() (*CallResponse, error)
	grpc.ClientStream
}

type streamsStreamClient struct {
	grpc.ClientStream
}

func (x *streamsStreamClient) Send(m *CallRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *streamsStreamClient) Recv() (*CallResponse, error) {
	m := new(CallResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// StreamsServer is the server API for Streams service.
// All implementations must embed UnimplementedStreamsServer
// for forward compatibility
type StreamsServer interface {
	Call(context.Context, *CallRequest) (*CallResponse, error)
	Stream(Streams_StreamServer) error
	mustEmbedUnimplementedStreamsServer()
}

// UnimplementedStreamsServer must be embedded to have forward compatible implementations.
type UnimplementedStreamsServer struct {
}

func (UnimplementedStreamsServer) Call(context.Context, *CallRequest) (*CallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Call not implemented")
}
func (UnimplementedStreamsServer) Stream(Streams_StreamServer) error {
	return status.Errorf(codes.Unimplemented, "method Stream not implemented")
}
func (UnimplementedStreamsServer) mustEmbedUnimplementedStreamsServer() {}

// UnsafeStreamsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StreamsServer will
// result in compilation errors.
type UnsafeStreamsServer interface {
	mustEmbedUnimplementedStreamsServer()
}

func RegisterStreamsServer(s grpc.ServiceRegistrar, srv StreamsServer) {
	s.RegisterService(&Streams_ServiceDesc, srv)
}

func _Streams_Call_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CallRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamsServer).Call(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/echo.Streams/Call",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamsServer).Call(ctx, req.(*CallRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Streams_Stream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(StreamsServer).Stream(&streamsStreamServer{stream})
}

type Streams_StreamServer interface {
	Send(*CallResponse) error
	Recv() (*CallRequest, error)
	grpc.ServerStream
}

type streamsStreamServer struct {
	grpc.ServerStream
}

func (x *streamsStreamServer) Send(m *CallResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *streamsStreamServer) Recv() (*CallRequest, error) {
	m := new(CallRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Streams_ServiceDesc is the grpc.ServiceDesc for Streams service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Streams_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "echo.Streams",
	HandlerType: (*StreamsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Call",
			Handler:    _Streams_Call_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Stream",
			Handler:       _Streams_Stream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "echo.proto",
}
