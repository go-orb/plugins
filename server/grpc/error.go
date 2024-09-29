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
var httpStatusToCode = map[int]codes.Code{
	http.StatusOK:                  codes.OK,
	499:                            codes.Canceled,
	http.StatusBadRequest:          codes.InvalidArgument,
	http.StatusInternalServerError: codes.Internal,
	http.StatusGatewayTimeout:      codes.DeadlineExceeded,
	http.StatusNotFound:            codes.NotFound,
	http.StatusConflict:            codes.AlreadyExists,
	http.StatusForbidden:           codes.PermissionDenied,
	http.StatusUnauthorized:        codes.Unauthenticated,
	http.StatusTooManyRequests:     codes.ResourceExhausted,
	http.StatusNotImplemented:      codes.Unimplemented,
	http.StatusServiceUnavailable:  codes.Unavailable,
}

// HTTPStatusToCode maps HTTP status codes to gRPC codes.
//
// Falls back to codes.Internal if the code is unrecognized.
func HTTPStatusToCode(code int) codes.Code {
	if status, ok := httpStatusToCode[code]; ok {
		return status
	}

	return codes.Internal
}
