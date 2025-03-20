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

// String returns the name of the source.
func (s *Source) String() string {
	return "file"
}

func (s *Source) Read(u *url.URL) (map[string]any, error) {
	// Handle base64-encoded content from URL parameter.
	if b64Param := u.Query().Get("base64"); b64Param != "" {
		return s.readFromBase64(u.Path, b64Param)
	}

	// Handle regular file path.
	return s.readFromFile(u.Host + u.Path)
}

// readFromBase64 reads config from a base64-encoded string.
func (s *Source) readFromBase64(path, b64Content string) (map[string]any, error) {
	result := map[string]any{}

	// Get marshaler for file extension.
	marshaler := s.getMarshaler(path)
	if marshaler == nil {
		return nil, config.ErrCodecNotFound
	}

	// Decode base64 content.
	data, err := base64.URLEncoding.DecodeString(b64Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 config: %w", err)
	}

	// Unmarshal the data.
	if err := marshaler.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base64 config: %w", err)
	}

	return result, nil
}

// readFromFile reads config from a filesystem path.
func (s *Source) readFromFile(path string) (map[string]any, error) {
	result := map[string]any{}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("%w: %w", config.ErrFileNotFound, err)
	}

	// Get marshaler for file extension.
	marshaler := s.getMarshaler(path)
	if marshaler == nil {
		return nil, config.ErrCodecNotFound
	}

	// Open and read the file.
	fh, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	defer func() {
		if err := fh.Close(); err != nil {
			log.Error("failed to close config file", "path", path, "error", err)
		}
	}()

	// Decode the file content.
	if err := marshaler.NewDecoder(fh).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return result, nil
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
