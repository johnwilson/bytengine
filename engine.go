package bytengine

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Config struct {
	Authentication struct{ Plugin string }
	ByteStore      struct{ Plugin string }
	FileSystem     struct{ Plugin string }
	StateStore     struct{ Plugin string }
	DataFilter     struct{ Plugin string }
	Parser         struct{ Plugin string }
}

type ConfigData struct {
	Authentication json.RawMessage
	ByteStore      json.RawMessage
	FileSystem     json.RawMessage
	StateStore     json.RawMessage
	DataFilter     json.RawMessage
	Parser         json.RawMessage
}

type Engine struct {
	Authentication Authentication
	FileSystem     FileSystem
	ByteStore      ByteStore
	StateStore     StateStore
	Parser         Parser
}

func NewEngine() *Engine {
	e := Engine{}
	return &e
}

func (eng *Engine) checkUser(token string) (*User, error) {
	// check token
	if len(token) == 0 {
		// anonymous user
		return nil, nil
	}

	uname, err := eng.StateStore.TokenGet(token)
	if err != nil {
		return nil, errors.New("invalid auth token")
	}

	return eng.Authentication.UserInfo(uname)
}

func (eng *Engine) parseScript(script string) ([]Command, error) {
	var cmds []Command
	if len(script) == 0 {
		return cmds, errors.New("empty script")
	}
	cmds, err := eng.Parser.Parse(script)
	if err != nil {
		return cmds, fmt.Errorf("script parse error:\n%s", err.Error())
	}
	if len(cmds) == 0 {
		return cmds, errors.New("no command found")
	}

	return cmds, nil
}

func createAuthentication(plugin string, config []byte) (Authentication, error) {
	return NewAuthentication(plugin, string(config))
}

func createByteStore(plugin string, config []byte) (ByteStore, error) {
	return NewByteStore(plugin, string(config))
}

func createFileSystem(bstore *ByteStore, plugin string, config []byte) (FileSystem, error) {
	return NewFileSystem(plugin, string(config), bstore)
}

func createStateStore(plugin string, config []byte) (StateStore, error) {
	return NewStateStore(plugin, string(config))
}

func createParser(plugin string, config []byte) (Parser, error) {
	return NewParser(plugin, string(config))
}

// start engine and configure plugins
func (eng *Engine) Start(b []byte) error {
	// read configuration
	config := Config{}
	err := json.Unmarshal(b, &config)
	if err != nil {
		return err
	}
	configdata := ConfigData{}
	err = json.Unmarshal(b, &configdata)
	if err != nil {
		return err
	}

	// get plugins from configuration
	auth, err := createAuthentication(config.Authentication.Plugin, configdata.Authentication)
	if err != nil {
		return err
	}
	bytestore, err := createByteStore(config.ByteStore.Plugin, configdata.ByteStore)
	if err != nil {
		return err
	}
	filesystem, err := createFileSystem(&bytestore, config.FileSystem.Plugin, configdata.FileSystem)
	if err != nil {
		return err
	}
	statestore, err := createStateStore(config.StateStore.Plugin, configdata.StateStore)
	if err != nil {
		return err
	}
	parser, err := createParser(config.Parser.Plugin, configdata.Parser)
	if err != nil {
		return err
	}

	// setup engine
	eng.Authentication = auth
	eng.ByteStore = bytestore
	eng.FileSystem = filesystem
	eng.StateStore = statestore
	eng.Parser = parser

	return nil
}

// script is parsed into commands before execution
func (eng *Engine) ExecuteScript(token, script string) (interface{}, error) {
	// check user
	user, err := eng.checkUser(token)
	// check anonymous login
	if user == nil && err == nil {
		return nil, errors.New("Authorization required")
	}
	if err != nil {
		return nil, err
	}

	// parse script
	cmdlist, err := eng.parseScript(script)
	if err != nil {
		return nil, err
	}

	// execute command(s)
	resultset := []interface{}{}
	for _, cmd := range cmdlist {
		r, err := eng.execute(cmd, user)
		if err != nil {
			return nil, err
		}
		resultset = append(resultset, r)
	}

	if len(resultset) > 1 {
		r := resultset
		return &r, nil
	}

	return resultset[0], nil
}

// command is sent directly for execution
func (eng *Engine) ExecuteCommand(token string, cmd Command) (interface{}, error) {
	user, err := eng.checkUser(token)
	if err != nil {
		return nil, err
	}
	// exec command
	r, err := eng.execute(cmd, user)
	return r, err
}

func (eng *Engine) CreateAdminUser(usr, pw string) error {
	err := eng.Authentication.NewUser(usr, pw, true)
	return err
}
