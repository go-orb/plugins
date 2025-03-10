// Package proto implements a marshaler interface for protocol buffers.
package proto

import (
	"errors"
	"io"

	"github.com/go-orb/go-orb/codecs"
	"google.golang.org/protobuf/proto"
)

var _ codecs.Marshaler = (*Proto)(nil)

// Proto is a proto marshaler that can encode and decode protocol buffers.
type Proto struct{}

func init() {
	codecs.Register("proto", &Proto{})
}

// ContentTypes returns the list of content types this marshaller is able to
// handle.
func (p *Proto) ContentTypes() []string {
	return []string{
		"application/octet-stream", // TODO(jochumdev): Remove me?
		"application/proto",
		"application/x-protobuf",
		"application/x-proto",
		"application/x-protobuf",
	}
}

// Encode encodes "value" into Proto.
func (*Proto) Marshal(value interface{}) ([]byte, error) {
	message, ok := value.(proto.Message)
	if !ok {
		return nil, errors.New("unable to marshal non proto field")
	}

	return proto.Marshal(message)
}

// Unmarshal decodes proto "data" into "value".
func (*Proto) Unmarshal(data []byte, value interface{}) error {
	message, ok := value.(proto.Message)
	if !ok {
		return errors.New("unable to unmarshal non proto field")
	}

	return proto.Unmarshal(data, message)
}

// NewDecoder returns a Decoder which reads proto stream from "reader".
func (p *Proto) NewDecoder(reader io.Reader) codecs.Decoder {
	return codecs.DecoderFunc(func(value interface{}) error {
		buffer, err := io.ReadAll(reader)
		if err != nil {
			return err
		}

		return p.Unmarshal(buffer, value)
	})
}

// NewEncoder returns an Encoder which writes proto stream into "writer".
func (p *Proto) NewEncoder(writer io.Writer) codecs.Encoder {
	return codecs.EncoderFunc(func(value interface{}) error {
		buffer, err := p.Marshal(value)
		if err != nil {
			return err
		}

		_, err = writer.Write(buffer)
		if err != nil {
			return err
		}

		return nil
	})
}

// Marshals returns if this is able to encode the given type.
func (p *Proto) Marshals(v any) bool {
	_, ok := v.(proto.Message)

	return ok
}

// Unmarshals returns if this is able to decode the given type.
func (p *Proto) Unmarshals(v any) bool {
	return p.Marshals(v)
}

// Name returns the codec name.
func (p *Proto) Name() string {
	return "proto"
}

// Exts is a list of file extensions this marshaler supports.
func (p *Proto) Exts() []string {
	return []string{".proto"}
}
