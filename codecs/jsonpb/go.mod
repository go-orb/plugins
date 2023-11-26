module github.com/go-orb/plugins/codecs/jsonpb

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20231126093803-b366a8714a50
	github.com/go-orb/plugins/codecs/proto v0.0.0-20230713091520-67e7b5a34489
	github.com/google/go-cmp v0.6.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.1
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/grpc v1.59.0 // indirect
)

replace github.com/go-orb/plugins/codecs/proto => ../proto
