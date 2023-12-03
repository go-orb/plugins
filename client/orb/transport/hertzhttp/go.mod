module github.com/go-orb/plugins/client/orb/transport/hertzhttp

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231127002523-4909ba192408
	github.com/go-orb/plugins/client/orb v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/client/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/codecs/proto v0.0.0
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/config/source/file v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/log/slog v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/registry/consul v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/registry/mdns v0.0.0-20231126232626-f2cd47f2724d
	github.com/go-orb/plugins/server/http v0.0.0-20231126232626-f2cd47f2724d
	github.com/stretchr/testify v1.8.4
)

replace github.com/go-orb/plugins/client/orb/transport/basehertz => ../basehertz
