package natsjs

import (
	"encoding/base32"
	"strings"
)

// bucketName creates a bucket name by combining the table and database if bucketPerTable is true.
func bucketName(database, table string, bucketPerTable bool) string {
	if bucketPerTable && table != "" {
		return database + "_" + table
	}

	return database
}

// natsKey converts a orb key to a nats key (encoded for the nats kv store).
func natsKey(table, orbkey, keyEncoding string, bucketPerTable bool) string {
	if orbkey == "" {
		return ""
	}

	fullKey := getKey(orbkey, table, bucketPerTable)

	return encode(fullKey, keyEncoding)
}

// orbKey converts a nats key to a orb key (plain text without table prefix).
func orbKey(table, natskey, keyEncoding string, bucketPerTable bool) string {
	fullKey := decode(natskey, keyEncoding)
	return trimKey(fullKey, table, bucketPerTable)
}

// orbKeyFilter converts a nats key to a orb key and checks if it matches the given filters.
// It returns the orb key and a boolean indicating if the key passes the filter.
func orbKeyFilter(table, natskey, keyEncoding, prefix, suffix string, bucketPerTable bool) (string, bool) {
	fullKey := decode(natskey, keyEncoding)
	orbKey := trimKey(fullKey, table, bucketPerTable)

	// Check if the key matches the filters
	if table != "" && fullKey != getKey(orbKey, table, bucketPerTable) {
		return orbKey, false
	}

	if prefix != "" && !strings.HasPrefix(orbKey, prefix) {
		return orbKey, false
	}

	if suffix != "" && !strings.HasSuffix(orbKey, suffix) {
		return orbKey, false
	}

	return orbKey, true
}

// encode encodes a string using the specified algorithm.
func encode(s string, alg string) string {
	switch alg {
	case "base32":
		return base32.StdEncoding.EncodeToString([]byte(s))
	default:
		return s
	}
}

// decode decodes a string using the specified algorithm.
func decode(s string, alg string) string {
	switch alg {
	case "base32":
		b, err := base32.StdEncoding.DecodeString(s)
		if err != nil {
			return s
		}

		return string(b)
	default:
		return s
	}
}

// getKey creates a full key by combining the table and key.
func getKey(key, table string, bucketPerTable bool) string {
	if bucketPerTable && table != "" {
		return table + "_" + key
	}

	return key
}

// trimKey removes the table prefix from a full key.
func trimKey(key, table string, bucketPerTable bool) string {
	if bucketPerTable && table != "" {
		return strings.TrimPrefix(key, table+"_")
	}

	return key
}
