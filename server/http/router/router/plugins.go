package router

import "go-micro.dev/v5/util/container"

// Plugins is the registry for Logger plugins.
var Plugins = container.NewMap[NewRouterFunc]() //nolint:gochecknoglobals
