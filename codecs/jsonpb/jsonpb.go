// Package jsonpb implements a JSON <> Protocol Buffer marshaler that supports
// more protocol buffer options thatn the default stdlib JSON marshaler.
package jsonpb

import (
	"io"

	"github.com/go-orb/go-orb/codecs"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

var _ codecs.Marshaler = (*JSONPb)(nil)

func init() {
	codecs.Register("jsonpb", &JSONPb{})
}

// JSONPb wraps Google's implementation of a JSON <> Protocol buffer marshaller
// that has more extented support for protocol buffer fields.
type JSONPb struct {
	runtime.JSONPb
}

// NewEncoder returns a new JSON/ProtocolBuffer encoder.
func (j *JSONPb) NewEncoder(w io.Writer) codecs.Encoder {
	return j.JSONPb.NewEncoder(w)
}

// NewDecoder returns a new JSON/ProtocolBuffer decoder.
func (j *JSONPb) NewDecoder(r io.Reader) codecs.Decoder {
	return j.JSONPb.NewDecoder(r)
}

// ContentTypes returns the content types the marshaller can handle.
func (j *JSONPb) ContentTypes() []string {
	return []string{
		"application/json",
		"application/json; charset=utf-8",
	}
}

// String returns the plugin implementation of the marshaler.
func (j *JSONPb) String() string {
	return "jsonpb"
}

// Exts is a list of file extensions this marshaler supports.
func (j *JSONPb) Exts() []string {
	return []string{""}
}
