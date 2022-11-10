module github.com/go-micro/plugins/codecs/proto

go 1.19

require (
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.28.1
)

require github.com/google/go-cmp v0.5.9 // indirect

replace go-micro.dev/v5 => ../../../orb
