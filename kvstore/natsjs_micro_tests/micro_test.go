package natsjsmicrotests

import (
	"context"
	"fmt"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"

	micronats "github.com/go-micro/plugins/v4/store/nats-js-kv"
	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/kvstore/natsjs"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
	"go-micro.dev/v4/store"

	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/log/slog"
)

type NatsJSMicroTestSuite struct {
	suite.Suite

	t *testing.T

	natsServer *server.Server

	// NATS server connection
	nc *nats.Conn

	// go-orb store
	orbStore *natsjs.NatsJS

	// go-micro store
	microStore store.Store
}

// SetupSuite establishes connection to NATS server and initializes both stores.
func (s *NatsJSMicroTestSuite) SetupSuite() {
	// Start embedded NATS server for testing
	tmpDir := s.t.TempDir()

	opts := test.DefaultTestOptions
	opts.Port = -1 // Random port
	opts.JetStream = true
	opts.StoreDir = tmpDir
	// Configure JetStream
	opts.JetStreamMaxMemory = -1      // Unlimited
	opts.JetStreamMaxStore = -1       // Unlimited
	opts.MaxPayload = 8 * 1024 * 1024 // 8MiB

	server := test.RunServer(&opts)
	s.Require().True(server.JetStreamEnabled())
	s.natsServer = server

	// Initialize go-orb store
	logger, err := log.New()
	s.Require().NoError(err)

	// Create config
	storeCfg := natsjs.NewConfig(
		natsjs.WithURL(s.natsServer.ClientURL()),
		natsjs.WithBucketPerTable(false),
		natsjs.WithJSONKeyValues(true),
		natsjs.WithKeyEncoding("base32"),
	)

	orbStore, err := natsjs.New(
		context.Background(),
		"test",
		storeCfg,
		logger,
	)
	s.Require().NoError(err)
	s.Require().NoError(orbStore.Start(context.Background()))
	s.orbStore = orbStore

	// Initialize go-micro store
	microStore := micronats.NewStore(
		store.Nodes(s.natsServer.ClientURL()),
		micronats.EncodeKeys(),
	)
	s.Require().NoError(microStore.Init())
	s.microStore = microStore
}

// TearDownSuite cleans up resources.
func (s *NatsJSMicroTestSuite) TearDownSuite() {
	s.nc.Close()
}

// TestInteroperability verifies that both implementations can read each other's data.
func (s *NatsJSMicroTestSuite) TestInteroperability() {
	// Test writing with go-orb and reading with go-micro
	s.Run("OrbWriteMicroRead", func() {
		// Write data using go-orb
		err := s.orbStore.Set("test-key", "", "", []byte("test-value"))
		s.Require().NoError(err)

		// Read using go-micro
		record, err := s.microStore.Read("test-key")
		s.Require().NoError(err)
		s.Require().Len(record, 1)
		s.Equal("test-value", string(record[0].Value))
	})

	// Test writing with go-micro and reading with go-orb
	s.Run("MicroWriteOrbRead", func() {
		// Write data using go-micro
		err := s.microStore.Write(&store.Record{
			Key:   "micro-key",
			Value: []byte("micro-value"),
		})
		s.Require().NoError(err)

		// Read using go-orb
		records, err := s.orbStore.Get("micro-key", "", "")
		s.Require().NoError(err)
		s.Require().Len(records, 1)
		s.Equal("micro-value", string(records[0].Value))
	})
}

// TestListInteroperability verifies that List operations work across implementations.
func (s *NatsJSMicroTestSuite) TestListInteroperability() {
	// Write multiple records with go-orb
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("orb-key-%d", i)
		value := fmt.Sprintf("orb-value-%d", i)
		err := s.orbStore.Set(key, "", "", []byte(value))
		s.Require().NoError(err)
	}

	// Write multiple records with go-micro
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("micro-key-%d", i)
		value := fmt.Sprintf("micro-value-%d", i)
		err := s.microStore.Write(&store.Record{
			Key:   key,
			Value: []byte(value),
		})
		s.Require().NoError(err)
	}

	// List all keys using go-orb
	orbKeys, err := s.orbStore.Keys("", "")
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(orbKeys), 6) // At least our 6 test keys

	// List all keys using go-micro
	microKeys, err := s.microStore.List()
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(microKeys), 6) // At least our 6 test keys

	// Verify both implementations can see each other's keys
	for i := 0; i < 3; i++ {
		orbKey := fmt.Sprintf("orb-key-%d", i)
		microKey := fmt.Sprintf("micro-key-%d", i)

		s.Contains(orbKeys, orbKey)
		s.Contains(orbKeys, microKey)
		s.Contains(microKeys, orbKey)
		s.Contains(microKeys, microKey)
	}
}

// TestDeleteInteroperability verifies that Delete operations work across implementations.
func (s *NatsJSMicroTestSuite) TestDeleteInteroperability() {
	// Write and delete with go-orb, verify with go-micro
	s.Run("OrbDeleteMicroVerify", func() {
		// Write with go-orb
		err := s.orbStore.Set("orb-delete-key", "", "", []byte("delete-me"))
		s.Require().NoError(err)

		// Verify go-micro can see it
		_, err = s.microStore.Read("orb-delete-key")
		s.Require().NoError(err)

		// Delete with go-orb
		err = s.orbStore.Purge("orb-delete-key", "", "")
		s.Require().NoError(err)

		// Verify it's gone in both implementations
		_, err = s.microStore.Read("orb-delete-key")
		s.Require().Error(err)
		_, err = s.orbStore.Get("orb-delete-key", "", "")
		s.Require().Error(err)
	})

	// Write and delete with go-micro, verify with go-orb
	s.Run("MicroDeleteOrbVerify", func() {
		// Write with go-micro
		err := s.microStore.Write(&store.Record{
			Key:   "micro-delete-key",
			Value: []byte("delete-me"),
		})
		s.Require().NoError(err)

		// Verify go-orb can see it
		_, err = s.orbStore.Get("micro-delete-key", "", "")
		s.Require().NoError(err)

		// Delete with go-micro
		err = s.microStore.Delete("micro-delete-key")
		s.Require().NoError(err)

		// Verify it's gone in both implementations
		_, err = s.microStore.Read("micro-delete-key")
		s.Require().Error(err)
		_, err = s.orbStore.Get("micro-delete-key", "", "")
		s.Require().Error(err)
	})
}

func TestNatsJSMicroSuite(t *testing.T) {
	suite.Run(t, &NatsJSMicroTestSuite{t: t})
}

// BenchmarkNatsJSMicro runs benchmarks comparing go-orb and go-micro implementations.
type BenchmarkNatsJSMicro struct {
	natsServer *server.Server
	orbStore   *natsjs.NatsJS
	microStore store.Store
}

// setupBenchmark initializes the stores for benchmarking.
func setupBenchmark(b *testing.B) *BenchmarkNatsJSMicro {
	b.Helper()

	// Start embedded NATS server for testing
	tmpDir := b.TempDir()

	opts := test.DefaultTestOptions
	opts.Port = -1 // Random port
	opts.JetStream = true
	opts.StoreDir = tmpDir
	// Configure JetStream
	opts.JetStreamMaxMemory = -1      // Unlimited
	opts.JetStreamMaxStore = -1       // Unlimited
	opts.MaxPayload = 8 * 1024 * 1024 // 8MiB

	server := test.RunServer(&opts)
	if !server.JetStreamEnabled() {
		b.Fatal("JetStream not enabled")
	}

	// Initialize go-orb store
	logger, err := log.New()
	if err != nil {
		b.Fatal(err)
	}

	// Create config
	storeCfg := natsjs.NewConfig(
		natsjs.WithURL(server.ClientURL()),
		natsjs.WithKeyEncoding("base32"),
	)

	orbStore, err := natsjs.New(
		context.Background(),
		"test",
		storeCfg,
		logger,
	)
	if err != nil {
		b.Fatal(err)
	}
	if err := orbStore.Start(context.Background()); err != nil {
		b.Fatal(err)
	}

	// Initialize go-micro store
	microStore := micronats.NewStore(
		store.Nodes(server.ClientURL()),
		micronats.EncodeKeys(),
	)
	if err := microStore.Init(); err != nil {
		b.Fatal(err)
	}

	return &BenchmarkNatsJSMicro{
		natsServer: server,
		orbStore:   orbStore,
		microStore: microStore,
	}
}

// BenchmarkOrbSet benchmarks the Set operation in go-orb.
func BenchmarkOrbSet(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.orbStore.Set(key, "", "", []byte(value)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroWrite benchmarks the Write operation in go-micro.
func BenchmarkMicroWrite(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: []byte(value),
		}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbSetGet benchmarks the Set and Get operations in go-orb.
func BenchmarkOrbSetGet(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.orbStore.Set(key, "", "", []byte(value)); err != nil {
			b.Fatal(err)
		}
		if _, err := bm.orbStore.Get(key, "", ""); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroWriteRead benchmarks the Write and Read operations in go-micro.
func BenchmarkMicroWriteRead(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: []byte(value),
		}); err != nil {
			b.Fatal(err)
		}
		if _, err := bm.microStore.Read(key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbGet benchmarks the Get operation in go-orb.
func BenchmarkOrbGet(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Pre-populate data
	key := "test-key"
	value := "test-value"
	if err := bm.orbStore.Set(key, "", "", []byte(value)); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.orbStore.Get(key, "", "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroRead benchmarks the Read operation in go-micro.
func BenchmarkMicroRead(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Pre-populate data
	key := "test-key"
	value := "test-value"
	if err := bm.microStore.Write(&store.Record{
		Key:   key,
		Value: []byte(value),
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.microStore.Read(key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbList benchmarks the Keys operation in go-orb.
func BenchmarkOrbList(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Pre-populate data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.orbStore.Set(key, "", "", []byte(value)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.orbStore.Keys("", "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroList benchmarks the List operation in go-micro.
func BenchmarkMicroList(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Pre-populate data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: []byte(value),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.microStore.List()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbListPagination benchmarks the Keys operation with pagination in go-orb.
func BenchmarkOrbListPagination(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Pre-populate data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.orbStore.Set(key, "", "", []byte(value)); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.orbStore.Keys("", "", kvstore.KeysLimit(100), kvstore.KeysOffset(100))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroList benchmarks the List operation in go-micro.
func BenchmarkMicroListPagination(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Pre-populate data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: []byte(value),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.microStore.List(store.ListLimit(100), store.ListOffset(100))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// generateLargeValue generates a value of the specified size in bytes.
func generateLargeValue(size int) []byte {
	value := make([]byte, size)
	for i := 0; i < size; i++ {
		value[i] = byte(i % 256)
	}
	return value
}

// BenchmarkOrbSetLarge benchmarks the Set operation with large values in go-orb.
func BenchmarkOrbSetLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value
	value := generateLargeValue(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("large-key-%d", i)
		if err := bm.orbStore.Set(key, "", "", value); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroWriteLarge benchmarks the Write operation with large values in go-micro.
func BenchmarkMicroWriteLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value
	value := generateLargeValue(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("large-key-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: value,
		}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbGetLarge benchmarks the Get operation with large values in go-orb.
func BenchmarkOrbGetLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value and store it
	key := "large-test-key"
	value := generateLargeValue(1024 * 1024)
	if err := bm.orbStore.Set(key, "", "", value); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.orbStore.Get(key, "", "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroReadLarge benchmarks the Read operation with large values in go-micro.
func BenchmarkMicroReadLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value and store it
	key := "large-test-key"
	value := generateLargeValue(1024 * 1024)
	if err := bm.microStore.Write(&store.Record{
		Key:   key,
		Value: value,
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.microStore.Read(key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbSetGetLarge benchmarks the Set and Get operations with large values in go-orb.
func BenchmarkOrbSetGetLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value
	value := generateLargeValue(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("large-key-%d", i)
		if err := bm.orbStore.Set(key, "", "", value); err != nil {
			b.Fatal(err)
		}
		if _, err := bm.orbStore.Get(key, "", ""); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroWriteReadLarge benchmarks the Write and Read operations with large values in go-micro.
func BenchmarkMicroWriteReadLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value
	value := generateLargeValue(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("large-key-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: value,
		}); err != nil {
			b.Fatal(err)
		}
		if _, err := bm.microStore.Read(key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrbDeleteLarge benchmarks the Delete operation with large values in go-orb.
func BenchmarkOrbDeleteLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value
	value := generateLargeValue(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Write a value first
		key := fmt.Sprintf("large-key-%d", i)
		if err := bm.orbStore.Set(key, "", "", value); err != nil {
			b.Fatal(err)
		}
		// Then delete it
		if err := bm.orbStore.Purge(key, "", ""); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMicroDeleteLarge benchmarks the Delete operation with large values in go-micro.
func BenchmarkMicroDeleteLarge(b *testing.B) {
	bm := setupBenchmark(b)
	defer bm.natsServer.Shutdown()

	// Generate 1MB value
	value := generateLargeValue(1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Write a value first
		key := fmt.Sprintf("large-key-%d", i)
		if err := bm.microStore.Write(&store.Record{
			Key:   key,
			Value: value,
		}); err != nil {
			b.Fatal(err)
		}
		// Then delete it
		if err := bm.microStore.Delete(key); err != nil {
			b.Fatal(err)
		}
	}
}
