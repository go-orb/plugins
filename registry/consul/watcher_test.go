package consul

import (
	"testing"

	"github.com/go-orb/go-orb/registry"
	"github.com/hashicorp/consul/api"
)

func TestHealthyServiceHandler(t *testing.T) {
	watcher := newWatcher()
	serviceEntry := newServiceEntry(
		"node-name", "node-address", "service-name", "v1.0.0", "http",
		[]*api.HealthCheck{
			newHealthCheck("node-name", "service-name", "passing"),
		},
	)

	watcher.serviceHandler(1234, []*api.ServiceEntry{serviceEntry})

	if len(watcher.services["service-name"][0].Nodes) != 1 {
		t.Errorf("Expected length of the service nodes to be 1")
	}
}

func TestUnhealthyServiceHandler(t *testing.T) {
	watcher := newWatcher()
	serviceEntry := newServiceEntry(
		"node-name", "node-address", "service-name", "v1.0.0", "grpc",
		[]*api.HealthCheck{
			newHealthCheck("node-name", "service-name", "critical"),
		},
	)

	watcher.serviceHandler(1234, []*api.ServiceEntry{serviceEntry})

	if len(watcher.services["service-name"][0].Nodes) != 0 {
		t.Errorf("Expected length of the service nodes to be 0")
	}
}

func TestUnhealthyNodeServiceHandler(t *testing.T) {
	watcher := newWatcher()
	serviceEntry := newServiceEntry(
		"node-name", "node-address", "service-name", "v1.0.0", "frpc",
		[]*api.HealthCheck{
			newHealthCheck("node-name", "service-name", "passing"),
			newHealthCheck("node-name", "serfHealth", "critical"),
		},
	)

	watcher.serviceHandler(1234, []*api.ServiceEntry{serviceEntry})

	if len(watcher.services["service-name"][0].Nodes) != 0 {
		t.Errorf("Expected length of the service nodes to be 0")
	}
}

func newWatcher() *consulWatcher {
	return &consulWatcher{
		exit:     make(chan bool),
		next:     make(chan *registry.Result, 10),
		services: make(map[string][]*registry.Service),
	}
}

func newHealthCheck(node, name, status string) *api.HealthCheck {
	return &api.HealthCheck{
		Node:        node,
		Name:        name,
		Status:      status,
		ServiceName: name,
	}
}

func newServiceEntry(node, address, name, version, scheme string, checks []*api.HealthCheck) *api.ServiceEntry {
	md := map[string]string{metaSchemeKey: scheme}
	return &api.ServiceEntry{
		Node: &api.Node{Node: node, Address: address, Meta: md},
		Service: &api.AgentService{
			Service: name,
			Address: address,
			Tags:    encodeVersion(version),
		},
		Checks: checks,
	}
}
