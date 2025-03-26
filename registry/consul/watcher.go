package consul

import (
	"maps"
	"strings"
	"sync"

	"github.com/go-orb/go-orb/registry"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/hashicorp/go-hclog"
)

var _ registry.Watcher = (*consulWatcher)(nil)

type consulWatcher struct {
	r        *RegistryConsul
	wo       registry.WatchOptions
	wp       *watch.Plan
	watchers map[string]*watch.Plan

	next chan *registry.Result
	exit chan bool

	sync.RWMutex
	nodes map[string]registry.ServiceNode
}

func newConsulWatcher(regConsul *RegistryConsul, opts ...registry.WatchOption) (*consulWatcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	watcher := &consulWatcher{
		r:        regConsul,
		wo:       wo,
		exit:     make(chan bool),
		next:     make(chan *registry.Result, 10),
		watchers: make(map[string]*watch.Plan),
		nodes:    make(map[string]registry.ServiceNode),
	}

	// If a specific service is provided, watch that service
	if len(wo.Service) > 0 {
		wp, err := watch.Parse(map[string]any{
			"service": wo.Service,
			"type":    "service",
		})
		if err != nil {
			return nil, err
		}

		wp.Logger = hclog.New(&hclog.LoggerOptions{Level: hclog.Off})
		wp.Handler = watcher.handle

		go func() {
			_ = wp.RunWithClientAndHclog(regConsul.Client(), wp.Logger) //nolint:errcheck
		}()

		watcher.wp = wp
	} else {
		// If no service name is specified, watch all services
		wp, err := watch.Parse(map[string]any{
			"type": "services",
		})
		if err != nil {
			return nil, err
		}

		wp.Logger = hclog.New(&hclog.LoggerOptions{Level: hclog.Off})
		wp.Handler = watcher.handle

		go func() {
			_ = wp.RunWithClientAndHclog(regConsul.Client(), wp.Logger) //nolint:errcheck
		}()

		watcher.wp = wp
	}

	return watcher, nil
}

func (cw *consulWatcher) serviceHandler(_ uint64, data any) {
	entries, entriesOk := data.([]*api.ServiceEntry)
	if !entriesOk {
		return
	}

	serviceNodes := make(map[string]registry.ServiceNode, len(entries))
	servicePrefix := ""

	for _, node := range entries {
		svcMeta := map[string]string{}

		for k, v := range node.Service.Meta {
			if strings.HasPrefix(k, metaPrefix) {
				svcMeta[strings.TrimPrefix(k, metaPrefix)] = v
			}
		}

		// Try to parse the node ID to get the service node
		serviceNode, err := idToNode(node.Service.ID)
		if err != nil {
			continue
		}

		serviceNode.Node = node.Service.Meta[myMetaPrefix+"node"]
		serviceNode.Network = node.Service.Meta[myMetaPrefix+"network"]
		serviceNode.Scheme = node.Service.Meta[myMetaPrefix+"scheme"]
		serviceNode.Address = node.Service.Meta[myMetaPrefix+"address"]

		// Update with the latest metadata
		serviceNode.Metadata = svcMeta

		if servicePrefix == "" {
			servicePrefix = idToPrefix(node.Service.ID)
		}

		serviceNodes[node.Service.ID] = serviceNode
	}

	// Get a copy of the current services
	cw.RLock()
	currentNodes := make(map[string]registry.ServiceNode)
	maps.Copy(currentNodes, cw.nodes)
	cw.RUnlock()

	// Check for new nodes.
	for id, newNode := range serviceNodes {
		oldNode, entriesOk := currentNodes[id]
		if !entriesOk {
			cw.next <- &registry.Result{Action: registry.Create, Node: newNode}

			// Update the current nodes
			cw.Lock()
			cw.nodes[id] = newNode
			cw.Unlock()
		} else if !metadataEqual(oldNode.Metadata, newNode.Metadata) {
			// Check for updates
			cw.next <- &registry.Result{Action: registry.Update, Node: newNode}

			// Update the current nodes
			cw.Lock()
			cw.nodes[id] = newNode
			cw.Unlock()
		}
	}

	if servicePrefix == "" {
		return
	}

	oldNodes := make(map[string]registry.ServiceNode)

	// Check for deleted nodes
	for k, v := range currentNodes {
		if strings.HasPrefix(k, servicePrefix) {
			oldNodes[k] = v
		}
	}

	for id, oldNode := range oldNodes {
		if _, exists := serviceNodes[id]; !exists {
			cw.next <- &registry.Result{Action: registry.Delete, Node: oldNode}

			cw.Lock()
			delete(cw.nodes, id)
			cw.Unlock()
		}
	}
}

// Helper function to compare metadata.
func metadataEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}

	return true
}

func (cw *consulWatcher) handle(_ uint64, data any) {
	services, ok := data.(map[string][]string)
	if !ok {
		return
	}

	// add new watchers
	for service := range services {
		// Filter on watch options
		// wo.Service: Only watch services we care about
		if len(cw.wo.Service) > 0 && service != cw.wo.Service {
			continue
		}

		if _, ok := cw.watchers[service]; ok {
			continue
		}

		wp, err := watch.Parse(map[string]any{
			"type":    "service",
			"service": service,
		})
		if err == nil {
			wp.Logger = hclog.New(&hclog.LoggerOptions{Level: hclog.Off})
			wp.Handler = cw.serviceHandler

			go func() {
				_ = wp.RunWithClientAndHclog(cw.r.Client(), wp.Logger) //nolint:errcheck
			}()

			cw.watchers[service] = wp

			continue
		}
	}

	// remove unknown services from registry
	for k := range cw.watchers {
		if _, ok := services[k]; !ok {
			delete(cw.watchers, k)

			cw.Lock()
			for id, node := range cw.nodes {
				if _, _, name := splitID(id); name == k {
					cw.next <- &registry.Result{Action: registry.Delete, Node: node}
					delete(cw.nodes, id)
				}
			}

			cw.Unlock()
		}
	}
}

func (cw *consulWatcher) Next() (*registry.Result, error) {
	select {
	case <-cw.exit:
		return nil, registry.ErrWatcherStopped
	case r := <-cw.next:
		return r, nil
	}
}

func (cw *consulWatcher) Stop() error {
	select {
	case <-cw.exit:
		return nil
	default:
		// stop the watchers
		for _, wp := range cw.watchers {
			wp.Stop()
		}

		if cw.wp != nil {
			cw.wp.Stop()
		}

		close(cw.exit)
	}

	return nil
}
