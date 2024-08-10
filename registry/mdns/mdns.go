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

	"log/slog"

	"github.com/go-orb/plugins/registry/mdns/client"
	"github.com/go-orb/plugins/registry/mdns/dns"
	"github.com/go-orb/plugins/registry/mdns/server"
	"github.com/go-orb/plugins/registry/mdns/zone"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/google/uuid"
)

type mdnsTxt struct {
	Service   string               `json:"service"`
	Version   string               `json:"version"`
	Endpoints []*registry.Endpoint `json:"endpoints"`
	Metadata  map[string]string    `json:"metadata"`
}

type mdnsEntry struct {
	id   string
	node *server.Server
}

// This is here to make sure RegistryMDNS implements registry.Registry.
var _ registry.Registry = (*RegistryMDNS)(nil)

// RegistryMDNS implements the registry interface. It runs a MDNS service registry.
type RegistryMDNS struct {
	serviceName    string
	serviceVersion string

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
	version types.ServiceVersion,
	datas types.ConfigData,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg, err := NewConfig(name, datas, opts...)
	if err != nil {
		return registry.Type{}, fmt.Errorf("create mdns registry config: %w", err)
	}

	// Return the new registry.
	reg := New(string(name), string(version), cfg, logger)

	return registry.Type{Registry: reg}, nil
}

// New creates a new mdns registry. This functions should rarely be called manually.
// To create a new registry use ProvideRegistryMDNS.
func New(serviceName string, serviceVersion string, cfg Config, log log.Logger) *RegistryMDNS {
	if cfg.Timeout == 0 {
		cfg.Timeout = registry.DefaultTimeout
	}

	if cfg.Domain == "" {
		cfg.Domain = DefaultDomain
	}

	return &RegistryMDNS{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		config:         cfg,
		logger:         log,
		services:       make(map[string][]*mdnsEntry),
		watchers:       make(map[string]*mdnsWatcher),
	}
}

// Start the registry.
func (m *RegistryMDNS) Start() error {
	// TODO: I think this should start something?
	return nil
}

// Stop the registry.
func (m *RegistryMDNS) Stop(_ context.Context) error {
	// TODO: do something here?
	return nil
}

// String returns the plugin name.
func (m *RegistryMDNS) String() string {
	return Name
}

// Type returns the component type.
func (m *RegistryMDNS) Type() string {
	return registry.ComponentType
}

// ServiceName returns the configured name of this service.
func (m *RegistryMDNS) ServiceName() string {
	return m.serviceName
}

// ServiceVersion returns the configured version of this service.
func (m *RegistryMDNS) ServiceVersion() string {
	return m.serviceVersion
}

// Register registes a service's nodes to the registry.
func (m *RegistryMDNS) Register(service *registry.Service, _ ...registry.RegisterOption) error {
	m.Lock()
	entries, ok := m.services[service.Name]
	m.Unlock()

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

	entries, err := m.registerNodes(service, entries)

	// Save
	m.Lock()
	m.services[service.Name] = entries
	m.Unlock()

	return err
}

//nolint:funlen
func (m *RegistryMDNS) registerNodes(service *registry.Service, entries []*mdnsEntry) ([]*mdnsEntry, error) {
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
		}

		// Doesn't exist
		entry = &mdnsEntry{}

		// encode the Transport with the metadata if not already given.
		if node.Metadata == nil {
			node.Metadata = make(map[string]string)
		}

		if _, ok := node.Metadata[metaTransportKey]; !ok {
			node.Metadata[metaTransportKey] = node.Transport
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

		m.logger.Trace("created a new node",
			slog.String("id", node.ID),
			slog.String("address", node.Address),
			slog.String("transport", node.Transport),
		)
	}

	return entries, gerr
}

// Deregister a service from the registry.
func (m *RegistryMDNS) Deregister(service *registry.Service, _ ...registry.DeregisterOption) error {
	m.Lock()
	defer m.Unlock()

	var newEntries []*mdnsEntry

	// loop existing entries, check if any match, shutdown those that do
	for _, entry := range m.services[service.Name] {
		var remove bool

		for _, node := range service.Nodes {
			if node.ID == entry.id {
				if err := entry.node.Shutdown(); err != nil {
					m.logger.Error("Failed to shutdown node", "err", err)
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
			m.logger.Error("failed to shutdown node", "err", err)
		}

		delete(m.services, service.Name)
	} else {
		m.services[service.Name] = newEntries
	}

	return nil
}

// GetService fetches a service from the registry.
func (m *RegistryMDNS) GetService(service string, _ ...registry.GetOption) ([]*registry.Service, error) {
	serviceMap := make(map[string]*registry.Service)
	entries := make(chan *client.ServiceEntry, 10)
	params := client.DefaultParams(service)

	// Set context with timeout
	var cancel context.CancelFunc

	params.Context, cancel = context.WithTimeout(context.Background(), time.Duration(m.config.Timeout)*time.Millisecond)
	defer cancel()

	params.Entries = entries
	params.Domain = m.config.Domain

	done := make(chan bool)
	go m.getService(service, params, serviceMap, entries, done)

	// Execute the query
	if err := client.Query(params); err != nil {
		return nil, err
	}

	// Wait for completion
	<-done

	// Create list and return
	services := make([]*registry.Service, len(serviceMap))

	var i int

	for _, service := range serviceMap {
		services[i] = service
		i++
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
				addr = net.JoinHostPort(entry.AddrV4.String(), strconv.Itoa(entry.Port))
			// Else use IPv6
			case len(entry.AddrV6) > 0:
				addr = net.JoinHostPort(entry.AddrV6.String(), strconv.Itoa(entry.Port))
			default:
				m.logger.Info("[mdns]: invalid endpoint received", "entry", entry.Name)
				continue
			}

			rNode := &registry.Node{
				ID:       strings.TrimSuffix(entry.Name, "."+params.Service+"."+params.Domain+"."),
				Address:  addr,
				Metadata: txt.Metadata,
			}

			// Fetch the trans back from the metadata.
			if trans, ok := rNode.Metadata[metaTransportKey]; ok {
				rNode.Transport = trans
			}

			service.Nodes = append(service.Nodes, rNode)

			serviceMap[txt.Version] = service
		case <-params.Context.Done():
			close(done)
			return
		}
	}
}

// ListServices fetches all services in the registry.
func (m *RegistryMDNS) ListServices(_ ...registry.ListOption) ([]*registry.Service, error) {
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
			m.logger.Error("Failed to listen", "err", err)
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
				addr = net.JoinHostPort(entry.AddrV4.String(), strconv.Itoa(entry.Port))
			case len(entry.AddrV6) > 0:
				addr = net.JoinHostPort(entry.AddrV6.String(), strconv.Itoa(entry.Port))
			}

			rNode := &registry.Node{
				ID:       strings.TrimSuffix(entry.Name, suffix),
				Address:  addr,
				Metadata: txt.Metadata,
			}

			if _, ok := rNode.Metadata[metaTransportKey]; ok {
				rNode.Transport = rNode.Metadata[metaTransportKey]
			}

			service.Nodes = append(service.Nodes, rNode)

			return &registry.Result{
				Action:  action,
				Service: service,
			}, nil
		case <-m.exit:
			return nil, registry.ErrWatcherStopped
		}
	}
}

func (m *mdnsWatcher) Stop() error {
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
		// remove self from the registry
		m.registry.mtx.Lock()
		delete(m.registry.watchers, m.id)
		m.registry.mtx.Unlock()

		return nil
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
