package tests

import (
	"context"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

<<<<<<< HEAD
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/log"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/types"
	mtls "go-micro.dev/v5/util/tls"

	mgrpc "github.com/go-micro/plugins/server/grpc"
	"github.com/go-micro/plugins/server/grpc/tests/handler"
	"github.com/go-micro/plugins/server/grpc/tests/proto"
	tgrpc "github.com/go-micro/plugins/server/grpc/tests/util/grpc"

	_ "github.com/go-micro/plugins/codecs/yaml"
	_ "github.com/go-micro/plugins/config/source/file"
	_ "github.com/go-micro/plugins/log/text"
=======
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	mtls "github.com/go-orb/go-orb/util/tls"

	mgrpc "github.com/go-orb/plugins/server/grpc"
	"github.com/go-orb/plugins/server/grpc/tests/handler"
	"github.com/go-orb/plugins/server/grpc/tests/proto"
	tgrpc "github.com/go-orb/plugins/server/grpc/tests/util/grpc"

	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/text"
>>>>>>> 3191204 (feat: update to go-orb/go-orb)
)

func init() {
	server.Handlers.Register("Streams",
		server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
			proto.RegisterStreamsServer,
			new(handler.EchoHandler),
		))
}

var _ proto.StreamsServer = (*handler.EchoHandler)(nil)

func TestGrpc(t *testing.T) {
	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithInsecure(true),
		mgrpc.WithTimeout(2),
		mgrpc.WithRegistration(
			"Streams",
			// Why do we have to explicitly set the generic type here, makes no sense?
			server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
				proto.RegisterStreamsServer, new(handler.EchoHandler)),
		),
	)
	require.NoError(t, err, "setup server")
	defer cleanup(t)

	assert.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", nil), "make request")
	assert.NoError(t, tgrpc.MakeStreamRequest(srv.Address(), "Alex", 5, nil), "stream request")
}

func TestGrpcTLS(t *testing.T) {
	addr := "127.0.0.1:42069"
	tlsConfig, err := mtls.GenTLSConfig(addr)
	require.NoError(t, err, "generate TLS config")

	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithTimeout(2),
		mgrpc.WithAddress(addr),
		mgrpc.WithTLS(tlsConfig),
		mgrpc.WithRegistration(
			"Streams",
			// Why do we have to explicitly set the generic type here, makes no sense?
			server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
				proto.RegisterStreamsServer, new(handler.EchoHandler)),
		),
	)
	require.NoError(t, err, "setup server")
	defer cleanup(t)

	assert.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", tlsConfig), "unary request")
	assert.NoError(t, tgrpc.MakeStreamRequest(srv.Address(), "Alex", 5, tlsConfig), "stream request")
}

func TestGrpcMiddleware(t *testing.T) {
	var unaryC atomic.Int64
	var streamC atomic.Int64

	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithInsecure(true),
		mgrpc.WithUnaryInterceptor("middleware-1", tgrpc.NewUnaryMiddlware(&unaryC)),
		mgrpc.WithUnaryInterceptor("middleware-2", tgrpc.NewUnaryMiddlware(&unaryC)),
		mgrpc.WithStreamInterceptor("middleware-1", tgrpc.NewStreamMiddleware(&streamC)),
		mgrpc.WithStreamInterceptor("middleware-2", tgrpc.NewStreamMiddleware(&streamC)),
		mgrpc.WithRegistration(
			"Streams",
			// Why do we have to explicitly set the generic type here, makes no sense?
			server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
				proto.RegisterStreamsServer, new(handler.EchoHandler)),
		),
	)
	require.NoError(t, err, "setup server")
	defer cleanup(t)

	assert.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", nil), "unary request")
	assert.EqualValues(t, 2, unaryC.Load())

	assert.NoError(t, tgrpc.MakeStreamRequest(srv.Address(), "Alex", 5, nil), "stream request")
	assert.EqualValues(t, 2, streamC.Load())
}

func TestGrpcStartStop(t *testing.T) {
	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithInsecure(true),
		mgrpc.WithTimeout(2),
		mgrpc.WithRegistration(
			"Streams",
			// Why do we have to explicitly set the generic type here, makes no sense?
			server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
				proto.RegisterStreamsServer, new(handler.EchoHandler)),
		),
	)
	require.NoError(t, err, "setup server")

	assert.NoError(t, srv.Start(), "start server 1")
	assert.NoError(t, srv.Start(), "start server 2")
	assert.NoError(t, srv.Start(), "start server 3")

	assert.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", nil), "make request")
	assert.NoError(t, tgrpc.MakeStreamRequest(srv.Address(), "Alex", 5, nil), "stream request")

	cleanup(t)
	cleanup(t)
	cleanup(t)
}

func TestGrpcIntegration(t *testing.T) {
	name := types.ServiceName("com.example.test")

	logger, err := log.ProvideLogger(name, nil)
	require.NoError(t, err, "failed to setup logger")

	srv, err := server.ProvideServer(name, nil, logger,
		mgrpc.WithDefaults(
			mgrpc.WithGRPCReflection(false),
			mgrpc.WithInsecure(true),
		),
		mgrpc.WithEntrypoint(
			mgrpc.WithName("test-ep-1"),
			mgrpc.WithGRPCReflection(true),
			mgrpc.WithRegistration(
				"Streams",
				// Why do we have to explicitly set the generic type here, makes no sense?
				server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
					proto.RegisterStreamsServer, new(handler.EchoHandler)),
			),
		),
		mgrpc.WithEntrypoint(
			mgrpc.WithName("test-ep-2"),
			mgrpc.WithRegistration(
				"Streams",
				// Why do we have to explicitly set the generic type here, makes no sense?
				server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
					proto.RegisterStreamsServer, new(handler.EchoHandler)),
			),
		),
	)
	require.NoError(t, err, "failed to setup server")
	require.NoError(t, srv.Start(), "failed to start server")

	e, err := srv.GetEntrypoint("test-ep-1")
	require.NoError(t, err, "failed to fetch entrypoint 1")
	ep := e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, true, ep.Config().Reflection, "server 1 reflection")
	require.Equal(t, true, ep.Config().HealthService, "server 1 health")
	require.Equal(t, true, ep.Config().Insecure, "server 1 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")

	e, err = srv.GetEntrypoint("test-ep-2")
	require.NoError(t, err, "failed to fetch entrypoint 2")
	ep = e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, false, ep.Config().Reflection, "server 2 reflection")
	require.Equal(t, true, ep.Config().HealthService, "server 2 health")
	require.Equal(t, true, ep.Config().Insecure, "server 2 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, srv.Stop(ctx))
}

func TestServerFileConfig(t *testing.T) {
	var (
		counter1  atomic.Int64
		counter2  atomic.Int64
		counter3  atomic.Int64
		counter4  atomic.Int64
		counterS1 atomic.Int64
	)

	server.Handlers.Register("handler-1", func(_ any) {})
	server.Handlers.Register("handler-2", func(_ any) {})
	mgrpc.UnaryInterceptors.Register("middleware-1", tgrpc.NewUnaryMiddlware(&counter1))
	mgrpc.UnaryInterceptors.Register("middleware-2", tgrpc.NewUnaryMiddlware(&counter2))
	mgrpc.UnaryInterceptors.Register("middleware-3", tgrpc.NewUnaryMiddlware(&counter3))
	mgrpc.UnaryInterceptors.Register("middleware-4", tgrpc.NewUnaryMiddlware(&counter4))
	mgrpc.StreamInterceptors.Register("middleware-S1", tgrpc.NewStreamMiddleware(&counterS1))

	name := types.ServiceName("com.example.test")

	fURL, err := url.Parse("file://config/config.yaml")
	require.NoError(t, err, "failed to parse file config url")
	// litter.Dump(fURL)
	// t.Logf("%+v", fURL.Host)

	config, err := config.Read([]*url.URL{fURL}, nil)
	require.NoError(t, err, "failed to read file config")

	logger, err := log.ProvideLogger(name, nil)
	require.NoError(t, err, "failed to setup logger")

	srv, err := server.ProvideServer(name, config, logger,
		mgrpc.WithEntrypoint(
			mgrpc.WithName("static-ep-1"),
			mgrpc.WithAddress(":48081"),
		),
		mgrpc.WithEntrypoint(
			mgrpc.WithName("test-ep-5"),
			mgrpc.WithRegistration(
				"Streams",
				// Why do we have to explicitly set the generic type here, makes no sense?
				server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
					proto.RegisterStreamsServer, new(handler.EchoHandler)),
			),
		),
	)

	require.NoError(t, err, "failed to setup server")
	require.NoError(t, srv.Start(), "failed to start server")

	e, err := srv.GetEntrypoint("static-ep-1")
	require.NoError(t, err, "failed to fetch static ep 1")
	ep := e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, true, ep.Config().Insecure, "static 1 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")
	require.EqualValues(t, 1, counter1.Load(), "counter 1, static ep 1")
	require.EqualValues(t, 1, counter2.Load(), "counter 2, static ep 1")
	require.EqualValues(t, 1, counterS1.Load(), "counter S1, static ep 1")
	require.Equal(t, 2, ep.Config().UnaryInterceptors.Len())
	require.Equal(t, 1, ep.Config().StreamInterceptors.Len())

	e, err = srv.GetEntrypoint("test-ep-1")
	require.NoError(t, err, "failed to fetch ep 1")
	ep = e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, true, ep.Config().Insecure, "server 1 insecure")
	require.Equal(t, false, ep.Config().Reflection, "server 1 reflection")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")
	require.EqualValues(t, 2, counter1.Load(), "counter 1, ep 1")
	require.EqualValues(t, 2, counter2.Load(), "counter 2, ep 1")
	require.EqualValues(t, 2, counterS1.Load(), "counter S1, ep 1")
	require.Equal(t, 2, ep.Config().UnaryInterceptors.Len())
	require.Equal(t, 1, ep.Config().StreamInterceptors.Len())

	e, err = srv.GetEntrypoint("test-ep-2")
	require.NoError(t, err, "failed to fetch ep 2")
	ep = e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, true, ep.Config().Insecure, "server 2 insecure")
	require.Equal(t, false, ep.Config().HealthService, "server 2 health")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")
	require.EqualValues(t, 3, counter1.Load(), "counter 1, ep 2")
	require.EqualValues(t, 3, counter2.Load(), "counter 2, ep 2")
	require.EqualValues(t, 3, counterS1.Load(), "counter S1, ep 2")
	require.Equal(t, 2, ep.Config().UnaryInterceptors.Len())
	require.Equal(t, 1, ep.Config().StreamInterceptors.Len())

	e, err = srv.GetEntrypoint("test-ep-3")
	require.NoError(t, err, "failed to fetch ep 3")
	ep = e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, true, ep.Config().Insecure, "server 3 insecure")
	require.Equal(t, false, ep.Config().HealthService, "server 3 health")
	require.Equal(t, false, ep.Config().Reflection, "server 3 reflection")
	require.Equal(t, time.Second*11, ep.Config().Timeout, "server 3 timeout")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")
	require.EqualValues(t, 4, counter1.Load(), "counter 1, ep 3")
	require.EqualValues(t, 4, counter2.Load(), "counter 2, ep 3")
	require.EqualValues(t, 4, counterS1.Load(), "counter S1, ep 3")
	require.Equal(t, 2, ep.Config().UnaryInterceptors.Len())
	require.Equal(t, 1, ep.Config().StreamInterceptors.Len())

	_, err = srv.GetEntrypoint("test-ep-4")
	require.Error(t, err, "should fail to fetch ep 4")

	e, err = srv.GetEntrypoint("test-ep-5")
	require.NoError(t, err, "failed to fetch ep 5")
	ep = e.(*mgrpc.ServerGRPC) //nolint:errcheck
	require.Equal(t, true, ep.Config().Insecure, "server 5 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
	require.NoError(t, tgrpc.MakeStreamRequest(ep.Address(), "Alex", 5, nil), "stream request")
	require.EqualValues(t, 5, counter1.Load(), "counter 1, ep 5")
	require.EqualValues(t, 5, counter2.Load(), "counter 2, ep 5")
	require.EqualValues(t, 1, counter4.Load(), "counter 4, ep 5")
	require.EqualValues(t, 5, counterS1.Load(), "counter S1, ep 5")
	require.Equal(t, 3, ep.Config().UnaryInterceptors.Len())
	require.Equal(t, 1, ep.Config().StreamInterceptors.Len())
	require.Equal(t, 3, len(ep.Config().HandlerRegistrations))
}
