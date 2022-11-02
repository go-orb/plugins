package form

// Source:
// https://github.com/go-kratos/kratos/blob/main/encoding/form/proto_encode.go
// This code has been copied over as the original package does not export the
// requuired types, and does various init operations.

import (
	"io"
	"net/url"
	"reflect"

	kform "github.com/go-kratos/kratos/v2/encoding/form"

	"github.com/go-playground/form/v4"
	"google.golang.org/protobuf/proto"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

const (
	// Name is form codec name
	Name = "x-www-form-urlencoded"
	// Null value string
	nullStr = "null"
)

// Codec is used to encode/decode HTML form values as used in GET request URL
// query parameters or POST request bodies.
type Codec struct {
	encoder *form.Encoder
	decoder *form.Decoder
}

// NewFormCodec will create a codec used to encode/decode HTML form values as
// used in GET request URL query parameters or POST request bodies.
func NewFormCodec() *Codec {
	return &Codec{
		encoder: form.NewEncoder(),
		decoder: form.NewDecoder(),
	}
}

func (c Codec) Marshal(v interface{}) ([]byte, error) {
	var (
		vs  url.Values
		err error
	)

	if m, ok := v.(proto.Message); ok {
		vs, err = kform.EncodeValues(m)
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

func (c Codec) Unmarshal(data []byte, v interface{}) error {
	vs, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}

		rv = rv.Elem()
	}

	if m, ok := v.(proto.Message); ok {
		return kform.DecodeValues(m, vs)
	} else if m, ok := reflect.Indirect(reflect.ValueOf(v)).Interface().(proto.Message); ok {
		return kform.DecodeValues(m, vs)
	}

	return c.decoder.Decode(v, vs)
}

// NewDecoder returns a Decoder which reads byte sequence from "r".
func (p Codec) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v interface{}) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		return p.Unmarshal(b, v)
	})
}

// NewEncoder returns an Encoder which writes bytes sequence into "w".
func (p Codec) NewEncoder(w io.Writer) runtime.Encoder {
	return runtime.EncoderFunc(func(v interface{}) error {
		b, err := p.Marshal(v)
		if err != nil {
			return err
		}

		_, err = w.Write(b)

		return err
	})
}

func (Codec) Name() string {
	return Name
}

// ContentType returns the Content-Type which this marshaler is responsible for.
// The parameter describes the type which is being marshalled, which can sometimes
// affect the content type returned.
func (p *Codec) ContentType(v interface{}) string {
	return p.Name()
}
