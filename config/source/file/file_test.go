package file

import (
	"encoding/base64"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	// Import codec plugins to register them.
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMarshaler(t *testing.T) {
	// Setup
	s := &Source{}

	// Test cases
	tests := []struct {
		name         string
		path         string
		expectingNil bool
	}{
		{
			name:         "JSON extension",
			path:         "config.json",
			expectingNil: false,
		},
		{
			name:         "YAML extension",
			path:         "config.yaml",
			expectingNil: false,
		},
		{
			name:         "Unknown extension",
			path:         "config.unknown",
			expectingNil: true,
		},
		{
			name:         "No extension",
			path:         "config",
			expectingNil: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaler := s.getMarshaler(tt.path)

			if tt.expectingNil {
				assert.Nil(t, marshaler, "Expected nil marshaler for %s", tt.path)
			} else {
				assert.NotNil(t, marshaler, "Expected non-nil marshaler for %s", tt.path)

				// If not nil, verify that the marshaler supports the expected extension
				if marshaler != nil {
					ext := filepath.Ext(tt.path)
					exts := marshaler.Exts()
					found := false
					for _, supportedExt := range exts {
						if ext == supportedExt {
							found = true
							break
						}
					}
					assert.True(t, found, "Marshaler doesn't support expected extension %s", ext)
				}
			}
		})
	}
}

func TestReadFromBase64(t *testing.T) {
	// Setup
	s := &Source{}

	// Test cases
	tests := []struct {
		name       string
		path       string
		b64Content string
		wantErr    bool
		wantData   map[string]interface{}
	}{
		{
			name:       "Valid JSON data",
			path:       "test.json",
			b64Content: base64.URLEncoding.EncodeToString([]byte(`{"foo":"bar"}`)),
			wantErr:    false,
			wantData: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			name:       "Invalid base64 data",
			path:       "test.json",
			b64Content: "invalid-base64",
			wantErr:    true,
		},
		{
			name:       "Invalid JSON data",
			path:       "test.json",
			b64Content: base64.URLEncoding.EncodeToString([]byte(`{invalid-json}`)),
			wantErr:    true,
		},
		{
			name:       "Unknown extension",
			path:       "test.unknown",
			b64Content: base64.URLEncoding.EncodeToString([]byte(`{"foo":"bar"}`)),
			wantErr:    true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.readFromBase64(tt.path, tt.b64Content)

			if tt.wantErr {
				require.Error(t, result.Error, "Expected error for %s", tt.name)
			} else {
				require.NoError(t, result.Error, "Unexpected error: %v", result.Error)
				require.Equal(t, tt.wantData, result.Data, "Data does not match expected values")
				require.NotNil(t, result.Marshaler, "Marshaler should not be nil")
			}
		})
	}
}

func TestReadFromFile(t *testing.T) {
	// Setup.
	s := &Source{}

	// Create temporary test files.
	tmpDir := t.TempDir()

	// Valid JSON file.
	jsonFile := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(jsonFile, []byte(`{"name":"test","value":123}`), 0600)
	require.NoError(t, err)

	// Invalid JSON file.
	invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(invalidJSONFile, []byte(`{invalid-json}`), 0600)
	require.NoError(t, err)

	// Unknown extension file.
	unknownFile := filepath.Join(tmpDir, "config.unknown")
	err = os.WriteFile(unknownFile, []byte(`{"name":"test"}`), 0600)
	require.NoError(t, err)

	// Test cases.
	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantData map[string]interface{}
	}{
		{
			name:    "Valid JSON file",
			path:    jsonFile,
			wantErr: false,
			wantData: map[string]interface{}{
				"name":  "test",
				"value": float64(123), // JSON numbers are unmarshaled as float64
			},
		},
		{
			name:    "Non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.json"),
			wantErr: true,
		},
		{
			name:    "Invalid JSON file",
			path:    invalidJSONFile,
			wantErr: true,
		},
		{
			name:    "Unknown extension file",
			path:    unknownFile,
			wantErr: true,
		},
	}

	// Run tests.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.readFromFile(tt.path)

			if tt.wantErr {
				require.Error(t, result.Error, "Expected error for %s", tt.name)
			} else {
				require.NoError(t, result.Error, "Unexpected error: %v", result.Error)
				require.Equal(t, tt.wantData, result.Data, "Data does not match expected values")
				require.NotNil(t, result.Marshaler, "Marshaler should not be nil")
			}
		})
	}
}

func TestRead(t *testing.T) {
	// Setup.
	s := &Source{}

	// Create a temporary test file.
	tmpDir := t.TempDir()

	jsonFile := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(jsonFile, []byte(`{"name":"file-test"}`), 0600)
	require.NoError(t, err)

	// Test cases.
	tests := []struct {
		name     string
		url      string
		wantErr  bool
		wantData map[string]interface{}
	}{
		{
			name:    "Valid file URL",
			url:     "file://" + jsonFile,
			wantErr: false,
			wantData: map[string]interface{}{
				"name": "file-test",
			},
		},
		{
			name:    "Valid base64 URL",
			url:     "file:///memory.json?base64=" + base64.URLEncoding.EncodeToString([]byte(`{"name":"base64-test"}`)),
			wantErr: false,
			wantData: map[string]interface{}{
				"name": "base64-test",
			},
		},
		{
			name:    "Invalid file URL",
			url:     "file:///non-existent-file.json",
			wantErr: true,
		},
		{
			name:    "Invalid base64 URL",
			url:     "file:///memory.json?base64=invalid-base64",
			wantErr: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			require.NoError(t, err)

			result := s.Read(u)

			if tt.wantErr {
				require.Error(t, result.Error, "Expected error for %s", tt.name)
			} else {
				require.NoError(t, result.Error, "Unexpected error: %v", result.Error)
				require.Equal(t, tt.wantData, result.Data, "Data does not match expected values")
				require.NotNil(t, result.Marshaler, "Marshaler should not be nil")
			}
		})
	}
}
