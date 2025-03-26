package memory

import (
	"context"
	"testing"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

func createRegistry(suite *tests.TestSuite) (registry.Registry, error) {
	t := suite.T()
	t.Helper()

	cfg := NewConfig()
	reg := New(cfg, suite.Logger.With("reg", "custom"))
	err := reg.Start(suite.Ctx)
	require.NoError(t, err, "while starting a registry")

	return reg, nil
}

func createSuite(tb testing.TB) (*tests.TestSuite, func() error) {
	tb.Helper()

	ctx := context.Background()

	logger, err := log.New(log.WithLevel(log.LevelTrace))
	require.NoError(tb, err, "while creating a logger")

	cfg1 := NewConfig()
	reg1 := New(cfg1, logger.With("reg", "reg1"))
	require.NoError(tb, reg1.Start(ctx), "while starting registry one")

	cfg2 := NewConfig()
	reg2 := New(cfg2, logger.With("reg", "reg2"))
	require.NoError(tb, reg2.Start(ctx), "while starting registry two")

	cleanup := func() error {
		return nil
	}

	return &tests.TestSuite{
		Ctx:            ctx,
		Logger:         logger,
		Registries:     []registry.Registry{reg1, reg2},
		UpdateTime:     time.Second,
		CreateRegistry: createRegistry,
	}, cleanup
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

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	s, cleanup := createSuite(b)

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
