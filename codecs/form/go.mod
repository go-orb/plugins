module github.com/go-micro/plugins/codecs/form

go 1.19

replace github.com/go-micro/plugins/codecs/proto => ../proto

replace go-micro.dev/v5 => ../../../orb

require (
	github.com/go-playground/form/v4 v4.2.0
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	google.golang.org/grpc v1.50.1
)
