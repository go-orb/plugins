package mdns

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/types"

	_ "github.com/go-micro/plugins/log/text"
	"github.com/go-micro/plugins/registry/tests"
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
					ID:      "test1-1",
					Address: "10.0.0.1:10001",
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
					ID:      "test2-1",
					Address: "10.0.0.2:10002",
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
					ID:      "test3-1",
					Address: "10.0.0.3:10003",
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
					ID:      "test4-1",
					Address: "[::]:10004",
					Metadata: map[string]string{
						"foo4": "bar4",
					},
				},
			},
		},
	}
)

func TestMain(m *testing.M) {
	logger, err := log.New(log.NewConfig())
	if err != nil {
		log.Error("failed to create logger", err)
		os.Exit(1)
	}

	cfg, err := NewConfig(types.ServiceName("test.service"), nil)
	if err != nil {
		logger.Error("failed to create registry config", err)
		os.Exit(1)
	}

	r := New(cfg, logger)
	if err := r.Start(); err != nil {
		logger.Error("failed to start", err)
		os.Exit(1)
	}

	tests.CreateSuite(logger, []registry.Registry{r}, 0, 0)
	tests.Suite.Setup()

	result := m.Run()

	tests.Suite.TearDown()

	if err := r.Stop(context.Background()); err != nil {
		logger.Error("failed to stop", err)
		os.Exit(1)
	}

	os.Exit(result)
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
		require.NotEqual(t, actual, nil, "expected result, got nil: "+expected.Name)
		require.Equal(t, expected.Name, actual.Name, "service name not equal")
		require.Equal(t, expected.Version, actual.Version, "service version not equal")
		require.Equal(t, 1, len(actual.Nodes), "expected only 1 node")

		expectedNode := expected.Nodes[0]
		actualNode := actual.Nodes[0]
		require.Equal(t, expectedNode.ID, actualNode.ID, "node IDs not equal")
		require.Equal(t, expectedNode.Address, actualNode.Address, "node addresses not equal")
		require.Equal(t, expectedNode.Scheme, actualNode.Scheme, "node scheme does not equal")
	}

	// New registry
	l, err := log.New(log.NewConfig())
	require.NoError(t, err, "failed to create logger")

	cfg, err := NewConfig(types.ServiceName("test.service"), nil)
	require.NoError(t, err, "failed to create registry config")

	r := New(cfg, l)
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

			require.Equal(t, res.Action, "create", "expected create event")

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

// TODO(rene): These tests fail, I think because there's no remote MDNS, we need to fix this!
func TestRegister(t *testing.T) {
	tests.Suite.TestRegister(t)
}

func TestDeregister(t *testing.T) {
	tests.Suite.TestDeregister(t)
}

func TestGetServiceAllRegistries(t *testing.T) {
	tests.Suite.TestGetServiceAllRegistries(t)
}

func TestGetServiceWithNoNodes(t *testing.T) {
	tests.Suite.TestGetServiceWithNoNodes(t)
}

func BenchmarkGetService(b *testing.B) {
	tests.Suite.BenchmarkGetService(b)
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	tests.Suite.BenchmarkGetServiceWithNoNodes(b)
}
