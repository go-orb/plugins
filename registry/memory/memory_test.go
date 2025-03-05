package memory

import (
	"context"
	"testing"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

func createRegistries() (*tests.TestSuite, func() error, error) {
	ctx := context.Background()

	logger, err := log.New()
	if err != nil {
		log.Error("failed to create logger", "err", err)
		return nil, func() error { return nil }, err
	}

	cfg1 := NewConfig()
	reg1 := Instance("orb.service.test1", "unset", cfg1, logger)
	if err := reg1.Start(ctx); err != nil {
		log.Error("failed to connect registry one to Consul server", "err", err)
		return nil, func() error { return nil }, err
	}

	cfg2 := NewConfig()
	reg2 := Instance("orb.service.test2", "unset", cfg2, logger)
	if err := reg2.Start(ctx); err != nil {
		log.Error("failed to connect registry two to Consul server", "err", err)
		return nil, func() error { return nil }, err
	}

	cfg3 := NewConfig()
	reg3 := Instance("orb.service.test3", "unset", cfg3, logger)
	if err := reg3.Start(ctx); err != nil {
		log.Error("failed to connect registry three to Consul server", "err", err)
		return nil, func() error { return nil }, err
	}

	cleanup := func() error {
		ctx := context.Background()
		_ = reg1.Stop(ctx) //nolint:errcheck
		return nil
	}

	return tests.CreateSuite(logger, []registry.Registry{reg1, reg2, reg3}, 0, 0), cleanup, nil
}

func TestSuite(t *testing.T) {
	s, cleanup, err := createRegistries()
	require.NoError(t, err, "while creating a server")

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	s, cleanup, err := createRegistries()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkParallelGetService(b *testing.B) {
	s, cleanup, err := createRegistries()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	s, cleanup, err := createRegistries()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
