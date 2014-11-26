package bytengine

import (
	"fmt"
)

var authPlugins = make(map[string]Authentication)
var bstPlugins = make(map[string]ByteStore)
var bfsPlugins = make(map[string]BFS)
var dfPlugins = make(map[string]DataFilter)
var stsPlugins = make(map[string]StateStore)

func RegisterAuthentication(name string, plugin Authentication) {
	if plugin == nil {
		panic("Authentication Plugin Registration: plugin is nil")
	}

	if _, exists := authPlugins[name]; exists {
		panic(fmt.Sprintf("Authentication Plugin Registration: plugin '%s' already registered", name))
	}
	authPlugins[name] = plugin
}

func RegisterByteStore(name string, plugin ByteStore) {
	if plugin == nil {
		panic("Byte Store Plugin Registration: plugin is nil")
	}

	if _, exists := bstPlugins[name]; exists {
		panic(fmt.Sprintf("Byte Store Plugin Registration: plugin '%s' already registered", name))
	}
	bstPlugins[name] = plugin
}

func RegisterFileSystem(name string, plugin BFS) {
	if plugin == nil {
		panic("File System Plugin Registration: plugin is nil")
	}

	if _, exists := bfsPlugins[name]; exists {
		panic(fmt.Sprintf("File System Plugin Registration: plugin '%s' already registered", name))
	}
	bfsPlugins[name] = plugin
}

func RegisterDataFilter(name string, plugin DataFilter) {
	if plugin == nil {
		panic("Data Filter Plugin Registration: plugin is nil")
	}

	if _, exists := dfPlugins[name]; exists {
		panic(fmt.Sprintf("Data Filter Plugin Registration: plugin '%s' already registered", name))
	}
	dfPlugins[name] = plugin
}

func RegisterStateStore(name string, plugin StateStore) {
	if plugin == nil {
		panic("State Store Plugin Registration: plugin is nil")
	}

	if _, exists := stsPlugins[name]; exists {
		panic(fmt.Sprintf("State Store Plugin Registration: plugin '%s' already registered", name))
	}
	stsPlugins[name] = plugin
}

func NewAuthentication(pluginName, config string) (plugin Authentication, err error) {
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

func NewByteStore(pluginName, config string) (plugin ByteStore, err error) {
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

func NewFileSystem(pluginName, config string, b *ByteStore) (plugin BFS, err error) {
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

func NewDataFilter(pluginName, config string) (plugin DataFilter, err error) {
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

func NewStateStore(pluginName, config string) (plugin StateStore, err error) {
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
