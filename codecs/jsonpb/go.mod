module github.com/go-orb/plugins/codecs/jsonpb

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231127002523-4909ba192408
	google.golang.org/protobuf v1.31.0
)

require github.com/google/go-cmp v0.6.0 // indirect

replace github.com/go-orb/plugins/codecs/proto => ../proto
