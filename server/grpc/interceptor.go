package grpc

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	gmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

//nolint:gochecknoglobals
var stdHeaders = []string{"content-type", "user-agent"}

func (s *Server) unaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, reqMd := metadata.WithIncoming(ctx)
		ctx, outMd := metadata.WithOutgoing(ctx)

		// Copy incoming metadata from grpc to orb.
		if gReqMd, ok := gmetadata.FromIncomingContext(ctx); ok {
			for k, v := range gReqMd {
				if slices.Contains(stdHeaders, k) {
					continue
				}

				reqMd[k] = v[0]
			}
		}

		fmSplit := strings.Split(info.FullMethod, "/")
		if len(fmSplit) >= 3 {
			reqMd[metadata.Service] = fmSplit[1]
			reqMd[metadata.Method] = fmSplit[2]
		}

		var cancel func()
		if s.config.Timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(s.config.Timeout))
			defer cancel()
		}

		h := func(ctx context.Context, req any) (any, error) {
			return handler(ctx, req)
		}
		for _, m := range s.config.OptMiddlewares {
			h = m.Call(h)
		}

		result, err := h(ctx, req)

		if len(outMd) > 0 {
			gOutMd := make(gmetadata.MD)

			for k, v := range outMd {
				gOutMd[k] = []string{v}
			}

			if err := grpc.SendHeader(ctx, gOutMd); err != nil {
				return nil, status.Errorf(codes.Internal, "internal error while sending headers")
			}
		}

		if err != nil {
			oErr := orberrors.From(err)
			gCode := HTTPStatusToCode(oErr.Code)

			if oErr.Wrapped != nil {
				return nil, status.Errorf(gCode, "%s: %s", oErr.Message, oErr.Wrapped.Error())
			}

			return nil, status.Errorf(gCode, "%s", oErr.Message)
		}

		return result, nil
	}
}

type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWrapper) Context() context.Context {
	return s.ctx
}

func (s *serverStreamWrapper) SendMsg(msg interface{}) error {
	outMd, ok := metadata.Outgoing(s.ctx)

	if ok && len(outMd) > 0 {
		gOutMd := make(gmetadata.MD)

		for k, v := range outMd {
			gOutMd[k] = []string{v}
		}

		if err := grpc.SendHeader(s.ctx, gOutMd); err != nil {
			return status.Errorf(codes.Internal, "internal error while sending headers")
		}
	}

	return s.ServerStream.SendMsg(msg)
}

func (s *Server) streamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, serverStream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, reqMd := metadata.WithIncoming(serverStream.Context())
		ctx, _ = metadata.WithOutgoing(ctx)

		// Copy incoming metadata from grpc to orb.
		if gReqMd, ok := gmetadata.FromIncomingContext(ctx); ok {
			for k, v := range gReqMd {
				if slices.Contains(stdHeaders, k) {
					continue
				}

				reqMd[k] = v[0]
			}
		}

		fmSplit := strings.Split(info.FullMethod, "/")
		if len(fmSplit) >= 3 {
			reqMd[metadata.Service] = fmSplit[1]
			reqMd[metadata.Method] = fmSplit[2]
		}

		var cancel func()
		if s.config.Timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(s.config.Timeout))
			defer cancel()
		}

		err := handler(srv, &serverStreamWrapper{serverStream, ctx})

		if err != nil {
			oErr := orberrors.From(err)
			gCode := HTTPStatusToCode(oErr.Code)

			if oErr.Wrapped != nil {
				return status.Errorf(gCode, "%s: %s", oErr.Message, oErr.Wrapped.Error())
			}

			return status.Errorf(gCode, "%s", oErr.Message)
		}

		return err
	}
}

// // Source: from https://github.com/grpc/grpc-go/blob/v1.51.0/server.go
// func chainUnaryInterceptors(interceptors []grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
// 	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
// 		var state struct {
// 			i    int
// 			next grpc.UnaryHandler
// 		}

// 		state.next = func(ctx context.Context, req any) (any, error) {
// 			if state.i == len(interceptors)-1 {
// 				return interceptors[state.i](ctx, req, info, handler)
// 			}

// 			state.i++

// 			return interceptors[state.i-1](ctx, req, info, state.next)
// 		}

// 		return state.next(ctx, req)
// 	}
// }

// // Source: from https://github.com/grpc/grpc-go/blob/v1.51.0/server.go
// func chainStreamInterceptors(interceptors []grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
// 	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
// 		var state struct {
// 			i    int
// 			next grpc.StreamHandler
// 		}

// 		state.next = func(srv interface{}, ss grpc.ServerStream) error {
// 			if state.i == len(interceptors)-1 {
// 				return interceptors[state.i](srv, ss, info, handler)
// 			}

// 			state.i++

// 			return interceptors[state.i-1](srv, ss, info, state.next)
// 		}

// 		return state.next(srv, ss)
// 	}
// }
