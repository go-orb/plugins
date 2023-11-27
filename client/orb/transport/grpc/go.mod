module github.com/go-orb/plugins/client/orb/transport/grpc

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231127002523-4909ba192408
	github.com/go-orb/plugins/client/orb v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/client/tests v0.0.0-00010101000000-000000000000
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/codecs/proto v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/config/source/file v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/log/slog v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/registry/consul v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/registry/mdns v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/server/http v0.0.0-20231126232626-f2cd47f2724d
	github.com/stretchr/testify v1.8.4
	google.golang.org/grpc v1.59.0
)

require (
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/go-chi/chi v1.5.5 // indirect
	github.com/go-orb/plugins/registry/regutil v0.0.0-20231126232626-f2cd47f2724d // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20231101202521-4ca4178f5c7a // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/hashicorp/consul/api v1.26.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.57 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/onsi/ginkgo/v2 v2.13.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.4.1 // indirect
	github.com/quic-go/quic-go v0.40.0 // indirect
	go.uber.org/mock v0.3.0 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.15.0 // indirect
	google.golang.org/genproto v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/client/orb => ../..

replace github.com/go-orb/plugins/client/tests => ../../../tests

replace github.com/go-orb/plugins/codecs/jsonpb => ../../../../codecs/jsonpb

replace github.com/go-orb/plugins/codecs/proto => ../../../../codecs/proto

replace github.com/go-orb/plugins/codecs/yaml => ../../../../codecs/yaml

replace github.com/go-orb/plugins/config/source/file => ../../../../config/source/file

replace github.com/go-orb/plugins/log/slog => ../../../../log/slog

replace github.com/go-orb/plugins/registry/mdns => ../../../../registry/mdns

replace github.com/go-orb/plugins/server/hertz => ../../../../server/hertz

replace github.com/go-orb/plugins/server/http => ../../../../server/http

replace github.com/go-orb/plugins/registry/consul => ../../../../registry/consul

replace github.com/go-orb/plugins/registry/regutil => ../../../../registry/regutil
