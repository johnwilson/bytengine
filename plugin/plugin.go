package plugin

import (
	"fmt"

	"github.com/johnwilson/bytengine/auth"
	bst "github.com/johnwilson/bytengine/bytestore"
	"github.com/johnwilson/bytengine/datafilter"
	bfs "github.com/johnwilson/bytengine/filesystem"
	sts "github.com/johnwilson/bytengine/statestore"
)

var authPlugins = make(map[string]auth.Authentication)
var bstPlugins = make(map[string]bst.ByteStore)
var bfsPlugins = make(map[string]bfs.BFS)
var dfPlugins = make(map[string]datafilter.DataFilter)
var stsPlugins = make(map[string]sts.StateStore)

func Register(name string, plugin interface{}) {
	if plugin == nil {
		panic("Plugin Registration: plugin is nil")
	}

	// check plugin type
	switch typ := plugin.(type) {
	case auth.Authentication:
		if _, exists := authPlugins[name]; exists {
			panic(fmt.Sprintf("Authentication Plugin Registration: plugin '%s' already registered", name))
		}
		authPlugins[name] = plugin.(auth.Authentication)
	case bst.ByteStore:
		if _, exists := bstPlugins[name]; exists {
			panic(fmt.Sprintf("Byte Store Plugin Registration: plugin '%s' already registered", name))
		}
		bstPlugins[name] = plugin.(bst.ByteStore)
	case bfs.BFS:
		if _, exists := bfsPlugins[name]; exists {
			panic(fmt.Sprintf("File System Plugin Registration: plugin '%s' already registered", name))
		}
		bfsPlugins[name] = plugin.(bfs.BFS)
	case datafilter.DataFilter:
		if _, exists := dfPlugins[name]; exists {
			panic(fmt.Sprintf("Data Filter Plugin Registration: plugin '%s' already registered", name))
		}
		dfPlugins[name] = plugin.(datafilter.DataFilter)
	case sts.StateStore:
		if _, exists := stsPlugins[name]; exists {
			panic(fmt.Sprintf("State Store Plugin Registration: plugin '%s' already registered", name))
		}
		stsPlugins[name] = plugin.(sts.StateStore)
	default:
		panic(fmt.Sprintf("Plugin Registration: plugin type '%s' isn't supported", typ))
	}
}

func NewAuthentication(pluginName, config string) (plugin auth.Authentication, err error) {
	plugin, ok := authPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("Authentication Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config)
	if err != nil {
		plugin = nil
	}
	return
}

func NewByteStore(pluginName, config string) (plugin bst.ByteStore, err error) {
	plugin, ok := bstPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("Byte Store Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config)
	if err != nil {
		plugin = nil
	}
	return
}

func NewFileSystem(pluginName, config string, b *bst.ByteStore) (plugin bfs.BFS, err error) {
	plugin, ok := bfsPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("File System Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config, b)
	if err != nil {
		plugin = nil
	}
	return
}

func NewDataFilter(pluginName, config string) (plugin datafilter.DataFilter, err error) {
	plugin, ok := dfPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("Data Filter Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config)
	if err != nil {
		plugin = nil
	}
	return
}

func NewStateStore(pluginName, config string) (plugin sts.StateStore, err error) {
	plugin, ok := stsPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("State Store Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config)
	if err != nil {
		plugin = nil
	}
	return
}
