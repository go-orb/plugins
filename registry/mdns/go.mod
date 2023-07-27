module github.com/go-orb/plugins/registry/mdns

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230727224033-0f3d9735ffae
	github.com/go-orb/plugins/log/slog v0.0.0-20230726151712-9ff06382f83b
	github.com/go-orb/plugins/registry/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/google/uuid v1.3.0
	github.com/miekg/dns v1.1.55
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090
	golang.org/x/net v0.12.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/registry/tests => ../tests
