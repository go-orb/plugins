// Package file is the file source for orb/config.
package file

import (
	"net/url"
	"os"
	"path/filepath"

	"jochum.dev/orb/orb/config"
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

func New() configsource.Source {
	return &Source{}
}

func (s *Source) String() string {
	return Name
}

func (s *Source) Init() error {
	return nil
}

func (s *Source) Read(u url.URL) (map[string]any, error) {
	result := map[string]any{}
	if u.Scheme != Name {
		return result, config.ErrUnknownScheme
	}

	path := u.Path
	marshalers := marshaler.Marshalers()

	var decoder marshaler.Marshaler
	if _, err2 := os.Stat(path); err2 != nil {
		// Guess the file extension
		for _, m := range marshalers {
			if _, err2 := os.Stat(path + m.FileExtension()); err2 == nil {
				decoder = m
				path += m.FileExtension()
			}
		}
	} else {
		// Get the marshaler from the extension.
		ext := filepath.Ext(path)
		for _, m := range marshalers {
			if ext == m.FileExtension() {
				decoder = m
			}
		}
	}

	// No marshaler found
	if decoder == nil {
		return result, marshaler.ErrNoFileMarshaler
	}

	// Open the file
	fh, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer fh.Close()

	// Provide the file handle to the marshaler
	if err := decoder.Init(fh, nil); err != nil {
		return result, err
	}

	// Decode
	if err := decoder.DecodeSocket(&result); err != nil {
		return result, err
	}

	return result, nil
}

func (s *Source) Write(u url.URL, data map[string]any) error {
	return nil
}
