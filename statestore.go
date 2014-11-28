package bytengine

import (
	"fmt"
	"log"
)

var stsPlugins = make(map[string]StateStore)

// Manages authentication tokens, upload tickets and caching
type StateStore interface {
	TokenSet(token, user string, timeout int64) error
	TokenGet(token string) (string, error)
	CacheSet(id, value string, timeout int64) error
	CacheGet(id string) (string, error)
	ClearAll() error
	Start(config string) error
}

func RegisterStateStore(name string, plugin StateStore) {
	if plugin == nil {
		log.Fatal("State Store Plugin Registration: plugin is nil")
	}

	if _, exists := stsPlugins[name]; exists {
		log.Printf("State Store Plugin Registration: plugin '%s' already registered", name)
	}
	stsPlugins[name] = plugin
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
