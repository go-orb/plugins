// Package grpc provides utilities to test the gRPC server.
package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"

	mgrpc "github.com/go-orb/plugins/server/grpc"
	"github.com/go-orb/plugins/server/grpc/tests/proto"
)

// SetupServer will create a gRPC test server.
func SetupServer(opts ...server.Option) (server.Entrypoint, func(t *testing.T), error) {
	logger, err := log.New()
	if err != nil {
		return nil, nil, fmt.Errorf("setup logger: %w", err)
	}

	components := types.NewComponents()

	reg, err := registry.New(nil, components, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("setup registry: %w", err)
	}

	srv, err := mgrpc.New("app", "v1.0.0", "", mgrpc.NewConfig(opts...), logger, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("setup gRPC server: %w", err)
	}

	if err := srv.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("start: %w", err)
	}

	cleanup := func(t *testing.T) {
		t.Helper()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := srv.Stop(ctx); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	}

	return srv, cleanup, nil
}

// MakeRequest makes a test request to the Echo endpoint.
func MakeRequest(addr, name string, tlsConfig *tls.Config) error {
	conn, err := dial(addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	resp := &proto.CallResponse{}

	if err = conn.Invoke(context.Background(), "/echo.Streams/Call", &proto.CallRequest{
		Name: name,
	}, resp); err != nil {
		return fmt.Errorf("grpc call: %w", err)
	}

	if resp.GetMsg() != "Hello "+name {
		return errors.New("message invalid")
	}

	return nil
}

func dial(addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{}

	if tlsConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return conn, nil
}

// NewUnaryMiddlware creates a new unary test middleware.
func NewUnaryMiddlware(i *atomic.Int64) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		i.Add(1)
		return handler(ctx, req)
	}
}

// NewStreamMiddleware creates a new stream test middleware.
func NewStreamMiddleware(i *atomic.Int64) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		i.Add(1)
		return handler(srv, ss)
	}
}
