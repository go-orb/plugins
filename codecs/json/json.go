// Package json implements a JSON Marshaler.
package json

import (
	"encoding/json"
	"io"

	"go-micro.dev/v5/codecs"
)

var _ codecs.Marshaler = (*JSON)(nil)

func init() {
	if err := codecs.Plugins.Add("json", &JSON{}); err != nil {
		panic(err)
	}
}

// JSON is the JSON Codec for go-micro.
type JSON struct{}

// Marshal marshals any pointer into json byte.
func (j *JSON) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal decodes json byte to v pointer.
func (j *JSON) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// NewEncoder returns a new JSON/ProtocolBuffer encoder.
func (j *JSON) NewEncoder(w io.Writer) codecs.Encoder {
	return json.NewEncoder(w)
}

// NewDecoder returns a new JSON/ProtocolBuffer decoder.
func (j *JSON) NewDecoder(r io.Reader) codecs.Decoder {
	return json.NewDecoder(r)
}

// ContentTypes returns the content types the marshaller can handle.
func (j *JSON) ContentTypes() []string {
	return []string{
		"application/json",
	}
}

// String returns the plugin implementation of the marshaler.
func (j *JSON) String() string {
	return "json"
}

// Exts is a list of file extensions this encoder supports.
func (j *JSON) Exts() []string {
	return []string{".json"}
}
