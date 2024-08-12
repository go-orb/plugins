// Package proto ...
package proto

//go:generate protoc -I . --go-orb_out=paths=source_relative:. --go-orb_opt="supported_servers=drpc;grpc;http;hertz" ./echo/echo.proto
