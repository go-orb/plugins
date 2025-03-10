// Package goccyjson contains a fast replacement for encoding/json.
package goccyjson

import (
	"io"

	"github.com/go-orb/go-orb/codecs"

	"github.com/goccy/go-json"
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

type wrapEncoder struct {
	w    io.Writer
	impl *json.Encoder
}

func (j *wrapEncoder) Encode(v any) error {
	switch vt := v.(type) {
	case string:
		_, err := j.w.Write([]byte(vt))
		return err
	default:
		return j.impl.Encode(v)
	}
}

// NewEncoder returns a new JSON encoder.
func (j *CodecJSON) NewEncoder(w io.Writer) codecs.Encoder {
	return &wrapEncoder{w: w, impl: json.NewEncoder(w)}
}

type wrapDecoder struct {
	impl *json.Decoder
}

func (j *wrapDecoder) Decode(v any) error {
	return j.impl.Decode(v)
}

// NewDecoder returns a new JSON decoder.
func (j *CodecJSON) NewDecoder(r io.Reader) codecs.Decoder {
	return &wrapDecoder{impl: json.NewDecoder(r)}
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
