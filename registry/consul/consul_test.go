package consul

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

func createServer() (*tests.TestSuite, func() error, error) {
	ctx := context.Background()

	logger, err := log.New(log.WithLevel("TRACE"))
	if err != nil {
		log.Error("failed to create logger", "err", err)
		return nil, func() error { return nil }, err
	}

	server, err := createServer1(&testing.T{})
	if err != nil {
		logger.Error("failed to create a consul server", "err", err)
		return nil, func() error { return nil }, err
	}

	cfg1, err := NewConfig(types.ServiceName("test1.service"), nil, WithAddress(server.HTTPAddr))
	if err != nil {
		log.Error("failed to create config", "err", err)
		server.Stop() //nolint:errcheck
		return nil, func() error { return nil }, err
	}

	reg1 := New("", "", cfg1, logger)
	if err := reg1.Start(ctx); err != nil {
		log.Error("failed to connect registry one to Consul server", "err", err)
		server.Stop() //nolint:errcheck
		return nil, func() error { return nil }, err
	}

	cfg2, err := NewConfig(types.ServiceName("test2.service"), nil, WithAddress(server.HTTPAddr))
	if err != nil {
		log.Error("failed to create config", "err", err)
		server.Stop() //nolint:errcheck
		return nil, func() error { return nil }, err
	}

	reg2 := New("", "", cfg2, logger)
	if err := reg2.Start(ctx); err != nil {
		log.Error("failed to connect registry two to Consul server", "err", err)
		server.Stop() //nolint:errcheck
		return nil, func() error { return nil }, err
	}

	cfg3, err := NewConfig(types.ServiceName("test3.service"), nil, WithAddress(server.HTTPAddr))
	if err != nil {
		log.Error("failed to create config", err)
		server.Stop() //nolint:errcheck
		return nil, func() error { return nil }, err
	}

	reg3 := New("", "", cfg3, logger)
	if err := reg3.Start(ctx); err != nil {
		log.Error("failed to connect registry three to Consul server", "err", err)
		server.Stop() //nolint:errcheck
		return nil, func() error { return nil }, err
	}

	cleanup := func() error {
		_ = server.Stop() //nolint:errcheck
		return nil
	}

	return tests.CreateSuite(logger, []registry.Registry{reg1, reg2, reg3}, 0, 0), cleanup, nil
}

func createServer1(tb testing.TB) (*testutil.TestServer, error) {
	tb.Helper()

	// Compile our consul path.
	myConsulPath, err := filepath.Abs(filepath.Join("./test/bin/", runtime.GOOS+"_"+runtime.GOARCH))
	if err != nil {
		return nil, err
	}

	// Prepend path with our consul path.
	path := os.Getenv("PATH")
	tb.Setenv("PATH", myConsulPath+":"+path)

	server, err := testutil.NewTestServerConfigT(tb, func(c *testutil.TestServerConfig) {
		c.EnableDebug = false
	})
	if err != nil {
		return nil, err
	}

	// Revert path.
	tb.Setenv("PATH", path)

	return server, nil
}

func TestSuite(t *testing.T) {
	s, cleanup, err := createServer()
	require.NoError(t, err, "while creating a server")

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkParallelGetService(b *testing.B) {
	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
