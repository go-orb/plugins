// Package file provides the file source for the config. It allows you to read in
// files from disk with any extension for which a codec plugin has been loaded.
package file

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/config/source"
	"go-micro.dev/v5/log"
)

func init() {
	if err := source.Plugins.Add(New()); err != nil {
		panic(err)
	}
}

// Source is the http source for config.
type Source struct{}

// New creates a new http source for config.
func New() source.Source {
	return &Source{}
}

// Schemes returns the supported schemes for this source.
func (s *Source) Schemes() []string {
	return []string{"http", "https"}
}

// PrependSections returns whetever this source needs sections to be prepended.
func (s *Source) PrependSections() bool {
	return false
}

// String returns the name of the source.
func (s *Source) String() string {
	return "http"
}

func (s *Source) Read(myURL *url.URL) source.Data {
	result := source.Data{
		Data: make(map[string]any),
	}

	path := myURL.Path
	marshalers := codecs.Plugins.All()

	var decoder codecs.Marshaler

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

	if decoder == nil {
		result.Error = codecs.ErrNoFileMarshaler
		return result
	}

	result.Marshaler = decoder

	// Download the file
	resp, err := http.Get(myURL.String())
	if err != nil {
		result.Error = err
		return result
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("Error while closing the body", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			log.Error("Error while closing the body", err)
		}
		result.Error = fmt.Errorf("bad response status code '%d', status text: %s", resp.StatusCode, resp.Status)
		return result
	}

	if err := decoder.NewDecoder(resp.Body).Decode(&result.Data); err != nil {
		result.Error = err
		return result
	}

	return result
}
