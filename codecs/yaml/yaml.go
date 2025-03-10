// Package yaml implements a YAML Marshaler.
package yaml

import (
	"io"

	"github.com/go-orb/go-orb/codecs"

	"gopkg.in/yaml.v3"
)

var _ codecs.Marshaler = (*Yaml)(nil)

func init() {
	codecs.Register("yaml", &Yaml{})
}

// Yaml implements the codecs.Marshaler interface. It can be used to encode/decode
// yaml files, or web requests.
type Yaml struct{}

// Marshal encodes any pointer into yaml byte.
func (j *Yaml) Marshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

// Unmarshal decodes yaml bytes into object v.
// Param v should be a pointer type.
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

// Marshals returns if this is able to encode the given type.
func (j *Yaml) Marshals(v any) bool {
	switch v.(type) {
	case []string:
		return true
	case []byte:
		return true
	case []any:
		return true
	case map[string]any:
		return true
	case string:
		return true
	default:
		return false
	}
}

// Unmarshals returns if this is able to decode the given type.
func (j *Yaml) Unmarshals(v any) bool {
	return j.Marshals(v)
}

// ContentTypes returns the content types the marshaller can handle.
func (j *Yaml) ContentTypes() []string {
	return []string{
		"application/yaml",
	}
}

// Name returns the codec name.
func (j *Yaml) Name() string {
	return "yaml"
}

// Exts is a list of file extensions this marshaler supports.
func (j *Yaml) Exts() []string {
	return []string{".yaml", ".yml"}
}
