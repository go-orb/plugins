module github.com/go-orb/plugins/server/tests

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230725190534-6e856aec238f
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/config/source/file v0.0.0-20230713091520-67e7b5a34489
	github.com/google/uuid v1.3.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/codecs/yaml => ../../codecs/yaml

replace github.com/go-orb/plugins/config/source/file => ../../config/source/file
