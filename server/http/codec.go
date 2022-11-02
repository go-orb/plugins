package http

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-micro/plugins/server/http/headers"
	"github.com/go-micro/plugins/server/http/utils/header"
)

// TODO: decode body now also does content type setting, maybe sepearte that out

// Decode body takes the request body and decodes it into the proto type.
func (s *Server) decodeBody(w http.ResponseWriter, request *http.Request, in any) (string, error) {
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
			s.logger.Debug("Request failed while parsing content type: %v", err)
			return "", err
		}

		// Gzip decode if needed
		eHeader := request.Header.Get(headers.ConentEncoding)
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
	w.Header().Set(headers.ContentType, accept)

	codec, ok := s.codecs[contentType]
	if !ok {
		s.logger.Debug("Request failed, codec not found for contet type: %v", contentType)
		return "", ErrContentTypeNotSupported
	}

	if err := codec.NewDecoder(body).Decode(in); err != nil {
		s.logger.Debug("Request failed, failed to decode body: %v", err)
		return "", fmt.Errorf("decode content type '%s': %w", err)
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
		s.logger.Debug("Request failed, codec for content type not available: %v", contentType)
		return ErrContentTypeNotSupported
	}

	nw := w.(io.Writer)

	eHeader := r.Header.Get(headers.AcceptEncoding)
	if s.Config.EnableGzip && strings.Contains(eHeader, headers.GzipContentEncoding) {
		w.Header().Set(headers.ConentEncoding, headers.GzipContentEncoding)
		nw = gzip.NewWriter(w)
		defer nw.(io.Closer).Close()
	}

	if err := codec.NewEncoder(nw).Encode(v); err != nil {
		s.logger.Debug("Request failed, failed to encode response: %v", err)
		return err
	}

	return nil
}
