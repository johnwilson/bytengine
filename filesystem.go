package bytengine

import (
	"fmt"
	"log"
)

var bfsPlugins = make(map[string]FileSystem)

type FileSystem interface {
	Start(config string, b *ByteStore) error
	ClearAll() ([]string, error)
	ListDatabase(filter string) ([]string, error)
	CreateDatabase(db string) error
	DropDatabase(db string) error
	NewDir(p, db string) error
	NewFile(p, db string, jsondata map[string]interface{}) error
	ListDir(p, filter, db string) (map[string][]string, error)
	ReadJson(p, db string, fields []string) (interface{}, error)
	Delete(p, db string) error
	Rename(p, newname, db string) error
	Move(from, to, db string) error
	Copy(from, to, db string) error
	Info(p, db string) (map[string]interface{}, error)
	FileAccess(p, db string, protect bool) error
	SetCounter(counter, action string, value int64, db string) (int64, error)
	ListCounter(filter, db string) (map[string]int64, error)
	WriteBytes(p, ap, db string) (int64, error)
	ReadBytes(fp, db string) (string, error)
	DirectAccess(fp, db, layer string) (map[string]interface{}, string, error)
	DeleteBytes(p, db string) error
	UpdateJson(p, db string, j map[string]interface{}) error
	BQLSearch(db string, query map[string]interface{}) (interface{}, error)
	BQLSet(db string, query map[string]interface{}) (int, error)
	BQLUnset(db string, query map[string]interface{}) (int, error)
}

func RegisterFileSystem(name string, plugin FileSystem) {
	if plugin == nil {
		log.Fatal("File System Plugin Registration: plugin is nil")
	}

	if _, exists := bfsPlugins[name]; exists {
		log.Printf("File System Plugin Registration: plugin '%s' already registered", name)
	}
	bfsPlugins[name] = plugin
}

func NewFileSystem(pluginName, config string, b *ByteStore) (plugin FileSystem, err error) {
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
