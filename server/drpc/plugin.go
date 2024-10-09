package drpc

import (
	"github.com/go-orb/go-orb/server"
)

func init() {
	server.Plugins.Add(Plugin, Provide)
	server.PluginsNew.Add(Plugin, New)
}
