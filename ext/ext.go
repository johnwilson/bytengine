package ext

import (
	"fmt"
	"github.com/johnwilson/bytengine/bfs"
)

type DataFilter interface {
	Start(config string) error
	Apply(filter string, r *bfs.BFSResponse) bfs.BFSResponse
	Info(filter string) string
	Check(filter string) bool
	All() []string
}

var plugins = make(map[string]DataFilter)

func Register(name string, plugin DataFilter) {
	if plugin == nil {
		panic("DataFilter Plugin Registration: plugin is nil")
	}
	if _, exists := plugins[name]; exists {
		panic("DataFilter Plugin Registration: plugin '" + name + "' already registered")
	}
	plugins[name] = plugin
}

func NewPlugin(pluginName, config string) (plugin DataFilter, err error) {
	plugin, ok := plugins[pluginName]
	if !ok {
		err = fmt.Errorf("DataFilter Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config)
	if err != nil {
		plugin = nil
	}
	return
}
