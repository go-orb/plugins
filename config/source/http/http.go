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

// String returns the name of the source.
func (s *Source) String() string {
	return "http"
}

func (s *Source) Read(myURL *url.URL) (map[string]any, error) {
	path := myURL.Path

	var decoder codecs.Marshaler

	// Get the marshaler from the extension.
	pathExt := filepath.Ext(path)

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
		return nil, codecs.ErrNoFileMarshaler
	}

	// Download the file
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, myURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("Error while closing the body", "url", myURL.String(), "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			log.Error("Error while closing the body", "url", myURL.String(), "error", err)
		}

		return nil, fmt.Errorf("bad response status code '%d', status text: %s", resp.StatusCode, resp.Status)
	}

	result := map[string]any{}

	if err := decoder.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
