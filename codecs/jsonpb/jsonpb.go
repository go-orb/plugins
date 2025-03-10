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
	b, err := protojson.Marshal(v.(protoreflect.ProtoMessage)) //nolint:errcheck
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

	return protojson.Unmarshal(b, v.(protoreflect.ProtoMessage)) //nolint:errcheck
}

// JSONPb wraps Google's implementation of a JSON <> Protocol buffer marshaller
// that has more extented support for protocol buffer fields.
type JSONPb struct{}

// Marshal marshals any object into json bytes.
// Param v should be a pointer type.
func (j *JSONPb) Marshal(v any) ([]byte, error) {
	return protojson.Marshal(v.(protoreflect.ProtoMessage)) //nolint:errcheck
}

// Unmarshal decodes json bytes into object v.
// Param v should be a pointer type.
func (j *JSONPb) Unmarshal(b []byte, v any) error {
	return protojson.Unmarshal(b, v.(protoreflect.ProtoMessage)) //nolint:errcheck
}

// NewEncoder returns a new JSON/ProtocolBuffer encoder.
func (j *JSONPb) NewEncoder(w io.Writer) codecs.Encoder {
	return &mEncoder{w: w}
}

// NewDecoder returns a new JSON/ProtocolBuffer decoder.
func (j *JSONPb) NewDecoder(r io.Reader) codecs.Decoder {
	return &mDecoder{r: r}
}

// Marshals returns if this is able to encode the given type.
func (j *JSONPb) Marshals(v any) bool {
	_, ok := v.(protoreflect.ProtoMessage)

	return ok
}

// Unmarshals returns if this is able to decode the given type.
func (j *JSONPb) Unmarshals(v any) bool {
	_, ok := v.(protoreflect.ProtoMessage)

	return ok
}

// ContentTypes returns the content types the marshaller can handle.
func (j *JSONPb) ContentTypes() []string {
	return []string{
		codecs.MimeJSON,
	}
}

// Name returns the codec name.
func (j *JSONPb) Name() string {
	return "jsonpb"
}

// Exts is a list of file extensions this marshaler supports.
func (j *JSONPb) Exts() []string {
	return []string{""}
}
