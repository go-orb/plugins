// Package yaml implements a YAML Marshaler.
package yaml

import (
	"io"

	"go-micro.dev/v5/codecs"

	"gopkg.in/yaml.v3"
)

var _ codecs.Marshaler = (*Yaml)(nil)

func init() {
	if err := codecs.Plugins.Add("yaml", &Yaml{}); err != nil {
		panic(err)
	}
}

type Yaml struct{}

func (j *Yaml) Marshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func (j *Yaml) Unmarshal(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

// NewEncoder returns a new JSON/ProtocolBuffer encoder.
func (j *Yaml) NewEncoder(w io.Writer) codecs.Encoder {
	return yaml.NewEncoder(w)
}

// NewDecoder returns a new JSON/ProtocolBuffer decoder.
func (j *Yaml) NewDecoder(r io.Reader) codecs.Decoder {
	return yaml.NewDecoder(r)
}

// ContentTypes returns the content types the marshaller can handle.
func (j *Yaml) ContentTypes() []string {
	return []string{
		"application/yaml",
	}
}

// String returns the plugin implementation of the marshaler.
func (j *Yaml) String() string {
	return "yaml"
}

func (j *Yaml) Exts() []string {
	return []string{".yaml", ".yml"}
}
