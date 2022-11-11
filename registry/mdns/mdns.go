// Package mdns provides a multicast dns registry
package mdns

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-micro/plugins/registry/mdns/client"
	"github.com/go-micro/plugins/registry/mdns/dns"
	"github.com/go-micro/plugins/registry/mdns/server"
	"github.com/go-micro/plugins/registry/mdns/zone"

	"github.com/google/uuid"
	"go-micro.dev/v5/config/source"
	"go-micro.dev/v5/log"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/types"
	"go-micro.dev/v5/types/component"
)

type mdnsTxt struct {
	Service   string
	Version   string
	Endpoints []*registry.Endpoint
	Metadata  map[string]string
}

type mdnsEntry struct {
	id   string
	node *server.Server
}

// This is here to make sure mdnsRegistry implements registry.Registry.
var _ registry.Registry = (*RegistryMDNS)(nil)

// RegistryMDNS implements the registry interface. It runs a MDNS service registry.
type RegistryMDNS struct {
	config Config
	logger log.Logger

	sync.Mutex
	services map[string][]*mdnsEntry

	mtx sync.RWMutex

	// watchers
	watchers map[string]*mdnsWatcher

	// listener
	listener chan *client.ServiceEntry
}

type mdnsWatcher struct {
	id   string
	wo   registry.WatchOptions
	ch   chan *client.ServiceEntry
	exit chan struct{}
	// the mdns domain
	domain string
	// the registry
	registry *RegistryMDNS
}

// ProvideRegistryMDNS creates a new MDNS registry.
func ProvideRegistryMDNS(
	name types.ServiceName,
	data []source.Data,
	logger log.Logger,
	opts ...registry.Option,
) (*registry.MicroRegistry, error) {
	cfg, err := NewConfig(name, data, opts...)
	if err != nil {
		return nil, fmt.Errorf("create mdns registry config: %w", err)
	}

	logger, err = logger.WithComponent(registry.ComponentType, "mdns", "", nil)
	if err != nil {
		return nil, err
	}

	cfg.Logger = logger

	// Return the new registry.
	reg := New(cfg, logger)

	return &registry.MicroRegistry{Registry: reg}, nil
}

// New creates a new mdns registry. This functions should rarely be called manually.
// To create a new registry use ProvideRegistryMDNS.
func New(cfg Config, log log.Logger) *RegistryMDNS {
	if cfg.Timeout == 0 {
		cfg.Timeout = registry.DefaultTimeout
	}

	if cfg.Domain == "" {
		cfg.Domain = DefaultDomain
	}

	return &RegistryMDNS{
		config:   cfg,
		logger:   log,
		services: make(map[string][]*mdnsEntry),
		watchers: make(map[string]*mdnsWatcher),
	}
}

// Start the registry.
func (m *RegistryMDNS) Start() error {
	// TODO: I think this should start something?
	return nil
}

// Stop the registry.
func (m *RegistryMDNS) Stop() error {
	// TODO: do something here?
	return nil
}

// String returns the plugin name.
func (m *RegistryMDNS) String() string {
	return name
}

// Type returns the component type.
func (m *RegistryMDNS) Type() component.Type {
	return registry.ComponentType
}

// Register registes a service's nodes to the registry.
func (m *RegistryMDNS) Register(service *registry.Service, opts ...registry.RegisterOption) error {
	m.Lock()
	defer m.Unlock()

	entries, ok := m.services[service.Name]
	// first entry, create wildcard used for list queries
	if !ok {
		s, err := zone.NewMDNSService(
			service.Name,
			"_services",
			m.config.Domain+".",
			"",
			9999,
			[]net.IP{net.ParseIP("0.0.0.0")},
			nil,
		)
		if err != nil {
			return err
		}

		srv, err := server.NewServer(&server.Config{Zone: &dns.SDService{MDNSService: s}})
		if err != nil {
			return err
		}

		// append the wildcard entry
		entries = append(entries, &mdnsEntry{id: "*", node: srv})
	}

	var gerr error

	for _, node := range service.Nodes {
		var (
			seen  bool
			entry *mdnsEntry
		)

		for _, entry := range entries {
			if node.ID == entry.id {
				seen = true
				break
			}
		}

		// Already registered, continue
		if seen {
			continue
		} else {
			// Doesn't exist
			entry = &mdnsEntry{}
		}

		txt, err := encode(&mdnsTxt{
			Service:   service.Name,
			Version:   service.Version,
			Endpoints: service.Endpoints,
			Metadata:  node.Metadata,
		})

		if err != nil {
			gerr = err
			continue
		}

		host, pt, err := net.SplitHostPort(node.Address)
		if err != nil {
			gerr = err
			continue
		}

		port, _ := strconv.Atoi(pt) //nolint:errcheck

		m.logger.Debug("[mdns] registry create new service with ip: %s for: %s", net.ParseIP(host).String(), host)

		// we got here, new node
		s, err := zone.NewMDNSService(
			node.ID,
			service.Name,
			m.config.Domain+".",
			"",
			port,
			[]net.IP{net.ParseIP(host)},
			txt,
		)
		if err != nil {
			gerr = err
			continue
		}

		srv, err := server.NewServer(&server.Config{Zone: s, LocalhostChecking: true})
		if err != nil {
			gerr = err
			continue
		}

		entry.id = node.ID
		entry.node = srv
		entries = append(entries, entry)
	}

	// save
	m.services[service.Name] = entries

	return gerr
}

// Deregister a service from the registry.
func (m *RegistryMDNS) Deregister(service *registry.Service, opts ...registry.DeregisterOption) error {
	m.Lock()
	defer m.Unlock()

	var newEntries []*mdnsEntry

	// loop existing entries, check if any match, shutdown those that do
	for _, entry := range m.services[service.Name] {
		var remove bool

		for _, node := range service.Nodes {
			if node.ID == entry.id {
				if err := entry.node.Shutdown(); err != nil {
					m.logger.Error("Failed to shutdown node", err)
				}

				remove = true

				break
			}
		}

		// keep it?
		if !remove {
			newEntries = append(newEntries, entry)
		}
	}

	// last entry is the wildcard for list queries. Remove it.
	if len(newEntries) == 1 && newEntries[0].id == "*" {
		if err := newEntries[0].node.Shutdown(); err != nil {
			m.logger.Error("failed to shutdown node", err)
		}

		delete(m.services, service.Name)
	} else {
		m.services[service.Name] = newEntries
	}

	return nil
}

// GetService fetches a service from the registry.
func (m *RegistryMDNS) GetService(service string, opts ...registry.GetOption) ([]*registry.Service, error) {
	serviceMap := make(map[string]*registry.Service)
	entries := make(chan *client.ServiceEntry, 10)
	done := make(chan bool)

	params := client.DefaultParams(service)

	// Set context with timeout
	var cancel context.CancelFunc

	params.Context, cancel = context.WithTimeout(context.Background(), time.Duration(m.config.Timeout)*time.Millisecond)
	defer cancel()

	params.Entries = entries
	params.Domain = m.config.Domain

	go m.getService(service, params, serviceMap, entries, done)

	// Execute the query
	if err := client.Query(params); err != nil {
		return nil, err
	}

	// Wait for completion
	<-done

	// Create list and return
	services := make([]*registry.Service, 0, len(serviceMap))

	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (m *RegistryMDNS) getService(
	service string,
	params *client.QueryParam,
	serviceMap map[string]*registry.Service,
	entries chan *client.ServiceEntry,
	done chan bool) {
	for {
		select {
		case entry := <-entries:
			// List record so skip
			if params.Service == "_services" {
				continue
			}

			if params.Domain != m.config.Domain {
				continue
			}

			if entry.TTL == 0 {
				continue
			}

			txt, err := decode(entry.InfoFields)
			if err != nil {
				continue
			}

			if txt.Service != service {
				continue
			}

			service, ok := serviceMap[txt.Version]
			if !ok {
				service = &registry.Service{
					Name:      txt.Service,
					Version:   txt.Version,
					Endpoints: txt.Endpoints,
				}
			}

			addr := ""

			switch {
			// Prefer IPv4 addrs
			case len(entry.AddrV4) > 0:
				addr = net.JoinHostPort(entry.AddrV4.String(), fmt.Sprint(entry.Port))
			// Else use IPv6
			case len(entry.AddrV6) > 0:
				addr = net.JoinHostPort(entry.AddrV6.String(), fmt.Sprint(entry.Port))
			default:
				m.logger.Info("[mdns]: invalid endpoint received: %v", entry)
				continue
			}

			service.Nodes = append(service.Nodes, &registry.Node{
				ID:       strings.TrimSuffix(entry.Name, "."+params.Service+"."+params.Domain+"."),
				Address:  addr,
				Metadata: txt.Metadata,
			})

			serviceMap[txt.Version] = service
		case <-params.Context.Done():
			close(done)
			return
		}
	}
}

// ListServices fetches all services in the registry.
func (m *RegistryMDNS) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	serviceMap := make(map[string]bool)
	entries := make(chan *client.ServiceEntry, 10)
	done := make(chan bool)

	params := client.DefaultParams("_services")

	// set context with timeout
	var cancel context.CancelFunc

	timeout := time.Duration(m.config.Timeout) * time.Millisecond

	params.Context, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	params.Entries = entries
	params.Domain = m.config.Domain

	var services []*registry.Service

	go func() {
		for {
			select {
			case e := <-entries:
				if e.TTL == 0 {
					continue
				}

				if !strings.HasSuffix(e.Name, params.Domain+".") {
					continue
				}

				name := strings.TrimSuffix(e.Name, "."+params.Service+"."+params.Domain+".")
				if !serviceMap[name] {
					serviceMap[name] = true

					services = append(services, &registry.Service{Name: name})
				}
			case <-params.Context.Done():
				close(done)
				return
			}
		}
	}()

	// execute query
	if err := client.Query(params); err != nil {
		return nil, err
	}

	// wait till done
	<-done

	return services, nil
}

// Watch for registration changes.
func (m *RegistryMDNS) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	watcher := &mdnsWatcher{
		id:       uuid.New().String(),
		wo:       wo,
		ch:       make(chan *client.ServiceEntry, 32),
		exit:     make(chan struct{}),
		domain:   m.config.Domain,
		registry: m,
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Save the watcher
	m.watchers[watcher.id] = watcher

	// Check of the listener exists
	if m.listener != nil {
		return watcher, nil
	}

	// Start the listener
	go m.watch()

	return watcher, nil
}

func (m *RegistryMDNS) watch() {
	// Go to infinity
	for {
		m.mtx.Lock()

		// Just return if there are no watchers
		if len(m.watchers) == 0 {
			m.listener = nil
			m.mtx.Unlock()

			return
		}

		// Check existing listener
		if m.listener != nil {
			m.mtx.Unlock()
			return
		}

		// Reset the listener
		exit := make(chan struct{})
		ch := make(chan *client.ServiceEntry, 32)
		m.listener = ch

		m.mtx.Unlock()

		// Send messages to the watchers
		go func() {
			send := func(w *mdnsWatcher, e *client.ServiceEntry) {
				select {
				case w.ch <- e:
				default:
				}
			}

			for {
				select {
				case <-exit:
					return
				case e, ok := <-ch:
					if !ok {
						return
					}

					// Send service entry to all watchers
					m.mtx.RLock()
					for _, w := range m.watchers {
						send(w, e)
					}
					m.mtx.RUnlock()
				}
			}
		}()

		// Start listening, blocking call
		if err := client.Listen(ch, exit); err != nil {
			m.logger.Error("Failed to listen", err)
		} else {
			m.logger.Info("Listening")
		}

		// mdns.Listen has unblocked
		// Kill the saved listener
		m.mtx.Lock()

		m.listener = nil

		close(ch)

		m.mtx.Unlock()
	}
}

func (m *mdnsWatcher) Next() (*registry.Result, error) {
	for {
		select {
		case entry := <-m.ch:
			txt, err := decode(entry.InfoFields)
			if err != nil {
				continue
			}

			if len(txt.Service) == 0 || len(txt.Version) == 0 {
				continue
			}

			// Filter watch options
			// wo.Service: Only keep services we care about
			if len(m.wo.Service) > 0 && txt.Service != m.wo.Service {
				continue
			}

			var action string

			if entry.TTL == 0 {
				action = "delete"
			} else {
				action = "create"
			}

			service := &registry.Service{
				Name:      txt.Service,
				Version:   txt.Version,
				Endpoints: txt.Endpoints,
			}

			// skip anything without the domain we care about
			suffix := fmt.Sprintf(".%s.%s.", service.Name, m.domain)
			if !strings.HasSuffix(entry.Name, suffix) {
				continue
			}

			var addr string

			switch {
			case len(entry.AddrV4) > 0:
				addr = net.JoinHostPort(entry.AddrV4.String(), fmt.Sprint(entry.Port))
			case len(entry.AddrV6) > 0:
				addr = net.JoinHostPort(entry.AddrV6.String(), fmt.Sprint(entry.Port))
			default:
				addr = entry.Addr.String()
			}

			service.Nodes = append(service.Nodes, &registry.Node{
				ID:       strings.TrimSuffix(entry.Name, suffix),
				Address:  addr,
				Metadata: txt.Metadata,
			})

			return &registry.Result{
				Action:  action,
				Service: service,
			}, nil
		case <-m.exit:
			return nil, registry.ErrWatcherStopped
		}
	}
}

func (m *mdnsWatcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
		// remove self from the registry
		m.registry.mtx.Lock()
		delete(m.registry.watchers, m.id)
		m.registry.mtx.Unlock()
	}
}

func encode(txt *mdnsTxt) ([]string, error) {
	b, err := json.Marshal(txt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	defer buf.Reset()

	w := zlib.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	encoded := hex.EncodeToString(buf.Bytes())

	// individual txt limit
	if len(encoded) <= 255 {
		return []string{encoded}, nil
	}

	// split encoded string
	var record []string

	for len(encoded) > 255 {
		record = append(record, encoded[:255])
		encoded = encoded[255:]
	}

	record = append(record, encoded)

	return record, nil
}

func decode(record []string) (*mdnsTxt, error) {
	encoded := strings.Join(record, "")

	hr, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	br := bytes.NewReader(hr)

	zr, err := zlib.NewReader(br)
	if err != nil {
		return nil, err
	}

	rbuf, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	var txt *mdnsTxt

	if err := json.Unmarshal(rbuf, &txt); err != nil {
		return nil, err
	}

	return txt, nil
}
