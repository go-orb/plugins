// Package file is the file source for orb/config.
package file

import (
	"net/url"
	"os"
	"path/filepath"

	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/config/source"
	"go-micro.dev/v5/log"
)

func init() {
	if err := source.Plugins.Add(New()); err != nil {
		panic(err)
	}
}

// Source is the file source for config.
type Source struct{}

// New creates a new file source for config.
func New() source.Source {
	return &Source{}
}

// Schemes returns the supported schemes for this source.
func (s *Source) Schemes() []string {
	return []string{"", "file"}
}

// PrependSections returns whetever this source needs sections to be prepended.
func (s *Source) PrependSections() bool {
	return false
}

// String returns the name of the source.
func (s *Source) String() string {
	return "file"
}

func (s *Source) Read(u *url.URL) source.Data {
	result := source.Data{
		Data: make(map[string]any),
	}

	path := u.Path
	marshalers := codecs.Plugins.All()

	if _, err := os.Stat(path); err == nil {
		result.Error = config.ErrFileNotFound
		return result
	}

	// Get the marshaler from the extension.
	var (
		pathExt = filepath.Ext(path)
		decoder codecs.Marshaler
	)

	for _, m := range marshalers {
		for _, ext := range m.Exts() {
			if pathExt == ext {
				decoder = m
				break
			}
		}
	}

	if decoder == nil {
		result.Error = config.ErrCodecNotFound
		return result
	}

	result.Marshaler = decoder

	fh, err := os.Open(filepath.Clean(path))
	if err != nil {
		result.Error = err
		return result
	}

	defer func() {
		if err := fh.Close(); err != nil {
			log.Error("failed to close config file", err)
		}
	}()

	if err := decoder.NewDecoder(fh).Decode(&result.Data); err != nil {
		result.Error = err
		return result
	}

	return result
}
