package consul

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"io"
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
