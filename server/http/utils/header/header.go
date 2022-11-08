// Package header implements header manipulation utilities
package header

import (
	"mime"
	"strings"

	"go-micro.dev/v5/codecs"

	"github.com/go-micro/plugins/server/http/headers"
)

// GetContentType parses the content type from the header value.
func GetContentType(header string) (string, error) {
	ct, _, err := mime.ParseMediaType(header)
	if err != nil {
		return "", err
	}

	return ct, nil
}

// GetAcceptType parses the Accept header and checks against the available codecs
// to find a matching content type.
func GetAcceptType(c codecs.Map, acceptHeader string, contentType string) string {
	accept := contentType

	// If request used Form content type, return JSON instead of form.
	if accept == headers.FormContentType {
		accept = headers.JSONContentType
	}

	// If explicitly asked for a specific content type, use that
	acceptSlice := strings.Split(acceptHeader, ",")
	for _, acceptType := range acceptSlice {
		ct, _, err := mime.ParseMediaType(acceptType)
		if err != nil {
			continue
		}

		// Check if we have a codec for the content type
		if _, ok := c[ct]; ok {
			accept = ct
		}
	}

	return accept
}
