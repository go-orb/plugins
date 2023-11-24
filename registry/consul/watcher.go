package consul

import (
	"net"
	"strconv"
	"sync"

	"github.com/go-orb/go-orb/registry"
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

func newConsulWatcher(cr *RegistryConsul, opts ...registry.WatchOption) (*consulWatcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	cw := &consulWatcher{
		r:        cr,
		wo:       wo,
		exit:     make(chan bool),
		next:     make(chan *registry.Result, 10),
		watchers: make(map[string]*watch.Plan),
		services: make(map[string][]*registry.Service),
	}

	wp, err := watch.Parse(map[string]interface{}{
		"service": wo.Service,
		"type":    "service",
	})
	if err != nil {
		return nil, err
	}

	wp.Handler = cw.handle

	tmp := func() {
		_ = wp.RunWithClientAndHclog(cr.Client(), wp.Logger) //nolint:errcheck
	}
	go tmp()

	cw.wp = wp

	return cw, nil
}

func (cw *consulWatcher) serviceHandler(_ uint64, data interface{}) { //nolint:funlen,gocognit,gocyclo
	entries, ok := data.([]*api.ServiceEntry)
	if !ok {
		return
	}

	serviceMap := map[string]*registry.Service{}
	serviceName := ""

	for _, entry := range entries {
		serviceName = entry.Service.Service
		// version is now a tag
		version, _ := decodeVersion(entry.Service.Tags)
		// service ID is now the node id
		id := entry.Service.ID
		// key is always the version
		key := version
		// address is service address
		address := entry.Service.Address

		// use node address
		if len(address) == 0 {
			address = entry.Node.Address
		}

		svc, ok := serviceMap[key]
		if !ok {
			svc = &registry.Service{
				Endpoints: decodeEndpoints(entry.Service.Tags),
				Name:      entry.Service.Service,
				Version:   version,
			}
			serviceMap[key] = svc
		}

		var del bool

		for _, check := range entry.Checks {
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

		svc.Nodes = append(svc.Nodes, &registry.Node{
			ID:       id,
			Address:  net.JoinHostPort(address, strconv.Itoa(entry.Service.Port)),
			Metadata: entry.Service.Meta,
		})
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
			cw.next <- &registry.Result{Action: "create", Service: &registry.Service{Name: service}}
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
