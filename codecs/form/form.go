package form

// Source:
// https://github.com/go-kratos/kratos/blob/main/encoding/form/proto_encode.go
//
// This code has been copied over as the original package does not export the
// requuired types, and performs various unrequired init operations.

import (
	"io"
	"net/url"

	"github.com/go-playground/form/v4"
	"google.golang.org/protobuf/proto"

	"go-micro.dev/v5/codecs"
)

var _ codecs.Marshaler = (*Form)(nil)

const (
	// Name is form codec name.
	Name        = "form"
	ContentType = "x-www-form-urlencoded"
	// Null value string.
	nullStr = "null"
)

// Form is used to encode/decode HTML form values as used in GET request URL
// query parameters or POST request bodies.
type Form struct {
	encoder *form.Encoder
	decoder *form.Decoder
}

func init() {
	if err := codecs.Plugins.Add(Name, NewFormCodec()); err != nil {
		panic(err)
	}
}

// NewFormCodec will create a codec used to encode/decode HTML form values as
// used in GET request URL query parameters or POST request bodies.
func NewFormCodec() *Form {
	return &Form{
		encoder: form.NewEncoder(),
		decoder: form.NewDecoder(),
	}
}

func (c Form) Marshal(v any) ([]byte, error) {
	var (
		vs  url.Values
		err error
	)

	if m, ok := v.(proto.Message); ok {
		vs, err = c.EncodeValues(m)
		if err != nil {
			return nil, err
		}
	} else {
		vs, err = c.encoder.Encode(v)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range vs {
		if len(v) == 0 {
			delete(vs, k)
		}
	}

	return []byte(vs.Encode()), nil
}

func (c Form) Unmarshal(data []byte, v any) error {
	vs, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}

	if m, ok := v.(proto.Message); ok {
		return DecodeValues(m, vs)
	}

	return c.decoder.Decode(v, vs)
}

// NewDecoder returns a Decoder which reads byte sequence from "r".
func (p Form) NewDecoder(r io.Reader) codecs.Decoder {
	return codecs.DecoderFunc(func(v any) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		return p.Unmarshal(b, v)
	})
}

// NewEncoder returns an Encoder which writes bytes sequence into "w".
func (p Form) NewEncoder(w io.Writer) codecs.Encoder {
	return codecs.EncoderFunc(func(v any) error {
		b, err := p.Marshal(v)
		if err != nil {
			return err
		}

		_, err = w.Write(b)

		return err
	})
}

func (Form) String() string {
	return Name
}

// ContentTypes returns the Content-Type which this marshaler is responsible for.
// The parameter describes the type which is being marshalled, which can sometimes
// affect the content type returned.
func (p *Form) ContentTypes() []string {
	return []string{ContentType}
}
