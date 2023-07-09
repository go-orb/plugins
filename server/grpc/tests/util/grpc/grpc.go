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

	"go-micro.dev/v5/log"

	mgrpc "github.com/go-micro/plugins/server/grpc"
	"github.com/go-micro/plugins/server/grpc/tests/proto"
)

// SetupServer will create a gRPC test server.
func SetupServer(opts ...mgrpc.Option) (*mgrpc.ServerGRPC, func(t *testing.T), error) {
	cfg := mgrpc.NewConfig()

	logger, err := log.New(log.NewConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("setup logger: %w", err)
	}

	srv, err := mgrpc.ProvideServerGRPC("", logger, *cfg, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("setup gRPC server: %w", err)
	}

	if err := srv.Start(); err != nil {
		return nil, nil, fmt.Errorf("start: %w", err)
	}

	cleanup := func(t *testing.T) {
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

	client := proto.NewStreamsClient(conn)

	resp, err := client.Call(context.Background(), &proto.CallRequest{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("grpc call: %w", err)
	}

	if resp.Msg != "Hello "+name {
		return errors.New("message invalid")
	}

	return nil
}

// MakeStreamRequest makes a streaming test request to the Echo endpoint.
func MakeStreamRequest(addr, name string, s int, tlsConfig *tls.Config) error {
	conn, err := dial(addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	client := proto.NewStreamsClient(conn)

	msg, err := client.Stream(context.Background())
	if err != nil {
		return err
	}
	defer msg.CloseSend() //nolint:errcheck,wsl

	for i := 0; i < s; i++ {
		if err := msg.Send(&proto.CallRequest{Name: name}); err != nil {
			return err
		}

		m, err := msg.Recv()
		if err != nil {
			return err
		}

		if m.Msg != "hello "+name {
			return errors.New("invalid message received")
		}
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

	conn, err := grpc.Dial(addr, opts...)
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
