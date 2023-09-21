module github.com/go-orb/plugins/server/grpc

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230805173903-ba3da7c24b9d
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/config/source/file v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/log/slog v0.0.0-20230726151712-9ff06382f83b
	github.com/go-orb/plugins/registry/mdns v0.0.0-20230725003158-93bc3eff9bfa
	github.com/google/uuid v1.3.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/codecs/yaml => ../../codecs/yaml

replace github.com/go-orb/plugins/log/slog => ../../log/slog

replace github.com/go-orb/plugins/config/source/file => ../../config/source/file

replace github.com/go-orb/plugins/registry/mdns => ../../registry/mdns

replace github.com/go-orb/go-orb => ../../../go-orb
