module github.com/go-micro/plugins/log/text

go 1.19

replace go-micro.dev/v5 => ../../../orb

replace github.com/go-orb/config => ../../../config

require (
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	golang.org/x/exp v0.0.0-20221031165847-c99f073a8326
)

require github.com/go-orb/config v0.0.0-20221031022024-e60230f51cb8 // indirect
