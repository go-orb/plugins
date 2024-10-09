package hertz

import (
	"github.com/go-orb/go-orb/server"
)

// Plugin is the plugin name.
const Plugin = "hertz"

func init() {
	server.Plugins.Add(Plugin, Provide)
	server.PluginsNew.Add(Plugin, New)
}
