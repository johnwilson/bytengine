package bytengine

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	// to be expanded
	MimeList = map[string]string{
		".js":  "text/javascript",
		".css": "text/css",
	}

	bstPlugins = make(map[string]ByteStore)
)

type ByteStore interface {
	Start(config string) error
	Add(db string, file *os.File) (map[string]interface{}, error)
	Update(db, id string, file *os.File) (map[string]interface{}, error)
	Delete(db, id string) error
	Read(db, filename string, file io.Writer) error
	DropDatabase(db string) error
}

func RegisterByteStore(name string, plugin ByteStore) {
	if plugin == nil {
		log.Fatal("Byte Store Plugin Registration: plugin is nil")
	}

	if _, exists := bstPlugins[name]; exists {
		log.Printf("Byte Store Plugin Registration: plugin %q already registered", name)
		return
	}
	bstPlugins[name] = plugin
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
