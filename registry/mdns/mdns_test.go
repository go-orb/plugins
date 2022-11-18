package mdns

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/types"

	_ "github.com/go-micro/plugins/log/text"
)

var (
	testData = []*registry.Service{
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

func TestMDNS(t *testing.T) {
	l, err := log.New(log.NewConfig())
	require.NoError(t, err, "failed to create logger")

	cfg, err := NewConfig(types.ServiceName("test.service"), nil)
	require.NoError(t, err, "failed to create registry config")

	r := New(cfg, l)
	require.NoError(t, r.Start(), "failed to start")

	for _, service := range testData {
		require.NoError(t, r.Register(service), "failed to register service")

		// Assure service has been registered properly.
		var s []*registry.Service
		s, err = r.GetService(service.Name)
		require.NoError(t, err, "failed fetch services")
		require.Equal(t, len(s), 1, "registry should only contain one registered service")
		require.Equal(t, s[0].Name, service.Name, "service name does not match")
		require.Equal(t, s[0].Version, service.Version, "service version does not match")
		require.Equal(t, len(s[0].Nodes), 1, "service should only contain one node")

		node := s[0].Nodes[0]
		require.Equal(t, node.ID, service.Nodes[0].ID, "node ID does not match")
		require.Equal(t, node.Address, service.Nodes[0].Address, "node address does not match")
	}

	services, err := r.ListServices()
	require.NoError(t, err, "failed to list services")

	for _, service := range testData {
		var seen bool
		for _, s := range services {
			if s.Name == service.Name {
				seen = true
				break
			}
		}

		// Assure service is present in registry.
		require.Equal(t, seen, true,
			"service not found in listed services, it has not been registered properly: "+service.Name)

		// Deregister and give the registry time to process.
		require.NoError(t, r.Deregister(service), "failed to deregister service: "+service.Name)
		time.Sleep(time.Millisecond * 100)

		// Assure service has deregistered properly.
		s, _ := r.GetService(service.Name) //nolint:errcheck
		require.GreaterOrEqual(t, 0, len(s), "service count should be 0, as all services should be deregistered")
	}
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
	testFn := func(service, s *registry.Service) {
		require.NotEqual(t, s, nil, "expected result, got nil: "+service.Name)
		require.Equal(t, s.Name, service.Name, "service name not equal")
		require.Equal(t, s.Version, service.Version, "service version not equal")
		require.Equal(t, len(s.Nodes), 1, "expected only 1 node")

		node := s.Nodes[0]
		require.Equal(t, node.ID, service.Nodes[0].ID, "node IDs not equal")
		require.Equal(t, node.Address, service.Nodes[0].Address, "node addresses not equal")
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
