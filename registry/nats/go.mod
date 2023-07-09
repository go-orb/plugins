module github.com/go-orb/plugins/registry/nats

go 1.20

require (
<<<<<<< Updated upstream
	github.com/nats-io/nats.go v1.19.0
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
=======
	github.com/go-orb/go-orb v0.0.1
	github.com/go-orb/plugins/log/text v0.0.0-00010101000000-000000000000
	github.com/nats-io/nats-server/v2 v2.9.7
	github.com/nats-io/nats.go v1.27.1
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df
>>>>>>> Stashed changes
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/jwt/v2 v2.3.0 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/go-orb/plugins/log/text => ../../log/text
