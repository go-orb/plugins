package json

import "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

type JSON struct {
	runtime.JSONPb

	// contentType is used to overwrite the content type used by the encoder.
	// This is useful when one encoder encodes for multiple content types,
	// and you have to create a sperate instance for each one.
	contentType string
}

func NewCodec(contentType string) *JSON {
	return &JSON{contentType: contentType}
}

func (j *JSON) ContentType(_ any) string {
	return j.contentType
}
