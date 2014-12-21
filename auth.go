package bytengine

import (
	"fmt"
	"log"
)

var authPlugins = make(map[string]Authentication)

type User struct {
	Username  string   `json:"username"`
	Active    bool     `json:"active"`
	Databases []string `json:"databases"`
	Root      bool     `json:"root"`
}

type Authentication interface {
	Start(config string) error
	ClearAll() error
	Authenticate(usr, pw string) bool
	NewUser(usr, pw string, root bool) error
	ChangeUserPassword(usr, pw string) error
	ChangeUserStatus(usr string, isactive bool) error
	ListUser(rgx string) ([]string, error)
	ChangeUserDbAccess(usr, db string, grant bool) error
	HasDbAccess(usr, db string) bool
	RemoveUser(usr string) error
	UserInfo(u string) (*User, error)
}

func RegisterAuthentication(name string, plugin Authentication) {
	if plugin == nil {
		log.Fatal("Authentication Plugin Registration: plugin is nil")
	}

	if _, exists := authPlugins[name]; exists {
		log.Printf("Authentication Plugin Registration: plugin %q already registered", name)
		return
	}
	authPlugins[name] = plugin
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
