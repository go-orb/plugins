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
		s.Scheme,
		s.Address,
	}, nodeIDDelimiter)
}

func nodeKey(s registry.ServiceNode) string {
	return strings.Join([]string{
		s.Scheme,
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
		Scheme:    parts[4],
		Address:   parts[5],
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
func (m *Registry) Register(_ context.Context, node registry.ServiceNode) error {
	m.Lock()
	entries, ok := m.services[serviceKey(node.Namespace, node.Region, node.Name)]
	m.Unlock()

	// first entry, create wildcard used for list queries
	if !ok {
		s, err := zone.NewMDNSService(
			serviceKey(node.Namespace, node.Region, node.Name),
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

	entries, err := m.registerNodes(node, entries)

	// Save
	m.Lock()
	m.services[serviceKey(node.Namespace, node.Region, node.Name)] = entries
	m.Unlock()

	return err
}

func (m *Registry) registerNodes(node registry.ServiceNode, entries []*mdnsEntry) ([]*mdnsEntry, error) {
	id := nodeID(node)

	entry := &mdnsEntry{}

	// Add service metadata to the service
	metadata := make(map[string]string)
	for k, v := range node.Metadata {
		metadata[metaPrefix+k] = v
	}

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
	s, err := zone.NewMDNSService(
		nodeKey(node),
		serviceKey(node.Namespace, node.Region, node.Name),
		m.config.Domain+".",
		"",
		port,
		[]net.IP{net.ParseIP(host)},
		txt,
	)
	if err != nil {
		return entries, err
	}

	srv, err := server.NewServer(&server.Config{Zone: s, LocalhostChecking: true})
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
		entries[idx] = entry

		m.logger.Trace("updated existing node",
			slog.String("id", id),
		)
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
			m.logger.Trace("deregistering node", "id", nodeID)

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
		m.logger.Trace("unlisting service", "service", node.Name)

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
func (m *Registry) GetService(ctx context.Context, namespace, region, name string, schemes []string) ([]registry.ServiceNode, error) {
	return m.cache.GetService(ctx, namespace, region, name, schemes)
}

// ListServices fetches all services in the registry.
func (m *Registry) ListServices(ctx context.Context, namespace, region string, schemes []string) ([]registry.ServiceNode, error) {
	return m.cache.ListServices(ctx, namespace, region, schemes)
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
		if err := client.Listen(ctx, m.watchListener); err != nil {
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
			w.logger.Debug("Failed to decode entry", "err", err)
			return nil, err
		}

		// Create a service node from the entry
		serviceNode, err := idToNode(txt.ID)
		if err != nil {
			w.logger.Debug("Failed to create service node", "err", err)
			return nil, err
		}

		for k, v := range txt.Metadata {
			if strings.HasPrefix(k, myMetaPrefix) {
				continue
			}

			if strings.HasPrefix(k, metaPrefix) {
				serviceNode.Metadata[strings.TrimPrefix(k, metaPrefix)] = v
			}
		}

		// Filter for TTL
		if entry.TTL == 0 {
			// Delete the service
			serviceNode.Metadata["mdns-ttl"] = "expired"

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
