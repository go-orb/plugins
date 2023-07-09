package router

import "github.com/go-orb/go-orb/util/container"

// Plugins is the registry for Logger plugins.
var Plugins = container.NewMap[NewRouterFunc]() //nolint:gochecknoglobals
