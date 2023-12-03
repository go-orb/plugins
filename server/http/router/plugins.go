package router

import "github.com/go-orb/go-orb/util/container"

// Plugins is the registry for Registry plugins.
var Plugins = container.NewSafeMap[string, NewRouterFunc]() //nolint:gochecknoglobals
