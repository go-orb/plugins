module github.com/go-orb/plugins/client/orb/transport/basehttp

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231127002523-4909ba192408
	github.com/go-orb/plugins/client/orb v0.0.0-20231126232626-f2cd47f2724d
)

require golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect

replace github.com/go-orb/plugins/client/orb => ../..
