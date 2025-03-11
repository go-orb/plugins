// Package file is the file source for orb/config.
package file

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source"
	"github.com/go-orb/go-orb/log"
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
	// Handle base64-encoded content from URL parameter.
	if b64Param := u.Query().Get("base64"); b64Param != "" {
		return s.readFromBase64(u.Path, b64Param)
	}

	// Handle regular file path.
	return s.readFromFile(u.Host + u.Path)
}

// readFromBase64 reads config from a base64-encoded string.
func (s *Source) readFromBase64(path, b64Content string) source.Data {
	result := source.Data{
		Data: make(map[string]any),
	}

	// Get marshaler for file extension.
	marshaler := s.getMarshaler(path)
	if marshaler == nil {
		result.Error = config.ErrCodecNotFound
		return result
	}

	result.Marshaler = marshaler

	// Decode base64 content.
	data, err := base64.URLEncoding.DecodeString(b64Content)
	if err != nil {
		result.Error = fmt.Errorf("failed to decode base64 config: %w", err)
		return result
	}

	// Unmarshal the data.
	if err := marshaler.Unmarshal(data, &result.Data); err != nil {
		result.Error = fmt.Errorf("failed to unmarshal base64 config: %w", err)
		return result
	}

	return result
}

// readFromFile reads config from a filesystem path.
func (s *Source) readFromFile(path string) source.Data {
	result := source.Data{
		Data: make(map[string]any),
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		result.Error = fmt.Errorf("%w: %w", config.ErrFileNotFound, err)
		return result
	}

	// Get marshaler for file extension.
	marshaler := s.getMarshaler(path)
	if marshaler == nil {
		result.Error = config.ErrCodecNotFound
		return result
	}

	result.Marshaler = marshaler

	// Open and read the file.
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

	// Decode the file content.
	if err := marshaler.NewDecoder(fh).Decode(&result.Data); err != nil {
		result.Error = err
		return result
	}

	return result
}

// getMarshaler finds an appropriate marshaler for the given file path based on extension.
func (s *Source) getMarshaler(path string) codecs.Marshaler {
	ext := filepath.Ext(path)

	var result codecs.Marshaler

	codecs.Plugins.Range(func(_ string, m codecs.Marshaler) bool {
		for _, supportedExt := range m.Exts() {
			if ext == supportedExt {
				result = m
				return false // Stop iterating
			}
		}

		return true // Continue iterating
	})

	return result
}
