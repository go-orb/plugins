module github.com/go-orb/plugins/client/orb/transport/basehttp

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230728000045-a99830943143
	github.com/go-orb/plugins/client/orb v0.0.0-20230713091520-67e7b5a34489
)

require golang.org/x/exp v0.0.0-20230801115018-d63ba01acd4b // indirect

replace github.com/go-orb/plugins/client/orb => ../..

replace github.com/go-orb/go-orb => ../../../../../go-orb
