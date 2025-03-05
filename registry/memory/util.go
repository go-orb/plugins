package memory

import (
	"maps"
	"slices"
	"time"

	"github.com/go-orb/go-orb/registry"
)

func serviceToRecord(s *registry.Service, ttl time.Duration) *record {
	nodes := make(map[string]*node, len(s.Nodes))
	for _, n := range s.Nodes {
		nodes[n.ID] = &node{
			Node:     n,
			TTL:      ttl,
			LastSeen: time.Now(),
		}
	}

	return &record{
		Name:      s.Name,
		Version:   s.Version,
		Metadata:  maps.Clone(s.Metadata),
		Nodes:     nodes,
		Endpoints: slices.Clone(s.Endpoints),
	}
}

func recordToService(inRecord *record) *registry.Service {
	endpoints := make([]*registry.Endpoint, len(inRecord.Endpoints))

	for i, e := range inRecord.Endpoints {
		request := new(registry.Value)
		if e.Request != nil {
			*request = *e.Request
		}

		response := new(registry.Value)
		if e.Response != nil {
			*response = *e.Response
		}

		endpoints[i] = &registry.Endpoint{
			Name:     e.Name,
			Request:  request,
			Response: response,
			Metadata: maps.Clone(e.Metadata),
		}
	}

	nodes := make([]*registry.Node, len(inRecord.Nodes))
	i := 0

	for _, n := range inRecord.Nodes {
		nodes[i] = &registry.Node{
			ID:        n.ID,
			Address:   n.Address,
			Transport: n.Transport,
			Metadata:  maps.Clone(n.Metadata),
		}
		i++
	}

	return &registry.Service{
		Name:      inRecord.Name,
		Version:   inRecord.Version,
		Metadata:  maps.Clone(inRecord.Metadata),
		Endpoints: endpoints,
		Nodes:     nodes,
	}
}
