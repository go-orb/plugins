module github.com/go-micro/plugins/codecs/jsonpb

go 1.19

require (
	github.com/go-micro/plugins/codecs/proto v0.0.0-00010101000000-000000000000
	github.com/google/go-cmp v0.5.9
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.13.0
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c // indirect
	google.golang.org/grpc v1.50.1 // indirect
)

replace github.com/go-micro/plugins/codecs/proto => ../proto
