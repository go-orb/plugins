module github.com/go-orb/plugins/config/tests

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230709084536-48ca79fd6450
	github.com/go-orb/plugins/codecs/json v0.0.0
	github.com/go-orb/plugins/codecs/yaml v0.0.0
	github.com/go-orb/plugins/config/source/cli/urfave v0.0.0
	github.com/go-orb/plugins/config/source/file v0.0.0
	github.com/go-orb/plugins/config/source/http v0.0.0
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/urfave/cli/v2 v2.25.7 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/exp v0.0.0-20230711153332-06a737ee72cb // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/go-orb/plugins/codecs/json => ../../codecs/json
	github.com/go-orb/plugins/codecs/yaml => ../../codecs/yaml
	github.com/go-orb/plugins/config/source/cli/urfave => ../source/cli/urfave
	github.com/go-orb/plugins/config/source/file => ../source/file
	github.com/go-orb/plugins/config/source/http => ../source/http
)
