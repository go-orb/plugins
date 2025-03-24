package kvstore

import (
	"context"
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

//nolint:unparam
func createSuite(tb testing.TB) (*tests.TestSuite, func() error, error) {
	tb.Helper()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

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
	reg1, err := New(cfg1, logger, kvstore.Type{KVStore: store})
	require.NoError(tb, err)
	require.NoError(tb, reg1.Start(ctx))

	// Create second registry with caching
	cfg2 := NewConfig()
	reg2, err := New(cfg2, logger, kvstore.Type{KVStore: store})
	require.NoError(tb, err)
	require.NoError(tb, reg2.Start(ctx))

	cleanup := func() error {
		cancel()

		ctx = context.Background()

		_ = reg1.Stop(ctx)  //nolint:errcheck
		_ = reg2.Stop(ctx)  //nolint:errcheck
		_ = store.Stop(ctx) //nolint:errcheck
		server.Shutdown()
		return nil
	}

	return tests.CreateSuite(logger, []registry.Registry{reg1, reg2}, time.Millisecond*200), cleanup, nil
}

func TestSuite(t *testing.T) {
	s, cleanup, err := createSuite(t)
	require.NoError(t, err, "while creating a server")

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	s, cleanup, err := createSuite(b)
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkParallelGetService(b *testing.B) {
	s, cleanup, err := createSuite(b)
	if err != nil {
		b.Fatal("Error creating registries:", err)
	}

	s.BenchmarkParallelGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	s, cleanup, err := createSuite(b)
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
