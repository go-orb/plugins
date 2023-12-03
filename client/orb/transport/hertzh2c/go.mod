module github.com/go-orb/plugins/client/orb/transport/hertzh2c

go 1.21.4

require (
	github.com/cloudwego/hertz v0.7.2
	github.com/go-orb/go-orb v0.0.0-20231203061431-2cf52a164da0
	github.com/go-orb/plugins/client/orb v0.0.0-20231203063539-a0de6a0006d9
	github.com/go-orb/plugins/client/orb/transport/basehertz v0.0.0-20231203063539-a0de6a0006d9
	github.com/go-orb/plugins/client/tests v0.0.0-20231203063539-a0de6a0006d9
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/codecs/proto v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/config/source/file v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/log/slog v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/registry/consul v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/registry/mdns v0.0.0-20231203062758-5020673db140
	github.com/go-orb/plugins/server/http v0.0.0-20231203062758-5020673db140
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/andeya/ameda v1.5.3 // indirect
	github.com/andeya/goutil v1.0.1 // indirect
	github.com/bytedance/go-tagexpr/v2 v2.9.11 // indirect
	github.com/bytedance/gopkg v0.0.0-20230728082804-614d0af6619b // indirect
	github.com/bytedance/sonic v1.10.2 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/cloudwego/netpoll v0.5.1 // indirect
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/nyaruka/phonenumbers v1.2.2 // indirect
	github.com/tidwall/gjson v1.17.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	golang.org/x/arch v0.6.0 // indirect
	golang.org/x/exp v0.0.0-20231127185646-65229373498e // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/go-orb/plugins/client/orb/transports/basehertz => ../basehertz
