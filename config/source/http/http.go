// Package file is the file source for orb/config.
package file

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"go-micro.dev/v5/codecs"
	"go-micro.dev/v5/config/source"
)

func init() {
	err := source.Plugins.Add(New())
	if err != nil {
		panic(err)
	}
}

type Source struct{}

func New() source.Source {
	return &Source{}
}

func (s *Source) Schemes() []string {
	return []string{"http", "https"}
}

func (s *Source) PrependSections() bool {
	return false
}

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

	// No marshaler found
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("bad response status code '%d', status text: %s", resp.StatusCode, resp.Status)
		return result
	}

	// Decode
	if err := decoder.NewDecoder(resp.Body).Decode(&result.Data); err != nil {
		result.Error = err
		return result
	}

	return result
}
