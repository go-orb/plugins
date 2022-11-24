package grpc

import (
	"go-micro.dev/v5/util/container"
	"google.golang.org/grpc"
)

var (
	// UnaryInterceptors is a plugin container for unary interceptors middleware.
	UnaryInterceptors = container.NewPlugins[grpc.UnaryServerInterceptor]()

	// StreamInterceptors is a plugin container for streaming interceptors middleware.
	StreamInterceptors = container.NewPlugins[grpc.StreamServerInterceptor]()
)
