module github.com/go-orb/plugins/codecs/jsonpb

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231126231708-592c8d8d05c6
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
)

replace github.com/go-orb/plugins/codecs/proto => ../proto
