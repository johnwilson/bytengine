package bytengine

import (
	"errors"
	"fmt"

	"github.com/bitly/go-simplejson"
)

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

func createAuthManager(config *simplejson.Json) Authentication {
	plugin := config.Get("auth").Get("plugin").MustString("")
	b, err := config.Get("auth").MarshalJSON()
	if err != nil {
		panic(err)
	}
	authM, err := NewAuthentication(plugin, string(b))
	if err != nil {
		panic(err)
	}
	return authM
}

func createBSTManager(config *simplejson.Json) ByteStore {
	plugin := config.Get("bst").Get("plugin").MustString("")
	b, err := config.Get("bst").MarshalJSON()
	if err != nil {
		panic(err)
	}
	bstM, err := NewByteStore(plugin, string(b))
	if err != nil {
		panic(err)
	}
	return bstM
}

func createBFSManager(bstore *ByteStore, config *simplejson.Json) FileSystem {
	plugin := config.Get("bfs").Get("plugin").MustString("")
	b, err := config.Get("bfs").MarshalJSON()
	if err != nil {
		panic(err)
	}
	bfsM, err := NewFileSystem(plugin, string(b), bstore)
	if err != nil {
		panic(err)
	}
	return bfsM
}

func createStateManager(config *simplejson.Json) StateStore {
	plugin := config.Get("state").Get("plugin").MustString("")
	b, err := config.Get("state").MarshalJSON()
	if err != nil {
		panic(err)
	}
	stateM, err := NewStateStore(plugin, string(b))
	if err != nil {
		panic(err)
	}
	return stateM
}

func createParser(config *simplejson.Json) Parser {
	plugin := config.Get("parser").Get("plugin").MustString("")
	b, err := config.Get("parser").MarshalJSON()
	if err != nil {
		panic(err)
	}
	parser, err := NewParser(plugin, string(b))
	if err != nil {
		panic(err)
	}
	return parser
}

// start engine and configure plugins with 'config'
func (eng *Engine) Start(config *simplejson.Json) {
	eng.Authentication = createAuthManager(config)
	eng.ByteStore = createBSTManager(config)
	eng.FileSystem = createBFSManager(&eng.ByteStore, config)
	eng.StateStore = createStateManager(config)
	eng.Parser = createParser(config)
}

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
