// Package natsjs provides the nats kvstore client for go-orb.
package natsjs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cornelk/hashmap"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// keyValueEnvelope is the data structure stored in the key value store, if JsonKeyValues is enabled.
type keyValueEnvelope struct {
	Key      string         `json:"key"`
	Data     []byte         `json:"data"`
	Metadata map[string]any `json:"metadata"`
}

// This is here to make sure it implements kvstore.KVStore.
var _ kvstore.KVStore = (*NatsJS)(nil)
var _ kvstore.Watcher = (*NatsJS)(nil)

// NatsJS implements the kvstore.KVStore interface using NATS JetStream.
type NatsJS struct {
	serviceName string

	config Config
	logger log.Logger

	codec codecs.Marshaler

	ctx     context.Context
	buckets *hashmap.Map[string, jetstream.KeyValue]

	nc *nats.Conn
	js jetstream.JetStream
}

// New creates a new NATS JetStream KVStore. This function should rarely be called manually.
// To create a new KVStore use Provide.
func New(serviceName string, cfg Config, log log.Logger) (*NatsJS, error) {
	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		return nil, err
	}

	return &NatsJS{
		serviceName: serviceName,
		config:      cfg,
		logger:      log,
		codec:       codec,
		ctx:         context.Background(), // Initialize with a background context
	}, nil
}

// Provide creates a new NatsJS KVStore client.
func Provide(
	name types.ServiceName,
	data types.ConfigData,
	logger log.Logger,
	opts ...kvstore.Option,
) (kvstore.Type, error) {
	cfg := NewConfig(opts...)

	sections := types.SplitServiceName(name)
	sections = append(sections, kvstore.DefaultConfigSection)

	if err := config.Parse(sections, data, &cfg); err != nil {
		return kvstore.Type{}, err
	}

	instance, err := New(string(name), cfg, logger)
	if err != nil {
		return kvstore.Type{}, err
	}

	return kvstore.Type{KVStore: instance}, nil
}

// Start initializes the connection to NATS JetStream.
func (n *NatsJS) Start(ctx context.Context) error {
	// Save the context for later use
	n.ctx = ctx
	nopts := n.config.NatsOptions.ToOptions()

	n.buckets = hashmap.New[string, jetstream.KeyValue]()

	var err error

	n.nc, err = nopts.Connect()
	if err != nil {
		return err
	}

	// Create a JetStream management interface
	n.js, err = jetstream.New(n.nc)
	if err != nil {
		return err
	}

	return nil
}

// Stop closes the connection to NATS JetStream.
func (n *NatsJS) Stop(_ context.Context) error {
	// Clear the KV stores
	n.buckets = nil

	// Close the connection to nats jetstream.
	n.js = nil

	if n.nc != nil {
		n.nc.Close()
		n.nc = nil
	}

	return nil
}

// String returns the plugin name.
func (n *NatsJS) String() string {
	return Name
}

// Type returns the component type.
func (n *NatsJS) Type() string {
	return kvstore.ComponentType
}

// orbDataToNATS converts key and value to nats key value.
func (n *NatsJS) orbDataToNATS(table, key string, value []byte) (string, []byte, error) {
	if n.config.JSONKeyValues {
		b, err := n.codec.Marshal(&keyValueEnvelope{
			Key:      key,
			Data:     value,
			Metadata: map[string]any{},
		})
		if err != nil {
			return "", nil, err
		}

		return natsKey(table, key, n.config.KeyEncoding, n.config.BucketPerTable), b, nil
	}

	return natsKey(table, key, n.config.KeyEncoding, n.config.BucketPerTable), value, nil
}

// natsDataToOrb converts nats key value to key and value.
func (n *NatsJS) natsDataToOrb(table, key string, value []byte) (string, []byte, error) {
	if n.config.JSONKeyValues {
		var envelope keyValueEnvelope
		err := n.codec.Unmarshal(value, &envelope)

		if err != nil {
			return "", nil, err
		}

		return envelope.Key, envelope.Data, nil
	}

	return orbKey(table, key, n.config.KeyEncoding, n.config.BucketPerTable), value, nil
}

// getKVStore gets or creates a KeyValue store for the given database and table.
func (n *NatsJS) getKVStore(database, table string) (jetstream.KeyValue, error) {
	if n.js == nil {
		return nil, errors.New("not connected to NATS")
	}

	// Use default database/table if not provided
	if database == "" {
		database = n.config.Database
	}

	if table == "" {
		table = n.config.Table
	}

	bucketName := bucketName(database, table, n.config.BucketPerTable)

	// Check if we already have this KV store
	if kv, ok := n.buckets.Get(bucketName); ok {
		return kv, nil
	}

	// Try to get existing bucket
	kv, err := n.js.KeyValue(n.ctx, bucketName)
	if err == nil {
		n.buckets.Set(bucketName, kv)
		return kv, nil
	}

	// Create a new bucket if it doesn't exist
	kv, err = n.js.CreateKeyValue(n.ctx, jetstream.KeyValueConfig{
		Bucket:       bucketName,
		Description:  n.config.BucketDescription,
		MaxValueSize: -1, // No limit
		History:      1,  // Only keep the latest value
		TTL:          0,  // No expiration
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create KV bucket: %w", err)
	}

	n.buckets.Set(bucketName, kv)

	return kv, nil
}

// Get takes a key, database, table and optional GetOptions. It returns the Record or an error.
func (n *NatsJS) Get(key, database, table string, _ ...kvstore.GetOption) ([]kvstore.Record, error) {
	kv, err := n.getKVStore(database, table)
	if err != nil {
		return nil, err
	}

	entry, err := kv.Get(n.ctx, natsKey(table, key, n.config.KeyEncoding, n.config.BucketPerTable))
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, orberrors.ErrNotFound.Wrap(kvstore.ErrNotFound)
		}

		return nil, orberrors.ErrInternalServerError.Wrap(err)
	}

	key, data, err := n.natsDataToOrb(table, entry.Key(), entry.Value())
	if err != nil {
		return nil, orberrors.ErrInternalServerError.Wrap(err)
	}

	record := kvstore.Record{
		Key:   key,
		Value: data,
	}

	return []kvstore.Record{record}, nil
}

// Set takes a key, database, table and data, and optional SetOptions.
func (n *NatsJS) Set(key, database, table string, data []byte, _ ...kvstore.SetOption) error {
	kv, err := n.getKVStore(database, table)
	if err != nil {
		return err
	}

	key, data, err = n.orbDataToNATS(table, key, data)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	_, err = kv.Put(n.ctx, key, data)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return nil
}

// Purge takes a key, database and table and purges it.
func (n *NatsJS) Purge(key, database, table string) error {
	kv, err := n.getKVStore(database, table)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return kv.Purge(n.ctx, natsKey(table, key, n.config.KeyEncoding, n.config.BucketPerTable))
}

// Keys returns any keys that match, or an empty list with no error if none matched.
func (n *NatsJS) Keys(database, table string, opts ...kvstore.KeysOption) ([]string, error) {
	options := kvstore.NewKeysOptions(opts...)

	kv, err := n.getKVStore(database, table)
	if err != nil {
		return nil, orberrors.ErrInternalServerError.Wrap(err)
	}

	// Get all keys
	keys, err := kv.Keys(n.ctx, jetstream.IgnoreDeletes())
	if err != nil {
		return nil, orberrors.ErrInternalServerError.Wrap(err)
	}

	// Filter keys based on options
	//nolint:prealloc
	var filteredKeys []string

	for idx, k := range keys {
		if options.Offset != 0 && uint(idx) < options.Offset { //nolint:gosec
			continue
		}

		if options.Limit != 0 && uint(idx) >= options.Offset+options.Limit { //nolint:gosec
			break
		}

		key, ok := orbKeyFilter(table, k, n.config.KeyEncoding, options.Prefix, options.Suffix, n.config.BucketPerTable)
		if !ok {
			continue
		}

		filteredKeys = append(filteredKeys, key)
	}

	return filteredKeys, nil
}

// DropTable drops the table.
func (n *NatsJS) DropTable(database, table string) error {
	if n.js == nil {
		return errors.New("not connected to NATS")
	}

	if !n.config.BucketPerTable {
		return errors.New("can't drop table when bucket per table is disabled")
	}

	// Use default database/table if not provided
	if database == "" {
		database = n.config.Database
	}

	if table == "" {
		table = n.config.Table
	}

	// Create a bucket name from database and table
	bucketName := bucketName(database, table, n.config.BucketPerTable)

	// Remove from our cache
	n.buckets.Del(bucketName)

	// Delete the bucket
	return n.js.DeleteKeyValue(n.ctx, bucketName)
}

// DropDatabase drops the database.
func (n *NatsJS) DropDatabase(database string) error {
	if n.js == nil {
		return errors.New("not connected to NATS")
	}

	// Use default database if not provided
	if database == "" {
		database = n.config.Database
	}

	// Delete all KV stores with the database prefix
	for name := range n.js.KeyValueStoreNames(n.ctx).Name() {
		// Check if the bucket name starts with the database name
		if strings.HasPrefix(name, database) {
			// Remove from our cache
			n.buckets.Del(name)

			// Delete the bucket
			err := n.js.DeleteKeyValue(n.ctx, name)
			if err != nil {
				n.logger.Error("failed to delete KV bucket", "bucket", name, "error", err)
			}
		}
	}

	return nil
}

// Watch exposes the watcher interface from the underlying JetStreamContext.
func (n *NatsJS) Watch(
	ctx context.Context,
	database,
	table string,
	opts ...kvstore.WatchOption,
) (<-chan kvstore.WatchEvent, func() error, error) {
	b, err := n.getKVStore(database, table)
	if err != nil {
		return nil, nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("failed to get bucket: %w", err))
	}

	orbOpts := kvstore.NewWatchOptions(opts...)

	natsOpts := []jetstream.WatchOpt{}
	if orbOpts.IgnoreDeletes {
		natsOpts = append(natsOpts, jetstream.IgnoreDeletes())
	}

	if orbOpts.IncludeHistory {
		natsOpts = append(natsOpts, jetstream.IncludeHistory())
	}

	if orbOpts.UpdatesOnly {
		natsOpts = append(natsOpts, jetstream.UpdatesOnly())
	}

	if orbOpts.MetaOnly {
		natsOpts = append(natsOpts, jetstream.MetaOnly())
	}

	watcher, err := b.WatchAll(ctx, natsOpts...)
	if err != nil {
		return nil, nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("failed to watch bucket: %w", err))
	}

	ch := make(chan kvstore.WatchEvent)

	go func() {
		for u := range watcher.Updates() {
			if u == nil {
				continue
			}

			var (
				action kvstore.WatchOp
				key    string
				data   []byte
				err    error
			)

			switch u.Operation() {
			case jetstream.KeyValuePut:
				action = kvstore.WatchOpUpdate
				key, data, err = n.natsDataToOrb(table, u.Key(), u.Value())

				if err != nil {
					continue
				}
			case jetstream.KeyValueDelete:
				fallthrough
			case jetstream.KeyValuePurge:
				action = kvstore.WatchOpDelete
				key = orbKey(table, u.Key(), n.config.KeyEncoding, n.config.BucketPerTable)
			}

			ch <- kvstore.WatchEvent{
				Record: kvstore.Record{
					Key:   key,
					Value: data,
				},
				Operation: action,
			}
		}
	}()

	return ch, watcher.Stop, nil
}

// Read takes a single key and optional ReadOptions. It returns matching []*Record or an error.
// Deprecated: use Get instead.
func (n *NatsJS) Read(key string, opts ...kvstore.ReadOption) ([]*kvstore.Record, error) {
	options := kvstore.NewReadOptions(opts...)

	records, err := n.Get(key, options.Database, options.Table)
	if err != nil {
		return nil, err
	}

	// Convert to pointer records
	ptrRecords := make([]*kvstore.Record, len(records))
	for i := range records {
		ptrRecords[i] = &records[i]
	}

	return ptrRecords, nil
}

// Write takes a single key and value, and optional WriteOptions.
// Deprecated: use Set instead.
func (n *NatsJS) Write(r *kvstore.Record, opts ...kvstore.WriteOption) error {
	options := kvstore.NewWriteOptions(opts...)

	return n.Set(r.Key, options.Database, options.Table, r.Value)
}

// Delete removes the record with the corresponding key from the store.
// Deprecated: use Remove instead.
func (n *NatsJS) Delete(key string, opts ...kvstore.DeleteOption) error {
	options := kvstore.NewDeleteOptions(opts...)
	return n.Purge(key, options.Database, options.Table)
}

// List returns any keys that match, or an empty list with no error if none matched.
// Deprecated: use Keys instead.
func (n *NatsJS) List(opts ...kvstore.ListOption) ([]string, error) {
	options := kvstore.NewListOptions(opts...)

	// Convert ListOptions to KeysOptions
	keysOpts := []kvstore.KeysOption{}
	if options.Prefix != "" {
		keysOpts = append(keysOpts, kvstore.KeysPrefix(options.Prefix))
	}

	if options.Suffix != "" {
		keysOpts = append(keysOpts, kvstore.KeysSuffix(options.Suffix))
	}

	if options.Limit > 0 {
		keysOpts = append(keysOpts, kvstore.KeysLimit(options.Limit))
	}

	if options.Offset > 0 {
		keysOpts = append(keysOpts, kvstore.KeysOffset(options.Offset))
	}

	return n.Keys(options.Database, options.Table, keysOpts...)
}
