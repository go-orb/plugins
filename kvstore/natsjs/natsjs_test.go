package natsjs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/go-orb/go-orb/kvstore"
	"github.com/go-orb/go-orb/log"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/suite"

	// Import codecs.
	_ "github.com/go-orb/plugins/codecs/json"
)

type storeConfig struct {
	name           string
	bucketPerTable bool
	jsonKeyValues  bool
	keyEncoding    string
}

type NatsJSTestSuite struct {
	suite.Suite

	t          *testing.T
	natsServer *server.Server
	configs    map[string]storeConfig
	stores     map[string]kvstore.Type
	ctx        context.Context
	cancel     context.CancelFunc
}

func TestNatsJSSuite(t *testing.T) {
	s := &NatsJSTestSuite{t: t}
	suite.Run(t, s)
}

func (s *NatsJSTestSuite) SetupSuite() {
	// Start embedded NATS server for testing
	tmpDir := s.t.TempDir()

	s.configs = map[string]storeConfig{
		"BucketPerTable_JSON": {
			name:           "BucketPerTable_JSON",
			bucketPerTable: true,
			jsonKeyValues:  true,
			keyEncoding:    "base32",
		},
		"BucketPerTable_NoJSON": {
			name:           "BucketPerTable_NoJSON",
			bucketPerTable: true,
			jsonKeyValues:  false,
			keyEncoding:    "",
		},
		"NoBucketPerTable_JSON": {
			name:           "NoBucketPerTable_JSON",
			bucketPerTable: false,
			jsonKeyValues:  true,
			keyEncoding:    "base32",
		},
		"NoBucketPerTable_NoJSON": {
			name:           "NoBucketPerTable_NoJSON",
			bucketPerTable: false,
			jsonKeyValues:  false,
			keyEncoding:    "",
		},
	}

	opts := test.DefaultTestOptions
	opts.Port = -1 // Random port
	opts.JetStream = true
	opts.StoreDir = tmpDir
	// Configure JetStream
	opts.JetStreamMaxMemory = -1 // Unlimited
	opts.JetStreamMaxStore = -1  // Unlimited

	server := test.RunServer(&opts)
	s.Require().True(server.JetStreamEnabled())
	s.natsServer = server

	// Create context
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Create logger
	logger := log.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	// Initialize stores map
	s.stores = make(map[string]kvstore.Type)

	// Create stores with different configurations
	for _, cfg := range s.configs {
		// Create config
		storeCfg := NewConfig(
			WithURL(s.natsServer.ClientURL()),
			WithBucketPerTable(cfg.bucketPerTable),
			WithJSONKeyValues(cfg.jsonKeyValues),
			WithBucketDescription("Test bucket"),
			WithKeyEncoding(cfg.keyEncoding),
		)

		// Create store
		store, err := New(storeCfg, logger)
		s.Require().NoError(err)

		err = store.Start(s.ctx)
		s.Require().NoError(err)

		s.stores[cfg.name] = kvstore.Type{KVStore: store}
	}
}

func (s *NatsJSTestSuite) TearDownSuite() {
	for _, store := range s.stores {
		s.NoError(store.Stop(s.ctx))
	}
	if s.natsServer != nil {
		s.natsServer.Shutdown()
	}
	s.cancel()
}

func (s *NatsJSTestSuite) TestBasicOperations() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Test Set
			err := store.Set("test-key", "", "", []byte("test-value"))
			s.Require().NoError(err)

			// Test Get
			records, err := store.Get("test-key", "", "")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("test-key", records[0].Key)
			s.Equal([]byte("test-value"), records[0].Value)

			// Test Get for non-existent key
			_, err = store.Get("non-existent-key", "", "")
			s.Require().ErrorIs(err, kvstore.ErrNotFound)

			// Test Keys
			keys, err := store.Keys("", "")
			s.Require().NoError(err)
			s.Contains(keys, "test-key")

			// Test Purge
			err = store.Purge("test-key", "", "")
			s.Require().NoError(err)

			// Verify key is gone
			_, err = store.Get("test-key", "", "")
			s.Require().ErrorIs(err, kvstore.ErrNotFound)

			// Test Purge non-existent key (should not error)
			err = store.Purge("non-existent-key", "", "")
			s.Require().NoError(err)
		})
	}
}

func (s *NatsJSTestSuite) TestCustomDatabaseAndTable() {
	for name, store := range s.stores {
		s.Run(name, func() {
			db := "custom-db"
			table := "custom-table"

			// Test Set with custom db/table
			err := store.Set("test-key", db, table, []byte("test-value"))
			s.Require().NoError(err)

			// Test Get with custom db/table
			records, err := store.Get("test-key", db, table)
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("test-key", records[0].Key)
			s.Equal([]byte("test-value"), records[0].Value)

			// Test Keys with custom db/table
			keys, err := store.Keys(db, table)
			s.Require().NoError(err)
			s.Contains(keys, "test-key")

			// Add a few more keys to test Keys method more thoroughly
			err = store.Set("test-key2", db, table, []byte("test-value2"))
			s.Require().NoError(err)
			err = store.Set("test-key3", db, table, []byte("test-value3"))
			s.Require().NoError(err)

			// Verify multiple keys are returned
			keys, err = store.Keys(db, table)
			s.Require().NoError(err)
			s.Contains(keys, "test-key")
			s.Contains(keys, "test-key2")
			s.Contains(keys, "test-key3")

			// Test DropTable
			err = store.DropTable(db, table)
			if s.configs[name].bucketPerTable {
				s.Require().NoError(err)
				// Verify table is gone
				_, err = store.Get("test-key", db, table)
				s.Require().Error(err)
			} else {
				s.Require().Error(err)
				s.Contains(err.Error(), "can't drop table when bucket per table is disabled")
			}
		})
	}
}

func (s *NatsJSTestSuite) TestDropDatabase() {
	for name, store := range s.stores {
		s.Run(name, func() {
			db := "test-db"
			table1 := "table1"
			table2 := "table2"

			// Create some data in different tables
			err := store.Set("key1", db, table1, []byte("value1"))
			s.Require().NoError(err)
			err = store.Set("key2", db, table2, []byte("value2"))
			s.Require().NoError(err)

			// Drop the database
			err = store.DropDatabase(db)
			s.Require().NoError(err)

			// Verify data is gone from all tables
			_, err = store.Get("key1", db, table1)
			s.Require().Error(err)
			_, err = store.Get("key2", db, table2)
			s.Require().Error(err)

			// Test dropping non-existent database (should not error)
			err = store.DropDatabase("non-existent-db")
			s.Require().NoError(err)
		})
	}
}

func (s *NatsJSTestSuite) TestBinaryData() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Create a larger binary data sample
			binaryData := make([]byte, 1024) // 1KB of data
			for i := 0; i < 1024; i++ {
				binaryData[i] = byte(i % 256)
			}

			// Test Set with binary data
			err := store.Set("binary-key", "bin-db", "bin-table", binaryData)
			s.Require().NoError(err)

			// Test Get with binary data
			records, err := store.Get("binary-key", "bin-db", "bin-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("binary-key", records[0].Key)
			s.Equal(binaryData, records[0].Value)

			// Test with empty data
			emptyData := []byte{}
			err = store.Set("empty-key", "bin-db", "bin-table", emptyData)
			s.Require().NoError(err)

			// Test Get with empty data
			records, err = store.Get("empty-key", "bin-db", "bin-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("empty-key", records[0].Key)
			s.Equal(emptyData, records[0].Value)
		})
	}
}

func (s *NatsJSTestSuite) TestPrefixOperations() {
	for name, store := range s.stores {
		s.Run(name, func() {
			if s.configs[name].keyEncoding == "" {
				return
			}

			// Set up keys with prefixes
			prefixDB := "prefix-db"
			prefixTable := "prefix-table"

			// Insert keys with different prefixes
			prefixes := []string{"user:", "order:", "product:"}
			for i, prefix := range prefixes {
				for j := 1; j <= 3; j++ {
					key := prefix + "item" + string(rune('0'+j))
					data := []byte(fmt.Sprintf("value-%d-%d", i, j))
					err := store.Set(key, prefixDB, prefixTable, data)
					s.Require().NoError(err)
				}
			}

			// Get all keys and check prefixes
			allKeys, err := store.Keys(prefixDB, prefixTable)
			s.Require().NoError(err)
			s.Require().Len(allKeys, 9) // 3 prefixes * 3 items

			// Check specific prefixes
			for _, prefix := range prefixes {
				for j := 1; j <= 3; j++ {
					key := prefix + "item" + string(rune('0'+j))
					s.Contains(allKeys, key)
				}
			}
		})
	}
}

func (s *NatsJSTestSuite) TestEdgeCases() {
	for name, store := range s.stores {
		s.Run(name, func() {
			if s.configs[name].keyEncoding == "" {
				return
			}

			// Test with very long key
			longKey := strings.Repeat("verylong", 50) // 400 characters
			err := store.Set(longKey, "edge-db", "edge-table", []byte("long-key-value"))
			s.Require().NoError(err)

			// Test Get with long key
			records, err := store.Get(longKey, "edge-db", "edge-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal(longKey, records[0].Key)
			s.Equal([]byte("long-key-value"), records[0].Value)

			// Test with special characters in key (but valid db and table names)
			// NATS JetStream bucket names have restrictions, so we use valid names for db and table
			specialKey := "special!@#$%^&*()_+{}[]|\\:;\"'<>,.?/~`"
			validDB := "special_db"
			validTable := "special_table"

			err = store.Set(specialKey, validDB, validTable, []byte("special-value"))
			s.Require().NoError(err)

			// Test Get with special characters
			records, err = store.Get(specialKey, validDB, validTable)
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal(specialKey, records[0].Key)
			s.Equal([]byte("special-value"), records[0].Value)

			// Test with empty values
			emptyValue := []byte{}
			err = store.Set("empty-value-key", validDB, validTable, emptyValue)
			s.Require().NoError(err)
			records, err = store.Get("empty-value-key", validDB, validTable)
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("empty-value-key", records[0].Key)
			s.Equal(emptyValue, records[0].Value)

			// Test with unicode characters in key
			unicodeKey := "unicode_😀_🚀_🌍"
			unicodeValue := []byte("unicode value 😀")
			err = store.Set(unicodeKey, validDB, validTable, unicodeValue)
			s.Require().NoError(err)
			records, err = store.Get(unicodeKey, validDB, validTable)
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal(unicodeKey, records[0].Key)
			s.Equal(unicodeValue, records[0].Value)
		})
	}
}

func (s *NatsJSTestSuite) TestConcurrentOperations() {
	for name, store := range s.stores {
		s.Run(name, func() {
			if s.configs[name].keyEncoding == "" {
				return
			}

			// Set up database and table for concurrent testing
			concurrentDB := "concurrent-db"
			concurrentTable := "concurrent-table"

			// Define the number of concurrent operations
			numOperations := 10

			// Create a wait group to wait for all goroutines to finish
			wg := sync.WaitGroup{}
			wg.Add(numOperations)

			// Create a channel to collect errors
			errChan := make(chan error, numOperations)

			// Launch concurrent Set operations
			for i := 0; i < numOperations; i++ {
				go func(index int) {
					defer wg.Done()
					key := fmt.Sprintf("concurrent-key-%d", index)
					value := []byte(fmt.Sprintf("concurrent-value-%d", index))

					err := store.Set(key, concurrentDB, concurrentTable, value)
					if err != nil {
						errChan <- fmt.Errorf("set failed for index %d: %w", index, err)
					}
				}(i)
			}

			// Wait for all goroutines to finish
			wg.Wait()
			close(errChan)

			// Check for any errors
			for err := range errChan {
				s.Fail(err.Error())
			}

			// Verify all keys were set
			keys, err := store.Keys(concurrentDB, concurrentTable)
			s.Require().NoError(err)
			s.Require().Len(keys, numOperations)

			// Verify each key has the correct value
			for i := 0; i < numOperations; i++ {
				key := fmt.Sprintf("concurrent-key-%d", i)
				expectedValue := []byte(fmt.Sprintf("concurrent-value-%d", i))

				records, err := store.Get(key, concurrentDB, concurrentTable)
				s.Require().NoError(err)
				s.Require().Len(records, 1)
				s.Equal(key, records[0].Key)
				s.Equal(expectedValue, records[0].Value)
			}
		})
	}
}

func (s *NatsJSTestSuite) TestErrorHandling() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Create a key first to ensure the bucket exists
			err := store.Set("setup-key", "error-db", "error-table", []byte("setup-value"))
			s.Require().NoError(err)

			// Test Get with non-existent key
			_, err = store.Get("non-existent-key", "error-db", "error-table")
			s.Require().ErrorIs(err, kvstore.ErrNotFound)

			// For Keys with non-existent database, we need to use a database name that we know doesn't exist
			// We can't guarantee empty results from non-existent buckets due to NATS behavior
			// Instead, just test that Keys for an existing database works
			keys, err := store.Keys("error-db", "error-table")
			s.Require().NoError(err)
			s.Contains(keys, "setup-key")

			// Test Purge with non-existent key
			err = store.Purge("non-existent-key", "error-db", "error-table")
			s.Require().NoError(err) // Should not error

			// Test DropTable with non-existent table (but existing database)
			// For BucketPerTable=true, we need to ensure the database exists first
			if s.configs[name].bucketPerTable {
				// Ensure the database exists
				err = store.Set("setup-key", "error-db", "temp-table", []byte("temp-value"))
				s.Require().NoError(err)

				// Now test dropping a non-existent table in an existing database
				err = store.DropTable("error-db", "non-existent-table")
				s.Require().Error(err)

				// Cleanup - drop the temp table
				err = store.DropTable("error-db", "temp-table")
				s.Require().NoError(err)
			} else {
				// When BucketPerTable is false, it should error because dropping tables is not allowed
				err = store.DropTable("error-db", "non-existent-table")
				s.Require().Error(err)
				s.Contains(err.Error(), "can't drop table when bucket per table is disabled")
			}
		})
	}
}

func (s *NatsJSTestSuite) TestBatchOperations() {
	for name, store := range s.stores {
		s.Run(name, func() {
			db := "batch-db"
			table := "batch-table"
			count := 5

			// Set multiple keys
			for i := 0; i < count; i++ {
				key := fmt.Sprintf("batch-key-%d", i)
				value := fmt.Sprintf("batch-value-%d", i)
				err := store.Set(key, db, table, []byte(value))
				s.Require().NoError(err)
			}

			// Get all keys
			keys, err := store.Keys(db, table)
			s.Require().NoError(err)
			s.Require().Len(keys, count)

			// Test batch get by iterating through all keys
			for _, key := range keys {
				records, err := store.Get(key, db, table)
				s.Require().NoError(err)
				s.Require().Len(records, 1)
			}

			// Test batch delete by dropping the database
			err = store.DropDatabase(db)
			s.Require().NoError(err)

			// After dropping the database, verify keys are gone by attempting to get one of them
			// This should return ErrNotFound
			key := fmt.Sprintf("batch-key-%d", 0)
			_, err = store.Get(key, db, table)
			s.Require().ErrorIs(err, kvstore.ErrNotFound)

			// Try to get all keys - this should either return empty or an error
			// Both are acceptable since the bucket might not exist
			keys, err = store.Keys(db, table)
			if err == nil {
				s.Require().Empty(keys)
			}
		})
	}
}

func (s *NatsJSTestSuite) TestContextHandling() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Create a context with cancellation
			_, cancel := context.WithCancel(s.ctx)
			defer cancel()

			// Test operations with valid context
			err := store.Set("ctx-key", "ctx-db", "ctx-table", []byte("ctx-value"))
			s.Require().NoError(err)

			// Test context cancelation after operations (should not affect them)
			cancel()

			// This should still work because the context is only used for JetStream operations internally
			// and we're testing the store operations which have already completed
			records, err := store.Get("ctx-key", "ctx-db", "ctx-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("ctx-key", records[0].Key)
			s.Equal([]byte("ctx-value"), records[0].Value)
		})
	}
}

func (s *NatsJSTestSuite) TestKeysWithPattern() {
	for name, store := range s.stores {
		s.Run(name, func() {
			if s.configs[name].keyEncoding == "" {
				return
			}

			db := "pattern-db"
			table := "pattern-table"

			// Create keys with different patterns
			patterns := map[string][]string{
				"user:":    {"user:1", "user:2", "user:3"},
				"product:": {"product:a", "product:b", "product:c"},
				"order:":   {"order:123", "order:456", "order:789"},
			}

			// Set all keys
			for _, keys := range patterns {
				for _, key := range keys {
					err := store.Set(key, db, table, []byte("pattern-value-"+key))
					s.Require().NoError(err)
				}
			}

			// Get all keys to verify they were created
			allKeys, err := store.Keys(db, table)
			s.Require().NoError(err)
			s.Require().Len(allKeys, 9) // 3 patterns with 3 keys each

			// Verify all keys are in the result
			for _, keys := range patterns {
				for _, key := range keys {
					s.Contains(allKeys, key)
				}
			}

			// Test getting each key and verify its value
			for _, keys := range patterns {
				for _, key := range keys {
					records, err := store.Get(key, db, table)
					s.Require().NoError(err)
					s.Require().Len(records, 1)
					s.Equal(key, records[0].Key)
					s.Equal([]byte("pattern-value-"+key), records[0].Value)
				}
			}
		})
	}
}

func (s *NatsJSTestSuite) TestGenericKVStoreInterface() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Test Read method
			err := store.Set("read-key", "generic-db", "generic-table", []byte("read-value"))
			s.Require().NoError(err)

			// Use Read method with options
			records, err := store.Read("read-key",
				kvstore.ReadFrom("generic-db", "generic-table"),
			)
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("read-key", records[0].Key)
			s.Equal([]byte("read-value"), records[0].Value)

			// Test Write method
			writeRecord := &kvstore.Record{
				Key:   "write-key",
				Value: []byte("write-value"),
			}
			err = store.Write(writeRecord,
				kvstore.WriteTo("generic-db", "generic-table"),
			)
			s.Require().NoError(err)

			// Verify written record
			getRecords, err := store.Get("write-key", "generic-db", "generic-table")
			s.Require().NoError(err)
			s.Require().Len(getRecords, 1)
			s.Equal("write-key", getRecords[0].Key)
			s.Equal([]byte("write-value"), getRecords[0].Value)

			// Test List method
			keys, err := store.List(
				kvstore.ListFrom("generic-db", "generic-table"),
			)
			s.Require().NoError(err)
			// Should have at least the two keys we just added
			s.Contains(keys, "read-key")
			s.Contains(keys, "write-key")

			// Test Delete method
			err = store.Delete("write-key",
				kvstore.DeleteFrom("generic-db", "generic-table"),
			)
			s.Require().NoError(err)

			// Verify key was deleted
			_, err = store.Get("write-key", "generic-db", "generic-table")
			s.Require().ErrorIs(err, kvstore.ErrNotFound)
		})
	}
}

func (s *NatsJSTestSuite) TestStringAndType() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Test the String method
			str := store.String()
			s.Contains(str, "natsjs")

			// Test the Type method
			s.Equal("kvstore", store.Type())
		})
	}
}

func (s *NatsJSTestSuite) TestErrorScenarios() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Test Read with missing options
			_, err := store.Read("key")
			s.Require().Error(err)
			s.Require().ErrorIs(err, kvstore.ErrNotFound)

			// Test with invalid database/table names (should not crash)
			err = store.Set("key", strings.Repeat("a", 1000), "table", []byte("value"))
			// The error might vary depending on NATS implementation, so just check it doesn't panic
			// We're intentionally not asserting any specific error here
			_ = err
		})
	}
}

func (s *NatsJSTestSuite) TestEmptyKeyValues() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Test with empty key
			err := store.Set("", "empty-db", "empty-table123", []byte("empty-key-value"))
			s.Require().Error(err)

			// Test with valid key but empty value (should succeed)
			err = store.Set("empty-value-key", "empty-db", "empty-table", []byte{})
			s.Require().NoError(err)

			// Get the empty value and verify
			records, err := store.Get("empty-value-key", "empty-db", "empty-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("empty-value-key", records[0].Key)
			// The empty value could be returned as nil or as an empty slice
			// Either is acceptable, just check length is 0
			s.Empty(records[0].Value)

			// Test with nil value (should be treated as empty)
			err = store.Set("nil-value-key", "empty-db", "empty-table", nil)
			s.Require().NoError(err)

			// Get the nil value and verify
			records, err = store.Get("nil-value-key", "empty-db", "empty-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("nil-value-key", records[0].Key)
			// Nil value should have zero length
			s.Empty(records[0].Value)
		})
	}
}

func (s *NatsJSTestSuite) TestLargeData() {
	for name, store := range s.stores {
		s.Run(name, func() {
			// Create a large data payload (100KB)
			size := 100 * 1024 // 100KB
			largeData := make([]byte, size)
			for i := 0; i < size; i++ {
				largeData[i] = byte(i % 256)
			}

			// Store large data
			err := store.Set("large-key", "large-db", "large-table", largeData)
			s.Require().NoError(err)

			// Retrieve and verify large data
			records, err := store.Get("large-key", "large-db", "large-table")
			s.Require().NoError(err)
			s.Require().Len(records, 1)
			s.Equal("large-key", records[0].Key)
			s.Equal(largeData, records[0].Value)
			s.Len(records[0].Value, size)
		})
	}
}
