package tests

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	mtls "github.com/go-orb/go-orb/util/tls"

	mgrpc "github.com/go-orb/plugins/server/grpc"
	"github.com/go-orb/plugins/server/grpc/tests/handler"
	"github.com/go-orb/plugins/server/grpc/tests/proto"
	tgrpc "github.com/go-orb/plugins/server/grpc/tests/util/grpc"

	_ "github.com/go-orb/plugins-experimental/registry/mdns"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/slog"
)

func init() {
	server.Handlers.Set("Streams", proto.RegisterStreamsHandler(new(handler.EchoHandler)))
}

var _ proto.StreamsServer = (*handler.EchoHandler)(nil)

func TestGrpc(t *testing.T) {
	h, _ := server.Handlers.Get("Streams")
	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithInsecure(),
		mgrpc.WithTimeout(2),
		mgrpc.WithHandlers(h),
	)
	require.NoError(t, err, "setup server")
	defer cleanup(t)

	require.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", nil), "make request")
}

func TestGrpcTLS(t *testing.T) {
	addr := "127.0.0.1:43069"
	tlsConfig, err := mtls.GenTLSConfig(addr)
	require.NoError(t, err, "generate TLS config")

	h, _ := server.Handlers.Get("Streams")
	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithTimeout(2),
		mgrpc.WithAddress(addr),
		mgrpc.WithTLS(tlsConfig),
		mgrpc.WithHandlers(h),
	)
	require.NoError(t, err, "setup server")
	defer cleanup(t)

	require.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", tlsConfig), "unary request")
}

func TestGrpcStartStop(t *testing.T) {
	h, _ := server.Handlers.Get("Streams")
	srv, cleanup, err := tgrpc.SetupServer(
		mgrpc.WithInsecure(),
		mgrpc.WithTimeout(2),
		mgrpc.WithHandlers(h),
	)
	require.NoError(t, err, "setup server")

	require.NoError(t, srv.Start(context.Background()), "start server 1")
	require.NoError(t, srv.Start(context.Background()), "start server 2")
	require.NoError(t, srv.Start(context.Background()), "start server 3")

	require.NoError(t, tgrpc.MakeRequest(srv.Address(), "Alex", nil), "make request")

	cleanup(t)
	cleanup(t)
	cleanup(t)
}

func TestGrpcIntegration(t *testing.T) {
	name := "com.example.test"
	version := "v1.0.0"

	components := types.NewComponents()

	logger, err := log.New()
	require.NoError(t, err, "failed to setup logger")

	reg, err := registry.New(name, version, nil, components, logger)
	require.NoError(t, err, "failed to setup the registry")

	h, _ := server.Handlers.Get("Streams")
	srv, err := server.New(nil, logger, reg,
		server.WithEntrypointConfig(mgrpc.NewConfig(
			mgrpc.WithName("test-ep-1"),
			mgrpc.WithReflection(true),
			mgrpc.WithHandlers(h),
			mgrpc.WithInsecure(),
			mgrpc.WithReflection(true),
			mgrpc.WithHealthService(true),
		)),
		server.WithEntrypointConfig(mgrpc.NewConfig(
			mgrpc.WithName("test-ep-2"),
			mgrpc.WithHandlers(h),
			mgrpc.WithInsecure(),
			mgrpc.WithReflection(false),
			mgrpc.WithHealthService(true),
		)),
	)
	require.NoError(t, err, "failed to setup server")
	require.NoError(t, srv.Start(context.Background()), "failed to start server")

	e, err := srv.GetEntrypoint("test-ep-1")
	require.NoError(t, err, "failed to fetch entrypoint 1")
	ep := e.(*mgrpc.Server) //nolint:errcheck
	require.True(t, ep.Config().Reflection, "server 1 reflection")
	require.True(t, ep.Config().HealthService, "server 1 health")
	require.True(t, ep.Config().Insecure, "server 1 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")

	e, err = srv.GetEntrypoint("test-ep-2")
	require.NoError(t, err, "failed to fetch entrypoint 2")
	ep = e.(*mgrpc.Server) //nolint:errcheck
	require.False(t, ep.Config().Reflection, "server 2 reflection")
	require.True(t, ep.Config().HealthService, "server 2 health")
	require.True(t, ep.Config().Insecure, "server 2 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, srv.Stop(ctx))
}

func TestServerFileConfig(t *testing.T) {
	server.Handlers.Set("handler-1", func(_ any) {})
	server.Handlers.Set("handler-2", func(_ any) {})

	name := "com.example.test"
	version := "v1.0.0"

	fURL, err := url.Parse("file://config/config.yaml")
	require.NoError(t, err, "failed to parse file config url")
	// litter.Dump(fURL)
	// t.Logf("%+v", fURL.Host)

	configData, err := config.Read(fURL)
	require.NoError(t, err, "failed to read file config")

	configData, err = config.WalkMap(types.SplitServiceName(name), configData)
	require.NoError(t, err, "failed to walk config")

	logger, err := log.New()
	require.NoError(t, err, "failed to setup logger")

	reg, err := registry.New(name, version, nil, &types.Components{}, logger)
	require.NoError(t, err, "failed to setup the registry")

	h, _ := server.Handlers.Get("Streams")
	srv, err := server.New(configData, logger, reg,
		server.WithEntrypointConfig(mgrpc.NewConfig(
			mgrpc.WithName("static-ep-1"),
			mgrpc.WithAddress(":48081"),
			mgrpc.WithInsecure(),
			mgrpc.WithHandlers(h),
		)),
		server.WithEntrypointConfig(mgrpc.NewConfig(
			mgrpc.WithName("test-ep-5"),
			mgrpc.WithHandlers(h),
		)),
	)

	require.NoError(t, err, "failed to setup server")
	require.NoError(t, srv.Start(context.Background()), "failed to start server")

	e, err := srv.GetEntrypoint("static-ep-1")
	require.NoError(t, err, "failed to fetch static ep 1")
	ep := e.(*mgrpc.Server) //nolint:errcheck
	require.True(t, ep.Config().Insecure, "static 1 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")

	e, err = srv.GetEntrypoint("test-ep-1")
	require.NoError(t, err, "failed to fetch ep 1")
	ep = e.(*mgrpc.Server) //nolint:errcheck
	require.True(t, ep.Config().Insecure, "server 1 insecure")
	require.False(t, ep.Config().Reflection, "server 1 reflection")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")

	e, err = srv.GetEntrypoint("test-ep-2")
	require.NoError(t, err, "failed to fetch ep 2")
	ep = e.(*mgrpc.Server) //nolint:errcheck
	require.True(t, ep.Config().Insecure, "server 2 insecure")
	require.False(t, ep.Config().HealthService, "server 2 health")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")

	e, err = srv.GetEntrypoint("test-ep-3")
	require.NoError(t, err, "failed to fetch ep 3")
	ep = e.(*mgrpc.Server) //nolint:errcheck
	require.True(t, ep.Config().Insecure, "server 3 insecure")
	require.False(t, ep.Config().HealthService, "server 3 health")
	require.False(t, ep.Config().Reflection, "server 3 reflection")
	require.Equal(t, config.Duration(time.Second*11), ep.Config().Timeout, "server 3 timeout")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")

	_, err = srv.GetEntrypoint("test-ep-4")
	require.Error(t, err, "should fail to fetch ep 4")

	e, err = srv.GetEntrypoint("test-ep-5")
	require.NoError(t, err, "failed to fetch ep 5")
	ep = e.(*mgrpc.Server) //nolint:errcheck
	require.True(t, ep.Config().Insecure, "server 5 insecure")
	require.NoError(t, tgrpc.MakeRequest(ep.Address(), "Alex", nil), "make request")
}
