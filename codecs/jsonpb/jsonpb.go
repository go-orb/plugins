// Package jsonpb implements a JSON <> Protocol Buffer marshaler that supports
// more protocol buffer options thatn the default stdlib JSON marshaler.
package jsonpb

import (
	"io"

	"github.com/go-orb/go-orb/codecs"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ codecs.Marshaler = (*JSONPb)(nil)

func init() {
	codecs.Register("jsonpb", &JSONPb{})
}

type mEncoder struct {
	w io.Writer
}

func (m *mEncoder) Encode(v any) error {
	b, err := protojson.Marshal(v.(protoreflect.ProtoMessage))
	if err != nil {
		return err
	}

	_, err = m.w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

type mDecoder struct {
	r io.Reader
}

func (m *mDecoder) Decode(v any) error {
	b, err := io.ReadAll(m.r)
	if err != nil {
		return err
	}

	return protojson.Unmarshal(b, v.(protoreflect.ProtoMessage))
}

// JSONPb wraps Google's implementation of a JSON <> Protocol buffer marshaller
// that has more extented support for protocol buffer fields.
type JSONPb struct{}

func (j *JSONPb) Marshal(v any) ([]byte, error) {
	return protojson.Marshal(v.(protoreflect.ProtoMessage))
}

func (j *JSONPb) Unmarshal(b []byte, v any) error {
	return protojson.Unmarshal(b, v.(protoreflect.ProtoMessage))
}

// NewEncoder returns a new JSON/ProtocolBuffer encoder.
func (j *JSONPb) NewEncoder(w io.Writer) codecs.Encoder {
	return &mEncoder{w: w}
}

// NewDecoder returns a new JSON/ProtocolBuffer decoder.
func (j *JSONPb) NewDecoder(r io.Reader) codecs.Decoder {
	return &mDecoder{r: r}
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
