package proto

import "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

type Proto struct {
	runtime.ProtoMarshaller

	// contentType is used to overwrite the content type used by the encoder.
	// This is useful when one encoder encodes for multiple content types,
	// and you have to create a sperate instance for each one.
	contentType string
}

func (p *Proto) ContentType(_ any) string {
	return p.contentType
}

func NewCodec(contentType string) *Proto {
	return &Proto{contentType: contentType}
}
