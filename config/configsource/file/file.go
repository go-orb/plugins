package file

import (
	"net/url"

	"jochum.dev/orb/orb/config/configsource"
	"jochum.dev/orb/orb/util/marshaler"
)

const Name = "file"

func init() {
	err := configsource.Plugins.Add(Name, New)
	if err != nil {
		panic(err)
	}
}

type Source struct{}

func New() configsource.ConfigSource {
	return &Source{}
}

func (s *Source) String() string {
	return Name
}

func (s *Source) Init() error {
	return nil
}

func (s *Source) Read(u url.URL) (map[string]any, error) {
	if u.Scheme != Name {
		return map[string]any{}, configsource.ErrUnknownScheme
	}

	path := u.Path

	return map[string]any{}, nil
}

func (s *Source) Write(u url.URL, data map[string]any) error {
	return nil
}

func marshalers() []marshaler.Marshaler {
	result := []marshaler.Marshaler{}
	for _, m := range marshaler.Plugins.All() {
		result = append(result, m())
	}

	return result
}
