module github.com/go-orb/plugins/client/orb

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230805173903-ba3da7c24b9d
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
	google.golang.org/grpc v1.58.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230726155614-23370e0ffb3e // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/go-orb/go-orb => ../../../go-orb
