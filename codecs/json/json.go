// Package json implements a JSON Marshaler.
package json

import (
	"encoding/json"
	"io"

	"go-micro.dev/v5/codecs"
)

var _ codecs.Marshaler = (*Json)(nil)

func init() {
	if err := codecs.Plugins.Add("json", &Json{}); err != nil {
		panic(err)
	}
}

type Json struct{}

func (j *Json) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j *Json) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// NewEncoder returns a new JSON/ProtocolBuffer encoder.
func (j *Json) NewEncoder(w io.Writer) codecs.Encoder {
	return json.NewEncoder(w)
}

// NewDecoder returns a new JSON/ProtocolBuffer decoder.
func (j *Json) NewDecoder(r io.Reader) codecs.Decoder {
	return json.NewDecoder(r)
}

// ContentTypes returns the content types the marshaller can handle.
func (j *Json) ContentTypes() []string {
	return []string{
		"application/json",
	}
}

// String returns the plugin implementation of the marshaler.
func (j *Json) String() string {
	return "json"
}

func (j *Json) Exts() []string {
	return []string{".json"}
}
