// Package http provides the http source for the config.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config/source"
	"github.com/go-orb/go-orb/log"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, myURL.String(), nil)
	if err != nil {
		result.Error = err

		return result
	}

	resp, err := http.DefaultClient.Do(req)
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
