module github.com/go-orb/plugins/registry/nats

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230709084536-48ca79fd6450
	github.com/go-orb/plugins/log/text v0.0.0
	github.com/go-orb/plugins/registry/tests v0.0.0
	github.com/nats-io/nats-server/v2 v2.9.19
	github.com/nats-io/nats.go v1.27.1
	golang.org/x/exp v0.0.0-20230711153332-06a737ee72cb
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/jwt/v2 v2.4.1 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/go-orb/plugins/log/text => ../../log/text
	github.com/go-orb/plugins/registry/tests => ../tests
)
