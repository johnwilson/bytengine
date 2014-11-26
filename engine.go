package bytengine

import (
	"errors"
	"fmt"

	"github.com/bitly/go-simplejson"
	"github.com/johnwilson/bytengine/dsl"
)

type CommandHandler func(cmd dsl.Command, user *User, eng *Engine) Response

type Engine struct {
	AuthPlugin       Authentication
	FileSystemPlugin BFS
	ByteStorePlugin  ByteStore
	StateStorePlugin StateStore
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

	uname, err := eng.StateStorePlugin.TokenGet(token)
	if err != nil {
		return nil, errors.New("invalid auth token")
	}

	return eng.AuthPlugin.UserInfo(uname)
}

func (eng *Engine) parseScript(script string) ([]dsl.Command, error) {
	var cmds []dsl.Command
	if len(script) == 0 {
		return cmds, errors.New("empty script")
	}
	p := dsl.NewParser()
	cmds, err := p.Parse(script)
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

func createBFSManager(bstore *ByteStore, config *simplejson.Json) BFS {
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

// start engine and configure plugins with 'config'
func (eng *Engine) Start(config *simplejson.Json) {
	eng.AuthPlugin = createAuthManager(config)
	eng.ByteStorePlugin = createBSTManager(config)
	eng.FileSystemPlugin = createBFSManager(&eng.ByteStorePlugin, config)
	eng.StateStorePlugin = createStateManager(config)
}

func (eng *Engine) ExecuteScript(token, script string) (*Response, error) {
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
		r := eng.Exec(cmd, user)
		if r.Status != OK {
			return nil, errors.New(r.StatusMessage)
		}
		resultset = append(resultset, r.Data)
	}

	if len(resultset) > 1 {
		r := OKResponse(resultset)
		return &r, nil
	}

	r := OKResponse(resultset[0])
	return &r, nil
}

func (eng *Engine) ExecuteCommand(token string, cmd dsl.Command) (*Response, error) {
	user, err := eng.checkUser(token)
	if err != nil {
		return nil, err
	}
	// exec command
	r := eng.Exec(cmd, user)
	return &r, nil
}

func (eng *Engine) CreateAdminUser(usr, pw string) error {
	err := eng.AuthPlugin.NewUser(usr, pw, true)
	return err
}
