<<<<<<< HEAD
module github.com/go-micro/plugins/server/grpc

go 1.19

require google.golang.org/grpc v1.51.0

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

// replace github.com/go-micro/plugins/server/grpc => ./

// replace github.com/go-micro/plugins/server/grpc/backend => ./backend
=======
module github.com/go-orb/plugins/server/grpc

go 1.19

require (
	github.com/google/uuid v1.3.0
	github.com/go-orb/go-orb v0.0.1 
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df
	google.golang.org/grpc v1.56.2
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto v0.0.0-20230629202037-9506855d4529 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230706204954-ccb25ca9f130 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
>>>>>>> 3191204 (feat: update to go-orb/go-orb)
