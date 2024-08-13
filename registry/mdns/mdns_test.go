package mdns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

var (
	testDataEncoding = []*mdnsTxt{
		{
			Version: "1.0.0",
			Metadata: map[string]string{
				"foo": "bar",
			},
			Endpoints: []*registry.Endpoint{
				{
					Name: "endpoint1",
					Request: &registry.Value{
						Name: "request",
						Type: "request",
					},
					Response: &registry.Value{
						Name: "response",
						Type: "response",
					},
					Metadata: map[string]string{
						"foo1": "bar1",
					},
				},
			},
		},
	}

	testDataWatcher = []*registry.Service{
		{
			Name:    "test1",
			Version: "1.0.1",
			Nodes: []*registry.Node{
				{
					ID:        "test1-1",
					Address:   "10.0.0.1:10001",
					Transport: "http",
					Metadata: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
		{
			Name:    "test2",
			Version: "1.0.2",
			Nodes: []*registry.Node{
				{
					ID:        "test2-1",
					Address:   "10.0.0.2:10002",
					Transport: "grpc",
					Metadata: map[string]string{
						"foo2": "bar2",
					},
				},
			},
		},
		{
			Name:    "test3",
			Version: "1.0.3",
			Nodes: []*registry.Node{
				{
					ID:        "test3-1",
					Address:   "10.0.0.3:10003",
					Transport: "frpc",
					Metadata: map[string]string{
						"foo3": "bar3",
					},
				},
			},
		},
		{
			Name:    "test4",
			Version: "1.0.4",
			Nodes: []*registry.Node{
				{
					ID:        "test4-1",
					Address:   "[::]:10004",
					Transport: "drpc",
					Metadata: map[string]string{
						"foo4": "bar4",
					},
				},
			},
		},
	}
)

func createServer() (*tests.TestSuite, func() error, error) {
	logger, err := log.New()
	if err != nil {
		log.Error("failed to create logger", err)
		return nil, func() error { return nil }, err
	}

	cfg, err := NewConfig("test.service", nil, WithDomain("mdns.test.local"))
	r := New("", "", cfg, logger)
	if err != nil {
		logger.Error("failed to create registry config", "err", err)
		return nil, func() error { return nil }, err
	}

	cleanup := func() error {
		return r.Stop(context.Background())
	}

	return tests.CreateSuite(logger, []registry.Registry{r}, 0, 0), cleanup, nil
}

func TestEncoding(t *testing.T) {
	for _, d := range testDataEncoding {
		encoded, err := encode(d)
		require.NoError(t, err, "failed to encode")

		for _, txt := range encoded {
			require.GreaterOrEqual(t, 255, len(txt), "one of parts for txt is too long")
		}

		decoded, err := decode(encoded)
		require.NoError(t, err, "failed to decode")
		require.Equal(t, decoded.Version, d.Version)
		require.Equal(t, decoded.Endpoints, d.Endpoints)

		for k, v := range d.Metadata {
			require.Equal(t, decoded.Metadata[k], v)
		}
	}
}

func TestWatcher(t *testing.T) {
	testFn := func(expected, actual *registry.Service) {
		require.NotNil(t, actual, "expected result, got nil: "+expected.Name)
		require.Equal(t, expected.Name, actual.Name, "service name not equal")
		require.Equal(t, expected.Version, actual.Version, "service version not equal")
		require.Len(t, actual.Nodes, 1, "expected only 1 node")

		node := expected.Nodes[0]
		require.Equal(t, node.ID, actual.Nodes[0].ID, "node IDs not equal")
		require.Equal(t, node.Address, actual.Nodes[0].Address, "node addresses not equal")
		require.Equal(t, node.Transport, actual.Nodes[0].Transport, "node Transport does not equal")
	}

	// New registry
	l, err := log.New()
	require.NoError(t, err, "failed to create logger")

	cfg, err := NewConfig(types.ServiceName("test.service"), nil)
	require.NoError(t, err, "failed to create registry config")

	r := New("", "", cfg, l)
	require.NoError(t, r.Start(), "failed to start service")

	w, err := r.Watch()
	require.NoError(t, err, "failed to start registry watcher")
	defer func() {
		w.Stop() //nolint:errcheck
	}()

	for _, service := range testDataWatcher {
		require.NoError(t, r.Register(service), "failed to register service")

		for {
			res, err := w.Next()
			require.NoError(t, err, "failed to fetch next")

			if res.Service.Name != service.Name {
				continue
			}

			require.Equal(t, "create", res.Action, "expected create event")

			testFn(service, res.Service)
			break
		}

		require.NoError(t, r.Deregister(service), "failed to deregister service: "+service.Name)

		for {
			res, err := w.Next()
			require.NoError(t, err, "failed to fetch next")

			if res.Service.Name != service.Name {
				continue
			}

			if res.Action != "delete" {
				continue
			}

			testFn(service, res.Service)
			break
		}
	}
}

func TestSuite(t *testing.T) {
	s, cleanup, err := createServer()
	require.NoError(t, err, "while creating a server")

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	b.StopTimer()

	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	b.StopTimer()

	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
