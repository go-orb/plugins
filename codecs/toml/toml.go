// Package toml implements a TOML Marshaler.
package toml

import (
	"io"

	"github.com/go-orb/go-orb/codecs"

	"github.com/BurntSushi/toml"
)

var _ codecs.Marshaler = (*Toml)(nil)

func init() {
	codecs.Register("toml", &Toml{})
}

// Toml implements the codecs.Marshaler interface. It can be used to encode/decode
// toml files, or web requests.
type Toml struct{}

// Marshal encodes "v" into byte sequence.
func (t *Toml) Marshal(v any) ([]byte, error) {
	return toml.Marshal(v)
}

// Unmarshal decodes "data" into "v".
// "v" must be a pointer value.
func (t *Toml) Unmarshal(data []byte, v any) error {
	return toml.Unmarshal(data, v)
}

// NewEncoder returns an Encoder which writes bytes sequence into "w".
func (t *Toml) NewEncoder(w io.Writer) codecs.Encoder {
	encoder := toml.NewEncoder(w)

	return codecs.EncoderFunc(encoder.Encode)
}

// NewDecoder returns a Decoder which reads byte sequence from "r".
func (t *Toml) NewDecoder(r io.Reader) codecs.Decoder {
	decoder := toml.NewDecoder(r)

	return codecs.DecoderFunc(func(v any) error {
		_, err := decoder.Decode(v)

		return err
	})
}

// Marshals returns if this codec is able to encode the given type.
func (t *Toml) Marshals(v any) bool {
	switch v.(type) {
	case []string:
		return true
	case []byte:
		return true
	case []any:
		return true
	case map[string]any:
		return true
	case string:
		return true
	default:
		return false
	}
}

// Unmarshals returns if this codec is able to decode the given type.
func (t *Toml) Unmarshals(v any) bool {
	return t.Marshals(v)
}

// ContentTypes returns the content types the marshaller can handle.
func (t *Toml) ContentTypes() []string {
	return []string{
		"application/toml",
		"text/toml",
	}
}

// Name returns the codec name.
func (t *Toml) Name() string {
	return "toml"
}

// Exts returns the common file extensions for this encoder.
func (t *Toml) Exts() []string {
	return []string{".toml", ".tml"}
}
