// Package proto ...
package proto

// Download Google proto HTTP annotation libs
//nolint:lll
//go:generate wget -q -O google/api/annotations.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
//go:generate wget -q -O google/api/http.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto

// Generate proto files
//nolint:lll
//go:generate protoc -I . --go-grpc_out=paths=source_relative:. --go-micro-http_out=paths=source_relative:. --go_out=paths=source_relative:. ./echo.proto
