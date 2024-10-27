module github.com/go-orb/plugins/client/tests

go 1.23

toolchain go1.23.0

require (
	github.com/go-orb/go-orb v0.0.0-20241027004147-7e65db198d73
	github.com/go-orb/plugins-experimental/registry/mdns v0.0.0-20241027000905-359dfc52c961
	github.com/go-orb/plugins/client/orb/transport/drpc v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/client/orb/transport/grpc v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/client/orb/transport/h2c v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/client/orb/transport/hertzh2c v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/client/orb/transport/hertzhttp v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/client/orb/transport/http v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/client/orb/transport/https v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/codecs/json v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/codecs/proto v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/config/source/cli/urfave v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/config/source/file v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/log/lumberjack v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/log/slog v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/server/drpc v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/server/grpc v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/server/hertz v0.0.0-20241027005319-1bca6b63ac9e
	github.com/go-orb/plugins/server/http v0.0.0-20241027005319-1bca6b63ac9e
	github.com/google/wire v0.6.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1
	storj.io/drpc v0.0.34
)

require (
	github.com/andeya/ameda v1.5.3 // indirect
	github.com/andeya/goutil v1.0.1 // indirect
	github.com/bytedance/go-tagexpr/v2 v2.9.11 // indirect
	github.com/bytedance/gopkg v0.1.1 // indirect
	github.com/bytedance/sonic v1.12.3 // indirect
	github.com/bytedance/sonic/loader v0.2.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/hertz v0.9.3 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/cloudwego/netpoll v0.6.4 // indirect
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-chi/chi/v5 v5.1.0 // indirect
	github.com/go-orb/plugins/client/orb v0.0.0-20241027005319-1bca6b63ac9e // indirect
	github.com/go-orb/plugins/client/orb/transport/basehertz v0.0.0-20241027005319-1bca6b63ac9e // indirect
	github.com/go-orb/plugins/client/orb/transport/basehttp v0.0.0-20241027005319-1bca6b63ac9e // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/pprof v0.0.0-20241023014458-598669927662 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hertz-contrib/http2 v0.1.8 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/miekg/dns v1.1.62 // indirect
	github.com/nyaruka/phonenumbers v1.4.1 // indirect
	github.com/onsi/ginkgo/v2 v2.20.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.48.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/urfave/cli/v2 v2.27.5 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/arch v0.11.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241021214115-324edc3d5d38 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
