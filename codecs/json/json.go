// Package json contains the encoding/json en/de-coder.
package json

import (
	"encoding/json"
	"io"

	"github.com/go-orb/go-orb/codecs"
)

var _ codecs.Marshaler = (*CodecJSON)(nil)

func init() {
	codecs.Register("json", &CodecJSON{})
}

// CodecJSON implements the codecs.Marshal interface, and can be used for marshaling
// CodecJSON config files, and web requests.
type CodecJSON struct{}

// Marshal marshals any object into json bytes.
// Param v should be a pointer type.
func (j *CodecJSON) Marshal(v any) ([]byte, error) {
	switch vt := v.(type) {
	case string:
		return []byte(vt), nil
	default:
		return json.Marshal(v)
	}
}

// Unmarshal decodes json bytes into object v.
// Param v should be a pointer type.
func (j *CodecJSON) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// NewEncoder returns an Encoder which writes bytes sequence into "w".
func (j *CodecJSON) NewEncoder(w io.Writer) codecs.Encoder {
	encoder := json.NewEncoder(w)

	return codecs.EncoderFunc(encoder.Encode)
}

// NewDecoder returns a Decoder which reads byte sequence from "r".
func (j *CodecJSON) NewDecoder(r io.Reader) codecs.Decoder {
	decoder := json.NewDecoder(r)

	return codecs.DecoderFunc(decoder.Decode)
}

// Marshals returns if this is able to encode the given type.
func (j *CodecJSON) Marshals(_ any) bool {
	return true
}

// Unmarshals returns if this is able to decode the given type.
func (j *CodecJSON) Unmarshals(_ any) bool {
	return true
}

// ContentTypes returns the content types the marshaler can handle.
func (j *CodecJSON) ContentTypes() []string {
	return []string{
		codecs.MimeJSON,
	}
}

// Name returns the codec name.
func (j *CodecJSON) Name() string {
	return "json"
}

// Exts is a list of file extensions this marshaler supports.
func (j *CodecJSON) Exts() []string {
	return []string{".json"}
}
