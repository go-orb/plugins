module github.com/go-orb/plugins/codecs/jsonpb

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230725195416-0c865f140784
	github.com/go-orb/plugins/codecs/proto v0.0.0-20230713091520-67e7b5a34489
	github.com/google/go-cmp v0.5.9
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.1
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230724170836-66ad5b6ff146 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230724170836-66ad5b6ff146 // indirect
	google.golang.org/grpc v1.56.2 // indirect
)

replace github.com/go-orb/plugins/codecs/proto => ../proto
