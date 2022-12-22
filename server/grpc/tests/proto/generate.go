// Package proto ...
package proto

// Generate proto files
//go:generate protoc -I . --go-grpc_out=paths=source_relative:.  --go_out=paths=source_relative:. ./echo.proto
