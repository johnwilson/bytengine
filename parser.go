package bytengine

import (
	"encoding/json"
	"fmt"
	"log"
)

type Parser interface {
	Parse(s string) (c []Command, err error)
}

type Command struct {
	Name     string
	Database string
	Args     map[string]interface{}
	Options  map[string]interface{}
	Filter   string // name of filter to apply to result
	IsAdmin  bool   // is it an admin command
}

func (c Command) String() string {
	val := map[string]interface{}{
		"command":  c.Name,
		"database": c.Database,
		"filter":   c.Filter,
		"args":     c.Args,
		"isadmin":  c.IsAdmin,
	}
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

var parserPlugins = make(map[string]Parser)

func RegisterParser(name string, plugin Parser) {
	if plugin == nil {
		log.Fatal("File System Plugin Registration: plugin is nil")
	}

	if _, exists := parserPlugins[name]; exists {
		log.Printf("Parser Plugin Registration: plugin %q already registered", name)
		return
	}
	parserPlugins[name] = plugin
}

func NewParser(pluginName, config string) (plugin Parser, err error) {
	plugin, ok := parserPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("Parser Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	return
}
