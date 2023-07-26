// Package grpc is the grpc transport for plugins/client/orb.
package grpc

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// codeToHTTPStatus maps gRPC codes to HTTP statuses.
// Based on https://cloud.google.com/apis/design/errors
//
// Copied from: https://github.com/luci/luci-go/blob/main/grpc/grpcutil/errors.go#L118 (Apache 2.0).
//
//nolint:gochecknoglobals
var codeMapToHTTPStatus = map[codes.Code]int{
	codes.OK:                 http.StatusOK,
	codes.Canceled:           499,
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.DataLoss:           http.StatusInternalServerError,
	codes.Internal:           http.StatusInternalServerError,
	codes.Unknown:            http.StatusInternalServerError,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.FailedPrecondition: http.StatusBadRequest,
	codes.OutOfRange:         http.StatusBadRequest,
	codes.Unimplemented:      http.StatusNotImplemented,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.Aborted:            http.StatusConflict,
}

// CodeToHTTPStatus maps gRPC codes to HTTP status codes.
//
// Falls back to http.StatusInternalServerError if the code is unrecognized.
func CodeToHTTPStatus(code codes.Code) int {
	if status, ok := codeMapToHTTPStatus[code]; ok {
		return status
	}

	return http.StatusInternalServerError
}
