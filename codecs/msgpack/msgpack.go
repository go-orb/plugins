// Package msgpack contains the msgpack en/de-coder.
package msgpack

import (
	"io"

	"github.com/go-orb/go-orb/codecs"

	"github.com/shamaton/msgpack/v2"
)

var _ codecs.Marshaler = (*Codec)(nil)

func init() {
	codecs.Register("msgpack", &Codec{})
}

// Codec implements the codecs.Marshal interface, and can be used for marshaling
// Codec config files, and web requests.
type Codec struct{}

// Marshal marshals any object into json bytes.
// Param v should be a pointer type.
func (j *Codec) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Unmarshal decodes json bytes into object v.
// Param v should be a pointer type.
func (j *Codec) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

type wrapEncoder struct {
	w io.Writer
}

func (j *wrapEncoder) Encode(v any) error {
	return msgpack.MarshalWrite(j.w, v)
}

type wrapDecoder struct {
	r io.Reader
}

func (j *wrapDecoder) Decode(v any) error {
	return msgpack.UnmarshalRead(j.r, v)
}

// NewEncoder returns a new JSON encoder.
func (j *Codec) NewEncoder(w io.Writer) codecs.Encoder {
	return &wrapEncoder{w: w}
}

// NewDecoder returns a new JSON decoder.
func (j *Codec) NewDecoder(r io.Reader) codecs.Decoder {
	return &wrapDecoder{r: r}
}

// Marshals returns if this is able to encode the given type.
func (j *Codec) Marshals(_ any) bool {
	return true
}

// Unmarshals returns if this is able to decode the given type.
func (j *Codec) Unmarshals(_ any) bool {
	return true
}

// ContentTypes returns the content types the marshaler can handle.
func (j *Codec) ContentTypes() []string {
	return []string{
		codecs.MimeMsgpack,
	}
}

// Name returns the codec name.
func (j *Codec) Name() string {
	return "msgpack"
}

// Exts is a list of file extensions this marshaler supports.
func (j *Codec) Exts() []string {
	return []string{".msgpack"}
}
