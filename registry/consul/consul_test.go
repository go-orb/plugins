package consul

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

func createRegistry(suite *tests.TestSuite) (registry.Registry, error) {
	t := suite.T()
	t.Helper()

	addr, ok := suite.Server.(string)
	if !ok {
		return nil, errors.New("while retrieving the store")
	}

	cfg := NewConfig(WithAddress(addr))
	reg := New(cfg, suite.Logger.With("reg", "custom"))
	err := reg.Start(suite.Ctx)
	require.NoError(t, err, "while starting a registry")

	return reg, nil
}

func createSuite(tb testing.TB) (*tests.TestSuite, func() error) {
	tb.Helper()

	ctx := context.Background()

	logger, err := log.New(log.WithLevel("TRACE"))
	require.NoError(tb, err, "while creating a logger")

	server, err := createServer(&testing.T{})
	require.NoError(tb, err, "while creating a server")

	reg1 := New(NewConfig(WithAddress(server.HTTPAddr), WithNoCache()), logger)
	require.NoError(tb, reg1.Start(ctx), "while starting registry one")

	reg2 := New(NewConfig(WithAddress(server.HTTPAddr)), logger)
	require.NoError(tb, reg2.Start(ctx), "while starting registry two")

	cleanup := func() error {
		_ = server.Stop() //nolint:errcheck
		return nil
	}

	r := &tests.TestSuite{
		Server:         server.HTTPAddr,
		Ctx:            context.Background(),
		Logger:         logger,
		Registries:     []registry.Registry{reg1, reg2},
		UpdateTime:     time.Second,
		CreateRegistry: createRegistry,
	}

	return r, cleanup
}

func createServer(tb testing.TB) (*testutil.TestServer, error) {
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
