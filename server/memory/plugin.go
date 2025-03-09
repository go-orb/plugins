package memory

import (
	"github.com/go-orb/go-orb/server"
)

func init() {
	server.Plugins.Add("memory", Provide)
	server.PluginsNew.Add("memory", New)
}
