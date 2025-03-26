package kvstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/codecs/json"
	"github.com/go-orb/plugins/kvstore/natsjs"
	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

func createRegistry(suite *tests.TestSuite) (registry.Registry, error) {
	t := suite.T()
	t.Helper()

	cfg := NewConfig()
	store, ok := suite.Server.(kvstore.Type)
	if !ok {
		return nil, errors.New("while retrieving the store")
	}
	reg, err := New(cfg, suite.Logger.With("reg", "custom"), store)
	require.NoError(t, err, "while creating a registry")
	err = reg.Start(suite.Ctx)
	require.NoError(t, err, "while starting a registry")

	return reg, nil
}

func createSuite(tb testing.TB) (*tests.TestSuite, func() error) {
	tb.Helper()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

	// Start embedded NATS server for testing
	tmpDir := tb.TempDir()

	opts := test.DefaultTestOptions
	opts.Port = -1 // Random port
	opts.JetStream = true
	opts.StoreDir = tmpDir
	// Configure JetStream
	opts.JetStreamMaxMemory = -1 // Unlimited
	opts.JetStreamMaxStore = -1  // Unlimited

	server := test.RunServer(&opts)
	require.True(tb, server.JetStreamEnabled())

	// Create logger
	logger, err := log.New(log.WithLevel("TRACE"))
	require.NoError(tb, err)

	// Initialize the store
	storeCfg := natsjs.NewConfig(
		natsjs.WithURL(server.ClientURL()),
	)

	// Create store
	store, err := natsjs.New(storeCfg, logger)
	require.NoError(tb, err)

	err = store.Start(ctx)
	require.NoError(tb, err)

	// Create first registry without caching
	cfg1 := NewConfig(WithNoCache())
	reg1, err := New(cfg1, logger.With("reg", "reg1"), kvstore.Type{KVStore: store})
	require.NoError(tb, err)
	require.NoError(tb, reg1.Start(ctx))

	// Create second registry with caching
	cfg2 := NewConfig()
	reg2, err := New(cfg2, logger.With("reg", "reg2"), kvstore.Type{KVStore: store})
	require.NoError(tb, err)
	require.NoError(tb, reg2.Start(ctx))

	cleanup := func() error {
		cancel()

		ctx = context.Background()

		_ = store.Stop(ctx) //nolint:errcheck
		server.Shutdown()
		return nil
	}

	r := &tests.TestSuite{
		Server:         kvstore.Type{KVStore: store},
		Ctx:            ctx,
		Logger:         logger,
		Registries:     []registry.Registry{reg1, reg2},
		UpdateTime:     time.Second,
		CreateRegistry: createRegistry,
	}

	return r, cleanup
}

func TestSuite(t *testing.T) {
	s, cleanup := createSuite(t)

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	s, cleanup := createSuite(b)

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkParallelGetService(b *testing.B) {
	s, cleanup := createSuite(b)

	s.BenchmarkParallelGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	s, cleanup := createSuite(b)

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
