// Package proto ...
package proto

// Generate proto files
//go:generate protoc -I . --go-orb_out=paths=source_relative:. --go-orb_opt=supported_servers=drpc;grpc;http echo.proto
