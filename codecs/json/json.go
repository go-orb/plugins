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

// JSON implements the codecs.Marshal interface, and can be used for marshaling
// JSON config files, and web requests.
type JSON struct{}

// Marshal marshals any object into json bytes.
// Param v should be a pointer type.
func (j *JSON) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal decodes json bytes into object v.
// Param v should be a pointer type.
func (j *JSON) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// NewEncoder returns a new JSON encoder.
func (j *JSON) NewEncoder(w io.Writer) codecs.Encoder {
	return json.NewEncoder(w)
}

// NewDecoder returns a new JSON decoder.
func (j *JSON) NewDecoder(r io.Reader) codecs.Decoder {
	return json.NewDecoder(r)
}

// ContentTypes returns the content types the marshaler can handle.
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
