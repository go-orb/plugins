package consul

import (
	"strings"
	"sync"

	"github.com/go-orb/go-orb/registry"
	maddr "github.com/go-orb/go-orb/util/addr"
	"github.com/go-orb/plugins/registry/regutil"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
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
	services map[string][]*registry.Service
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
		services: make(map[string][]*registry.Service),
	}

	// If a specific service is provided, watch that service
	if len(wo.Service) > 0 {
		wp, err := watch.Parse(map[string]interface{}{
			"service": wo.Service,
			"type":    "service",
		})
		if err != nil {
			return nil, err
		}

		wp.Handler = watcher.handle

		tmp := func() {
			_ = wp.RunWithClientAndHclog(regConsul.Client(), wp.Logger) //nolint:errcheck
		}
		go tmp()

		watcher.wp = wp
	} else {
		// If no service name is specified, watch all services
		wp, err := watch.Parse(map[string]interface{}{
			"type": "services",
		})
		if err != nil {
			return nil, err
		}

		wp.Handler = watcher.handle

		tmp := func() {
			_ = wp.RunWithClientAndHclog(regConsul.Client(), wp.Logger) //nolint:errcheck
		}
		go tmp()

		watcher.wp = wp
	}

	return watcher, nil
}

//nolint:funlen,gocognit,gocyclo,cyclop
func (cw *consulWatcher) serviceHandler(_ uint64, data interface{}) {
	entries, ok := data.([]*api.ServiceEntry)
	if !ok {
		return
	}

	var (
		haveService bool
		svc         *registry.Service
	)

	serviceMap := make(map[string]*registry.Service)
	serviceName := ""

	for _, node := range entries {
		// version is now a tag
		version, _ := decodeVersion(node.Service.Tags)

		nodeMeta := map[string]string{}
		svcMeta := map[string]string{}

		for k, v := range node.Service.Meta {
			if strings.HasPrefix(k, "orb_service_") {
				svcMeta[strings.TrimPrefix(k, "orb_service_")] = v
			} else if strings.HasPrefix(k, "orb_node_") {
				nodeMeta[strings.TrimPrefix(k, "orb_node_")] = v
			}
		}

		// Skip unknown services.
		if len(nodeMeta) == 0 && len(svcMeta) == 0 {
			continue
		}

		svc, haveService = serviceMap[version]
		if !haveService {
			serviceName = node.Service.Service
			svc = &registry.Service{
				Endpoints: decodeEndpoints(node.Service.Tags),
				Name:      node.Service.Service,
				Version:   svcMeta["version"],
				Metadata:  svcMeta,
			}
		}

		var del bool

		for _, check := range node.Checks {
			// delete the node if the status is critical
			if check.Status == "critical" {
				del = true
				break
			}
		}

		// if delete then skip the node
		if del {
			continue
		}

		rNode := &registry.Node{
			ID:       node.Node.ID,
			Address:  maddr.HostPort(node.Node.Address, node.Service.Port),
			Metadata: nodeMeta,
		}

		// Extract the transport from Metadata
		if transport, ok := node.Service.Meta[metaTransportKey]; ok {
			rNode.Transport = transport
			delete(rNode.Metadata, metaTransportKey)
		} else {
			continue
		}

		// Extract the node ID from Metadata
		if nodeID, ok := node.Service.Meta[metaNodeIDKey]; ok {
			rNode.ID = nodeID
			delete(rNode.Metadata, metaNodeIDKey)
		} else {
			continue
		}

		serviceMap[version] = svc
		svc.Nodes = append(svc.Nodes, rNode)
	}

	cw.RLock()
	// make a copy
	rservices := make(map[string][]*registry.Service)
	for k, v := range cw.services {
		rservices[k] = v
	}
	cw.RUnlock()

	var newServices []*registry.Service //nolint:prealloc

	// serviceMap is the new set of services keyed by name+version
	for _, newService := range serviceMap {
		// append to the new set of cached services
		newServices = append(newServices, newService)

		// check if the service exists in the existing cache
		oldServices, ok := rservices[serviceName]
		if !ok {
			// does not exist? then we're creating brand new entries
			cw.next <- &registry.Result{Action: "create", Service: newService}
			continue
		}

		// service exists. ok let's figure out what to update and delete version wise
		action := "create"

		for _, oldService := range oldServices {
			// does this version exist?
			// no? then default to create
			if oldService.Version != newService.Version {
				continue
			}

			// yes? then it's an update
			action = "update"

			var nodes []*registry.Node
			// check the old nodes to see if they've been deleted
			for _, oldNode := range oldService.Nodes {
				var seen bool

				for _, newNode := range newService.Nodes {
					if newNode.ID == oldNode.ID {
						seen = true
						break
					}
				}
				// does the old node exist in the new set of nodes
				// no? then delete that shit
				if !seen {
					nodes = append(nodes, oldNode)
				}
			}

			// it's an update rather than creation
			if len(nodes) > 0 {
				delService := regutil.CopyService(oldService)
				delService.Nodes = nodes
				cw.next <- &registry.Result{Action: "delete", Service: delService}
			}
		}

		cw.next <- &registry.Result{Action: action, Service: newService}
	}

	// Now check old versions that may not be in new services map
	for _, old := range rservices[serviceName] {
		// old version does not exist in new version map
		// kill it with fire!
		if _, ok := serviceMap[old.Version]; !ok {
			cw.next <- &registry.Result{Action: "delete", Service: old}
		}
	}

	cw.Lock()
	cw.services[serviceName] = newServices
	cw.Unlock()
}

func (cw *consulWatcher) handle(_ uint64, data interface{}) {
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

		wp, err := watch.Parse(map[string]interface{}{
			"type":    "service",
			"service": service,
		})
		if err == nil {
			wp.Handler = cw.serviceHandler

			tmp := func() {
				_ = wp.RunWithClientAndHclog(cw.r.Client(), wp.Logger) //nolint:errcheck
			}
			go tmp()

			cw.watchers[service] = wp
		}
	}

	cw.RLock()
	// make a copy
	rservices := make(map[string][]*registry.Service)
	for k, v := range cw.services {
		rservices[k] = v
	}
	cw.RUnlock()

	// remove unknown services from registry
	// save the things we want to delete
	deleted := make(map[string][]*registry.Service)

	for service := range rservices {
		if _, ok := services[service]; !ok {
			cw.Lock()
			// save this before deleting
			deleted[service] = cw.services[service]
			delete(cw.services, service)
			cw.Unlock()
		}
	}

	// remove unknown services from watchers
	for service, w := range cw.watchers {
		if _, ok := services[service]; !ok {
			w.Stop()
			delete(cw.watchers, service)

			for _, oldService := range deleted[service] {
				// send a delete for the service nodes that we're removing
				cw.next <- &registry.Result{Action: "delete", Service: oldService}
			}
			// sent the empty list as the last resort to indicate to delete the entire service
			cw.next <- &registry.Result{Action: "delete", Service: &registry.Service{Name: service}}
		}
	}
}

func (cw *consulWatcher) Next() (*registry.Result, error) {
	select {
	case <-cw.exit:
		return nil, registry.ErrWatcherStopped
	case r, ok := <-cw.next:
		if !ok {
			return nil, registry.ErrWatcherStopped
		}

		return r, nil
	}
}

func (cw *consulWatcher) Stop() error {
	select {
	case <-cw.exit:
		return nil
	default:
		close(cw.exit)

		if cw.wp == nil {
			return nil
		}

		cw.wp.Stop()

		// drain results
		for {
			select {
			case <-cw.next:
			default:
				return nil
			}
		}
	}
}
