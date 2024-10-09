package http

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"log/slog"

	"github.com/go-orb/plugins/server/http/headers"
	"github.com/go-orb/plugins/server/http/utils/header"
)

// TODO(davincible): decode body now also does content type setting, maybe separate that out

// Decode body takes the request body and decodes it into the proto type.
func (s *Server) decodeBody(resp http.ResponseWriter, request *http.Request, msg any) (string, error) {
	var (
		body        io.Reader
		contentType string
		err         error
	)

	ctHeader := request.Header.Get(headers.ContentType)

	// Parse params from query on GET request, or if no content type, on other read body
	switch {
	case request.Method == http.MethodGet || len(ctHeader) == 0:
		query := request.URL.Query().Encode()
		body = bytes.NewBufferString(query)

		contentType = headers.FormContentType
	default:
		contentType, err = header.GetContentType(ctHeader)
		if err != nil {
			s.logger.Debug("Request failed while parsing content type: "+err.Error(), slog.String("Content-Type", ctHeader))
			return "", err
		}

		// Gzip decode if needed
		eHeader := request.Header.Get(headers.ContentEncoding)
		if strings.Contains(eHeader, headers.GzipContentEncoding) {
			body, err = gzip.NewReader(request.Body)
			if err != nil {
				return "", err
			}
		} else {
			body = request.Body
		}
	}

	// Set response content type
	aHeader := request.Header.Get(headers.Accept)
	accept := header.GetAcceptType(s.codecs, aHeader, contentType)
	resp.Header().Set(headers.ContentType, accept)

	codec, ok := s.codecs[contentType]
	if !ok {
		s.logger.Debug("Request failed, codec not found for content type: " + contentType)
		return "", ErrContentTypeNotSupported
	}

	if err := codec.NewDecoder(body).Decode(msg); err != nil {
		s.logger.Debug("Request failed, failed to decode body", "error", err)
		return "", fmt.Errorf("decode content type '%s': %w", contentType, err)
	}

	return accept, nil
}

// encodeBody takes the return proto type and encodes it into the response.
func (s *Server) encodeBody(w http.ResponseWriter, r *http.Request, v any) error {
	contentType := w.Header().Get(headers.ContentType)
	if len(contentType) == 0 {
		contentType = headers.JSONContentType
	}

	codec, ok := s.codecs[contentType]
	if !ok {
		s.logger.Debug("Request failed, codec for content type not available", slog.String("Content-Type", contentType))
		return ErrContentTypeNotSupported
	}

	nw := w.(io.Writer) //nolint:errcheck

	// Gzip compress response if needed.
	aeHeader := r.Header.Get(headers.AcceptEncoding)
	reHeader := r.Header.Get(headers.ContentEncoding)
	gzipEnabled := s.config.Gzip || strings.Contains(reHeader, headers.GzipContentEncoding)

	if gzipEnabled && strings.Contains(aeHeader, headers.GzipContentEncoding) {
		w.Header().Set(headers.ContentEncoding, headers.GzipContentEncoding)

		nw = gzip.NewWriter(w)
		defer nw.(io.Closer).Close() //nolint:errcheck
	}

	if err := codec.NewEncoder(nw).Encode(v); err != nil {
		s.logger.Debug("Request failed, failed to encode response", "error", err)
		return err
	}

	return nil
}
