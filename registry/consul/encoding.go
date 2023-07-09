package consul

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/go-orb/go-orb/registry"
)

func encode(buf []byte) string {
	var b bytes.Buffer
	defer b.Reset()

	w := zlib.NewWriter(&b)
	if _, err := w.Write(buf); err != nil {
		return ""
	}

	w.Close() //nolint:errcheck,gosec

	return hex.EncodeToString(b.Bytes())
}

func decode(d string) []byte {
	hr, err := hex.DecodeString(d)
	if err != nil {
		return nil
	}

	br := bytes.NewReader(hr)
	zr, err := zlib.NewReader(br)

	if err != nil {
		return nil
	}

	rbuf, err := io.ReadAll(zr)

	if err != nil {
		return nil
	}

	zr.Close() //nolint:errcheck,gosec

	return rbuf
}

func encodeEndpoints(en []*registry.Endpoint) []string {
	var tags []string

	for _, e := range en {
		if b, err := json.Marshal(e); err == nil {
			tags = append(tags, "e-"+encode(b))
		}
	}

	return tags
}

func decodeEndpoints(tags []string) []*registry.Endpoint {
	var endpoint []*registry.Endpoint

	// use the first format you find
	var ver byte

	for _, tag := range tags {
		if len(tag) == 0 || tag[0] != 'e' {
			continue
		}

		// check version
		if ver > 0 && tag[1] != ver {
			continue
		}

		var (
			e   *registry.Endpoint
			buf []byte
		)

		// Old encoding was plain
		if tag[1] == '=' {
			buf = []byte(tag[2:])
		}

		// New encoding is hex
		if tag[1] == '-' {
			buf = decode(tag[2:])
		}

		if err := json.Unmarshal(buf, &e); err == nil {
			endpoint = append(endpoint, e)
		}

		// set version
		ver = tag[1]
	}

	return endpoint
}

func encodeVersion(v string) []string {
	return []string{"v-" + encode([]byte(v))}
}

func decodeVersion(tags []string) (string, bool) {
	for _, tag := range tags {
		if len(tag) < 2 || tag[0] != 'v' {
			continue
		}

		// Old encoding was plain
		if tag[1] == '=' {
			return tag[2:], true
		}

		// New encoding is hex
		if tag[1] == '-' {
			return string(decode(tag[2:])), true
		}
	}

	return "", false
}
