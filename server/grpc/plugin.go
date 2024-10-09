package grpc

import (
	"github.com/go-orb/go-orb/server"
)

// Plugin name.
const Plugin = "grpc"

func init() {
	server.Plugins.Add(Plugin, Provide)
	server.PluginsNew.Add(Plugin, New)
}
