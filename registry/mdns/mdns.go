// Package mdns provides a multicast dns registry
package mdns

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/go-orb/plugins/registry/mdns/client"
	"github.com/go-orb/plugins/registry/mdns/dns"
	"github.com/go-orb/plugins/registry/mdns/server"
	"github.com/go-orb/plugins/registry/mdns/zone"
	"github.com/go-orb/plugins/registry/regutil/cache"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/lithammer/shortuuid/v3"
)

const metaPrefix = "orb_app_"
const myMetaPrefix = "orb_internal_"
const nodeIDDelimiter = "@"
const nodeKeyDelimiter = "_"

func nodeID(s registry.ServiceNode) string {
	return strings.Join([]string{
		s.Namespace,
		s.Region,
		s.Name,
		s.Version,
		s.Node,
		s.Scheme,
	}, nodeIDDelimiter)
}

func nodeKey(s registry.ServiceNode) string {
	if s.Version == "" {
		return s.Node
	}

	return strings.Join([]string{
		s.Node,
		s.Version,
	}, nodeKeyDelimiter)
}

func serviceKey(namespace, region, name string) string {
	return strings.Join([]string{
		name,
		region,
		namespace,
	}, nodeKeyDelimiter)
}

func serviceDomain(namespace, region, domain string) string {
	if namespace == "" {
		namespace = "default"
	}

	if region == "" {
		region = "default"
	}

	return fmt.Sprintf("%s.%s.%s", namespace, region, domain)
}

func idToNode(id string) (registry.ServiceNode, error) {
	parts := strings.Split(id, nodeIDDelimiter)
	if len(parts) != 6 {
		return registry.ServiceNode{}, errors.New("invalid id format")
	}

	return registry.ServiceNode{
		Namespace: parts[0],
		Region:    parts[1],
		Name:      parts[2],
		Version:   parts[3],
		Node:      parts[4],
		Scheme:    parts[5],
		Metadata:  make(map[string]string),
	}, nil
}

type mdnsTxt struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
}

type mdnsEntry struct {
	id   string
	node *server.Server
}

// This is here to make sure RegistryMDNS implements registry.Registry.
var _ registry.Registry = (*Registry)(nil)

// Registry implements the registry interface. It runs a MDNS service registry.
type Registry struct {
	config Config
	logger log.Logger
	codec  codecs.Marshaler

	sync.Mutex
	services map[string][]*mdnsEntry

	mtx sync.RWMutex

	// watchers
	watchers map[string]*mdnsWatcher

	// watchListener
	watchListener chan *client.ServiceEntry

	// cache is used to cache registry operations.
	cache *cache.Cache
}

type mdnsWatcher struct {
	ctx context.Context
	id  string
	wo  registry.WatchOptions
	ch  chan *client.ServiceEntry
	// the mdns domain
	domain string

	logger log.Logger
	codec  codecs.Marshaler
}

// Provide creates a new MDNS registry.
func Provide(
	datas map[string]any,
	_ *types.Components,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg := NewConfig(opts...)
	if err := config.Parse(nil, registry.DefaultConfigSection, datas, &cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return registry.Type{}, fmt.Errorf("parse config: %w", err)
	}

	// Return the new registry.
	reg, err := New(cfg, logger)
	if err != nil {
		return registry.Type{}, fmt.Errorf("create registry: %w", err)
	}

	return registry.Type{Registry: reg}, nil
}

// New creates a new mdns registry. This functions should rarely be called manually.
// To create a new registry use ProvideRegistryMDNS.
func New(cfg Config, log log.Logger) (*Registry, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = registry.DefaultTimeout
	}

	if cfg.Domain == "" {
		cfg.Domain = DefaultDomain
	}

	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		return nil, fmt.Errorf("get codec: %w", err)
	}

	reg := &Registry{
		config:   cfg,
		codec:    codec,
		logger:   log,
		services: make(map[string][]*mdnsEntry),
		watchers: make(map[string]*mdnsWatcher),
	}

	// Initialize the cache with a reference to this registry
	reg.cache = cache.New(cache.Config{}, log, reg)

	return reg, nil
}

// Start the registry.
func (m *Registry) Start(ctx context.Context) error {
	return m.cache.Start(ctx)
}

// Stop the registry.
func (m *Registry) Stop(ctx context.Context) error {
	return m.cache.Stop(ctx)
}

// String returns the plugin name.
func (m *Registry) String() string {
	return Name
}

// Type returns the component type.
func (m *Registry) Type() string {
	return registry.ComponentType
}

// Register registes a service's nodes to the registry.
func (m *Registry) Register(_ context.Context, serviceNode registry.ServiceNode) error {
	if err := serviceNode.Valid(); err != nil {
		return err
	}

	m.Lock()
	entries, ok := m.services[serviceKey(serviceNode.Namespace, serviceNode.Region, serviceNode.Name)]
	m.Unlock()

	// first entry, create wildcard used for list queries
	if !ok {
		s, err := zone.NewMDNSService(
			serviceNode.Name,
			"_services",
			serviceDomain(serviceNode.Namespace, serviceNode.Region, m.config.Domain)+".",
			"",
			9999,
			[]net.IP{net.ParseIP("0.0.0.0")},
			nil,
		)
		if err != nil {
			return err
		}

		srv, err := server.NewServer(
			&server.Config{Zone: &dns.SDService{MDNSService: s}, LocalhostChecking: true},
			m.logger,
		)
		if err != nil {
			return err
		}

		// append the wildcard entry
		entries = append(entries, &mdnsEntry{id: "*", node: srv})
	}

	entries, err := m.registerNode(serviceNode, entries)

	// Save
	m.Lock()
	m.services[serviceKey(serviceNode.Namespace, serviceNode.Region, serviceNode.Name)] = entries
	m.Unlock()

	return err
}

func (m *Registry) registerNode(node registry.ServiceNode, entries []*mdnsEntry) ([]*mdnsEntry, error) {
	id := nodeID(node)

	entry := &mdnsEntry{}

	// Add service metadata to the service
	metadata := make(map[string]string)
	for k, v := range node.Metadata {
		metadata[metaPrefix+k] = v
	}

	metadata[myMetaPrefix+"network"] = node.Network

	var s zone.Zone

	if node.Network == "unix" {
		metadata[myMetaPrefix+"address"] = node.Address

		txt, err := encode(m.codec, &mdnsTxt{
			ID:       id,
			Metadata: metadata,
		})

		if err != nil {
			return entries, err
		}

		s, err = zone.NewMDNSService(
			nodeKey(node),
			node.Name,
			serviceDomain(node.Namespace, node.Region, m.config.Domain)+".",
			"",
			9999,
			[]net.IP{net.ParseIP("0.0.0.0")},
			txt,
		)
		if err != nil {
			return entries, err
		}
	} else {
		txt, err := encode(m.codec, &mdnsTxt{
			ID:       id,
			Metadata: metadata,
		})

		if err != nil {
			return entries, err
		}

		host, pt, err := net.SplitHostPort(node.Address)
		if err != nil {
			return entries, err
		}

		port, _ := strconv.Atoi(pt) //nolint:errcheck

		// we got here, new node
		s, err = zone.NewMDNSService(
			nodeKey(node),
			node.Name,
			serviceDomain(node.Namespace, node.Region, m.config.Domain)+".",
			"",
			port,
			[]net.IP{net.ParseIP(host)},
			txt,
		)
		if err != nil {
			return entries, err
		}
	}

	srv, err := server.NewServer(
		&server.Config{Zone: s, LocalhostChecking: true},
		m.logger,
	)
	if err != nil {
		return entries, err
	}

	entry.id = id
	entry.node = srv

	// Check if the entry already exists
	idx := slices.IndexFunc(entries, func(e *mdnsEntry) bool {
		return e.id == id
	})

	if idx == -1 {
		entries = append(entries, entry)

		m.logger.Trace("created a new node",
			slog.String("id", id),
		)
	} else {
		m.logger.Trace("updating existing node",
			slog.String("id", id),
		)

		if err := entries[idx].node.Shutdown(); err != nil {
			m.logger.Error("Failed to shutdown node", "err", err)
		}

		entries[idx] = entry
	}

	return entries, nil
}

// Deregister a service from the registry.
func (m *Registry) Deregister(_ context.Context, node registry.ServiceNode) error {
	m.Lock()
	defer m.Unlock()

	// Create a unique node ID similar to registration
	nodeID := nodeID(node)

	var keepEntries []*mdnsEntry

	// loop existing entries, check if any match, shutdown those that do
	for _, entry := range m.services[serviceKey(node.Namespace, node.Region, node.Name)] {
		var remove bool

		if entry.id == nodeID {
			m.logger.Trace("deregistering", "node", nodeID)

			if err := entry.node.Shutdown(); err != nil {
				m.logger.Error("Failed to shutdown node", "err", err)
			}

			remove = true
		}

		// keep it?
		if !remove {
			keepEntries = append(keepEntries, entry)
		}
	}

	// last entry is the wildcard for list queries. Remove it.
	if len(keepEntries) == 1 && keepEntries[0].id == "*" {
		m.logger.Trace("unlisting", "node", node.Name)

		if err := keepEntries[0].node.Shutdown(); err != nil {
			m.logger.Error("failed to shutdown node", "err", err)
		}

		delete(m.services, node.Name)
	} else {
		m.services[node.Name] = keepEntries
	}

	return nil
}

// GetService fetches a service from the registry.
//
//nolint:gocognit,gocyclo,funlen
func (m *Registry) GetService(ctx context.Context, namespace, region, name string, schemes []string) ([]registry.ServiceNode, error) {
	nodes, err := m.cache.GetService(ctx, namespace, region, name, schemes)
	if err == nil {
		return nodes, nil
	}

	entries := make(chan *client.ServiceEntry, 24)
	params := client.DefaultParams(name)

	// Set context with timeout
	var cancel context.CancelFunc

	qCtx, cancel := context.WithTimeout(ctx, time.Duration(m.config.Timeout))
	defer cancel()

	params.Context = qCtx
	params.Entries = entries
	params.Domain = serviceDomain(namespace, region, m.config.Domain)

	// Execute the query
	go func() {
		if err := client.Query(params, m.logger); err != nil {
			m.logger.Error("Failed to query", "err", err)
		}
	}()

	// Filter the nodes.
GET_SERVICE:
	for {
		select {
		case entry := <-entries:
			if entry.TTL == 0 {
				m.logger.Trace("Skipping zero TTL")
				continue
			}

			txt, err := decode(m.codec, entry.InfoFields)
			if err != nil {
				m.logger.Warn("Failed to decode entry", "err", err, "entry", entry.InfoFields)
				continue
			}

			// Create a service node from the entry
			serviceNode, err := idToNode(txt.ID)
			if err != nil {
				m.logger.Warn("Failed to create service node", "err", err, "id", txt.ID)
				continue
			}

			for k, v := range txt.Metadata {
				if strings.HasPrefix(k, myMetaPrefix) {
					continue
				}

				if strings.HasPrefix(k, metaPrefix) {
					serviceNode.Metadata[strings.TrimPrefix(k, metaPrefix)] = v
				}
			}

			if txt.Metadata[myMetaPrefix+"network"] == "unix" {
				serviceNode.Address = txt.Metadata[myMetaPrefix+"address"]
			} else {
				switch {
				// Prefer IPv4 addrs
				case len(entry.AddrV4) > 0:
					serviceNode.Address = net.JoinHostPort(entry.AddrV4.String(), strconv.Itoa(entry.Port))
				// Else use IPv6
				case len(entry.AddrV6) > 0:
					serviceNode.Address = net.JoinHostPort(entry.AddrV6.String(), strconv.Itoa(entry.Port))
				default:
					m.logger.Info("invalid address received", "entry", entry.Name)
					continue
				}
			}

			// Add the service node to the list
			nodes = append(nodes, serviceNode)
		case <-qCtx.Done():
			break GET_SERVICE
		}
	}

	if len(nodes) == 0 {
		return nil, registry.ErrNotFound
	}

	for _, node := range nodes {
		if err := m.cache.Register(ctx, node); err != nil {
			m.logger.Warn("failed to register service with cache", "err", err)
		}
	}

	return m.cache.GetService(ctx, namespace, region, name, schemes)
}

// ListServices fetches all services in the registry.
func (m *Registry) ListServices(ctx context.Context, namespace, region string, schemes []string) ([]registry.ServiceNode, error) {
	nodes, err := m.cache.ListServices(ctx, namespace, region, schemes)
	if err == nil {
		return nodes, nil
	}

	entries := make(chan *client.ServiceEntry, 10)

	params := client.DefaultParams("_services")

	// set context with timeout
	var cancel context.CancelFunc

	params.Context, cancel = context.WithTimeout(context.Background(), time.Duration(m.config.Timeout))
	defer cancel()

	params.Entries = entries
	params.Domain = serviceDomain(namespace, region, m.config.Domain)

	var services []registry.ServiceNode

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

				name := strings.TrimSuffix(e.Name, "."+params.Service+"."+serviceDomain(namespace, region, m.config.Domain)+".")
				services = append(services, registry.ServiceNode{Name: name, Namespace: namespace, Region: region})
			case <-params.Context.Done():
				return
			}
		}
	}()

	// execute query
	if err := client.Query(params, m.logger); err != nil {
		return nil, err
	}

	// wait till done
	<-params.Context.Done()

	return services, nil
}

// Watch for registration changes.
func (m *Registry) Watch(ctx context.Context, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	watcher := &mdnsWatcher{
		ctx:    ctx,
		id:     shortuuid.New(),
		wo:     wo,
		ch:     make(chan *client.ServiceEntry, 32),
		domain: m.config.Domain,
		logger: m.logger.With("watcher", "mdns"),
		codec:  m.codec,
	}

	m.mtx.Lock()

	// Save the watcher
	m.watchers[watcher.id] = watcher

	// Check existing listener
	if m.watchListener != nil {
		m.mtx.Unlock()

		m.logger.Debug("Listener already exists, skipping")

		return watcher, nil
	}

	// Reset the listener
	m.watchListener = make(chan *client.ServiceEntry, 32)

	m.mtx.Unlock()

	// Send messages to the watchers
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-m.watchListener:
				if !ok {
					continue
				}

				// Skip entries that do not match the domain
				if !strings.HasSuffix(e.Name, m.config.Domain+".") {
					continue
				}

				// Send service entry to all watchers
				m.mtx.RLock()
				for _, w := range m.watchers {
					select {
					case w.ch <- e:
					default:
					}
				}
				m.mtx.RUnlock()
			}
		}
	}()

	go func() {
		// Start listening, blocking call
		if err := client.Listen(ctx, m.logger, m.watchListener); err != nil {
			m.logger.Error("Failed to listen", "err", err)
		}

		// mdns.Listen has unblocked
		// Kill the saved listener
		m.mtx.Lock()

		close(m.watchListener)

		m.mtx.Unlock()
	}()

	return watcher, nil
}

func (w *mdnsWatcher) Next() (*registry.Result, error) {
	select {
	case <-w.ctx.Done():
		return nil, registry.ErrWatcherStopped
	case entry := <-w.ch:
		txt, err := decode(w.codec, entry.InfoFields)
		if err != nil {
			w.logger.Warn("Failed to decode entry", "err", err, "entry", entry.InfoFields)
			return nil, fmt.Errorf("failed to decode entry: %w", err)
		}

		// Create a service node from the entry
		serviceNode, err := idToNode(txt.ID)
		if err != nil {
			w.logger.Warn("Failed to create service node", "err", err, "id", txt.ID)
			return nil, fmt.Errorf("failed to create service node: %w", err)
		}

		for k, v := range txt.Metadata {
			if strings.HasPrefix(k, myMetaPrefix) {
				continue
			}

			if strings.HasPrefix(k, metaPrefix) {
				serviceNode.Metadata[strings.TrimPrefix(k, metaPrefix)] = v
			}
		}

		if txt.Metadata[myMetaPrefix+"network"] == "unix" {
			serviceNode.Address = txt.Metadata[myMetaPrefix+"address"]
		} else {
			switch {
			// Prefer IPv4 addrs
			case len(entry.AddrV4) > 0:
				serviceNode.Address = net.JoinHostPort(entry.AddrV4.String(), strconv.Itoa(entry.Port))
			// Else use IPv6
			case len(entry.AddrV6) > 0:
				serviceNode.Address = net.JoinHostPort(entry.AddrV6.String(), strconv.Itoa(entry.Port))
			default:
				w.logger.Info("invalid address received", "entry", entry.Name)
				return nil, fmt.Errorf("invalid address received: %s", entry.Name)
			}
		}

		// Filter for TTL
		if entry.TTL <= 0 {
			return &registry.Result{
				Action: registry.Delete,
				Node:   serviceNode,
			}, nil
		}

		// Create the service
		return &registry.Result{
			Action: registry.Create,
			Node:   serviceNode,
		}, nil
	}
}

func encode(codec codecs.Marshaler, txt *mdnsTxt) ([]string, error) {
	b, err := codec.Marshal(txt)
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

func decode(codec codecs.Marshaler, record []string) (*mdnsTxt, error) {
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

	if err := codec.Unmarshal(rbuf, &txt); err != nil {
		return nil, err
	}

	return txt, nil
}
