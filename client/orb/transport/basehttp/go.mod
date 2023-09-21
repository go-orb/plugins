module github.com/go-orb/plugins/client/orb/transport/basehttp

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230805173903-ba3da7c24b9d
	github.com/go-orb/plugins/client/orb v0.0.0-20230713091520-67e7b5a34489
)

require golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect

replace github.com/go-orb/plugins/client/orb => ../..

replace github.com/go-orb/go-orb => ../../../../../go-orb
