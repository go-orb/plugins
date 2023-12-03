module github.com/go-orb/plugins/codecs/jsonpb

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231203061431-2cf52a164da0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
)

replace github.com/go-orb/plugins/codecs/proto => ../proto
