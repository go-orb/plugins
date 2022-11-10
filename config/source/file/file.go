// Package file is the file source for orb/config.
package file

import (
	"net/url"
	"os"
	"path/filepath"

	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/config/source"
)

func init() {
	if err := source.Plugins.Add(New()); err != nil {
		panic(err)
	}
}

type Source struct{}

func New() source.Source {
	return &Source{}
}

func (s *Source) Schemes() []string {
	return []string{"", "file"}
}

func (s *Source) PrependSections() bool {
	return false
}

func (s *Source) String() string {
	return "file"
}

func (s *Source) Read(u *url.URL) source.Data {
	result := source.Data{
		Data: make(map[string]any),
	}

	path := u.Path
	marshalers := codecs.Plugins.All()

	var decoder codecs.Marshaler
	if _, err2 := os.Stat(path); err2 != nil {
		// Guess the file extension
		for _, m := range marshalers {
			for _, ext := range m.Exts() {
				if _, err2 := os.Stat(path + ext); err2 == nil {
					decoder = m
					path += ext

					break
				}
			}

			if decoder != nil {
				break
			}
		}
	} else {
		// Get the marshaler from the extension.
		pathExt := filepath.Ext(path)
		for _, m := range marshalers {
			for _, ext := range m.Exts() {
				if pathExt == ext {
					decoder = m
					break
				}
			}

			if decoder != nil {
				break
			}
		}
	}

	if decoder == nil {
		result.Error = config.ErrNoSuchFile
		return result
	}

	result.Marshaler = decoder

	fh, err := os.Open(path)
	if err != nil {
		result.Error = err
		return result
	}
	defer fh.Close()

	if err := decoder.NewDecoder(fh).Decode(&result.Data); err != nil {
		result.Error = err
		return result
	}

	return result
}
