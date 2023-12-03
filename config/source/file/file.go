// Package file is the file source for orb/config.
package file

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source"
	"github.com/go-orb/go-orb/log"
	"github.com/google/uuid"
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

	path := u.Host + u.Path

	if _, err := os.Stat(path); err != nil {
		result.Error = fmt.Errorf("%w: %w", config.ErrFileNotFound, err)
		return result
	}

	// Get the marshaler from the extension.
	var (
		pathExt = filepath.Ext(path)
		decoder codecs.Marshaler
	)

	codecs.Plugins.Range(func(_ string, m codecs.Marshaler) bool {
		for _, ext := range m.Exts() {
			if pathExt == ext {
				decoder = m
				return false
			}
		}

		return true
	})

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

// TempFile will take a byte sequence and write it to a temporary file. This
// is useful if you want to parse a config from memory.
//
// If at any point an erro occurs, it will panic. It does not return the error
// as the probability of one occurring is small, and now you can use it directly
// within your config array definition.
//
// Example:
//
//	configSource := []config.Source{"config.yaml", file.TempFile(myConfig, "yaml")}
func TempFile(data []byte, filetype string) *url.URL {
	dir := os.TempDir()
	filePath := path.Join(dir, "go-micro-config-"+uuid.NewString()+"."+filetype)

	file, err := os.OpenFile(path.Clean(filePath), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(fmt.Errorf("failed to create temporary config file '%s': %w", filePath, err))
	}

	_, err = file.Write(data)
	if err != nil {
		panic(fmt.Errorf("failed to write temporary config file '%s': %w", filePath, err))
	}

	file.Close() //nolint:errcheck,gosec

	u := "file://" + filePath

	url, err := url.Parse(u)
	if err != nil {
		panic(fmt.Errorf("failed to parse temporary config file as url '%s': %w", u, err))
	}

	return url
}
