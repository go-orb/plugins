module github.com/go-orb/plugins/benchmarks/rps_no_hertz

go 1.21

require (
	github.com/go-orb/go-orb v0.0.0-20240808121121-1f65fceb24af
	github.com/go-orb/plugins/client/middleware/log v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb_transport/drpc v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb_transport/grpc v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb_transport/h2c v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb_transport/http v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb_transport/http3 v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/client/orb_transport/https v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/codecs/proto v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/config/source/cli/urfave v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/config/source/file v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/log/lumberjack v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/log/slog v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/registry/consul v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/registry/mdns v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/server/drpc v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/server/grpc v0.0.0-20240809105731-a8ee45a15344
	github.com/go-orb/plugins/server/http v0.0.0-20240809105731-a8ee45a15344
	github.com/google/wire v0.6.0
	github.com/hashicorp/consul/sdk v0.16.1
	google.golang.org/genproto/googleapis/api v0.0.0-20240808171019-573a1156607a
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	storj.io/drpc v0.0.34
)

require (
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/go-chi/chi v1.5.5 // indirect
	github.com/go-orb/plugins/client/orb_transport/basehttp v0.0.0-20240809105731-a8ee45a15344 // indirect
	github.com/go-orb/plugins/registry/regutil v0.0.0-20240809105731-a8ee45a15344 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/consul/api v1.29.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.61 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/onsi/ginkgo/v2 v2.20.0 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/quic-go v0.46.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/urfave/cli/v2 v2.27.3 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/zeebo/errs v1.3.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/exp v0.0.0-20240808152545-0cdaa3abc0fa // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240808171019-573a1156607a // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
