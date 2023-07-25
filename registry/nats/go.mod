module github.com/go-orb/plugins/registry/nats

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230725190534-6e856aec238f
	github.com/go-orb/plugins/registry/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/google/uuid v1.3.0
	github.com/nats-io/nats.go v1.28.0
	github.com/pkg/errors v0.9.1
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/nats-io/nats-server/v2 v2.9.19 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/registry/tests => ../tests
