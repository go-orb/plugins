package codec

import (
	"errors"

	"github.com/go-micro/plugins/server/http/codec/form"
	"github.com/go-micro/plugins/server/http/codec/json"
	"github.com/go-micro/plugins/server/http/codec/proto"

	"github.com/google/wire"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// TODO: maybe codec shouldn't be a provider, but global container, but then we need to add a diable codec flag
// TODO: Cleanup this file, seperate into sub modules

var (
	CodecRegistry   = make(Codecs)
	ErrNotProtoType = errors.New("provided interface is not of type proto.Message")

	DefaultCodecSet = wire.NewSet(ProvideCodecJSON, ProvideCodecProto, ProvideCodecForm)
)

type Codec interface {
	runtime.Marshaler
}

type Codecs map[string]Codec

type CodecRegistration struct {
	// One or more content types for which the codec is responsible
	ContentTypes []string
	Codec        Codec
}

func ProvideCodecJSON() Codec {
	return json.NewCodec("application/json")
}

func ProvideCodecProto() []Codec {
	return []Codec{
		proto.NewCodec("application/octet-stream"),
		proto.NewCodec("application/proto"),
		proto.NewCodec("application/x-proto"),
		proto.NewCodec("application/protobuf"),
		proto.NewCodec("application/x-protobuf"),
	}
}

func ProvideCodecForm() Codec {
	return form.NewFormCodec()
}

func ProvideDefaultCodecs() []Codec {
	c := make([]Codec, 10)
	c = append(c, ProvideCodecForm())
	c = append(c, ProvideCodecJSON())
	c = append(c, ProvideCodecProto()...)

	return c
}

func ProvideCodecs(codecs ...Codec) map[string]Codec {
	m := make(map[string]Codec, len(codecs))

	for _, c := range codecs {
		if c == nil {
			continue
		}

		m[c.ContentType(nil)] = c
	}

	return m
}
