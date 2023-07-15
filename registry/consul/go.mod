module github.com/go-orb/plugins/registry/consul

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230714131405-cb8d9dfee72f
	github.com/go-orb/plugins/log/text v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/registry/regutil v0.0.0-20230714131340-78ea56c12430
	github.com/go-orb/plugins/registry/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/hashicorp/consul/api v1.22.0
	github.com/hashicorp/consul/sdk v0.14.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/exp v0.0.0-20230713183714-613f0c0eb8a1 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/log/text => ../../log/text

replace github.com/go-orb/plugins/registry/regutil => ../regutil

replace github.com/go-orb/plugins/registry/tests => ../tests
