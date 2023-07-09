module github.com/go-orb/plugins/registry/nats

go 1.20

require (
	github.com/go-orb/go-orb v0.0.1
	github.com/nats-io/nats.go v1.27.1
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/nats-io/nats-server/v2 v2.9.7 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/sys v0.10.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/go-orb/plugins/log/text => ../../log/text
