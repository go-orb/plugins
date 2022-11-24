package grpc

import (
	"context"

	"google.golang.org/grpc"
)

func (s *ServerGRPC) unaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var cancel func()
		if s.config.Timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
			defer cancel()
		}

		// Directly execute handler if no middleware is defined.
		if s.unaryMiddleware == 0 {
			return handler(ctx, req)
		}

		h := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}

		// Add user defined middleware if the route requires.
		if next := s.config.UnaryInterceptors.Match(info.FullMethod); len(next) > 0 {
			next = append(next, h)
			h = chainUnaryInterceptors(next)
		}

		return h(ctx, req, info, handler)
	}
}

func (s *ServerGRPC) streamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		h := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}

		// Add user defined middleware if the route requires.
		if next := s.config.StreamInterceptors.Match(info.FullMethod); len(next) > 0 {
			next = append(next, h)
			h = chainStreamInterceptors(next)
		}

		return h(srv, ss, info, handler)
	}
}

// Source: from https://github.com/grpc/grpc-go/blob/v1.51.0/server.go
func chainUnaryInterceptors(interceptors []grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var state struct {
			i    int
			next grpc.UnaryHandler
		}

		state.next = func(ctx context.Context, req any) (any, error) {
			if state.i == len(interceptors)-1 {
				return interceptors[state.i](ctx, req, info, handler)
			}

			state.i++

			return interceptors[state.i-1](ctx, req, info, state.next)
		}

		return state.next(ctx, req)
	}
}

// Source: from https://github.com/grpc/grpc-go/blob/v1.51.0/server.go
func chainStreamInterceptors(interceptors []grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var state struct {
			i    int
			next grpc.StreamHandler
		}

		state.next = func(srv interface{}, ss grpc.ServerStream) error {
			if state.i == len(interceptors)-1 {
				return interceptors[state.i](srv, ss, info, handler)
			}

			state.i++

			return interceptors[state.i-1](srv, ss, info, state.next)
		}

		return state.next(srv, ss)
	}
}
