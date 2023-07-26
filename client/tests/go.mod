module github.com/go-orb/plugins/client/tests

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230728000045-a99830943143
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20230727225210-cbbbd0c939a8
	github.com/go-orb/plugins/codecs/proto v0.0.0-20230727225210-cbbbd0c939a8
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20230727225210-cbbbd0c939a8
	github.com/go-orb/plugins/config/source/cli/urfave v0.0.0-20230728042620-269dc6079867
	github.com/go-orb/plugins/config/source/file v0.0.0-20230727225210-cbbbd0c939a8
	github.com/go-orb/plugins/log/slog v0.0.0-20230727225210-cbbbd0c939a8
	github.com/go-orb/plugins/registry/mdns v0.0.0-20230727225210-cbbbd0c939a8
	github.com/go-orb/plugins/server/grpc v0.0.0-20230728042620-269dc6079867
	github.com/go-orb/plugins/server/http v0.0.0-20230727225210-cbbbd0c939a8
	github.com/google/wire v0.5.0
	github.com/hashicorp/consul/sdk v0.14.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230801115018-d63ba01acd4b
	google.golang.org/genproto/googleapis/api v0.0.0-20230803162519-f966b187b2e5
	google.golang.org/grpc v1.57.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-chi/chi v1.5.4 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20230728192033-2ba5b33183c6 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/onsi/ginkgo/v2 v2.11.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.3.2 // indirect
	github.com/quic-go/quic-go v0.37.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/urfave/cli/v2 v2.25.7 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.13.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/tools v0.11.1 // indirect
	google.golang.org/genproto v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230803162519-f966b187b2e5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/codecs/jsonpb => ../../codecs/jsonpb

replace github.com/go-orb/plugins/codecs/proto => ../../codecs/proto

replace github.com/go-orb/plugins/codecs/yaml => ../../codecs/yaml

replace github.com/go-orb/plugins/config/source/file => ../../config/source/file

replace github.com/go-orb/plugins/server/http => ../../server/http

replace github.com/go-orb/plugins/registry/mdns => ../../registry/mdns

replace github.com/go-orb/plugins/log/slog => ../../log/slog

replace github.com/go-orb/plugins/config/source/cli/urfave => ../../config/source/cli/urfave

replace github.com/go-orb/plugins/server/grpc => ../../server/grpc

replace github.com/go-orb/go-orb => ../../../go-orb
