package bytengine

import (
	"errors"
	"fmt"

	"github.com/bitly/go-simplejson"
	"github.com/johnwilson/bytengine/dsl"
)

type CommandHandler func(cmd dsl.Command, user *User) Response

var (
	AuthPlugin       Authentication
	FileSystemPlugin BFS
	ByteStorePlugin  ByteStore
	StateStorePlugin StateStore
	DataFilterPlugin DataFilter
)

func checkUser(token string) (*User, error) {
	// check token
	if len(token) == 0 {
		// anonymous user
		return nil, nil
	}

	uname, err := StateStorePlugin.TokenGet(token)
	if err != nil {
		return nil, errors.New("invalid auth token")
	}

	return AuthPlugin.UserInfo(uname)
}

func parseScript(script string) ([]dsl.Command, error) {
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

func createDataFilter(config *simplejson.Json) DataFilter {
	plugin := config.Get("datafilter").Get("plugin").MustString("")
	df, err := NewDataFilter(plugin, "")
	if err != nil {
		panic(err)
	}
	return df
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
func Start(config *simplejson.Json) {
	AuthPlugin = createAuthManager(config)
	ByteStorePlugin = createBSTManager(config)
	FileSystemPlugin = createBFSManager(&ByteStorePlugin, config)
	StateStorePlugin = createStateManager(config)
	DataFilterPlugin = createDataFilter(config)
}

func ExecuteScript(token, script string) (*Response, error) {
	// check user
	user, err := checkUser(token)
	// check anonymous login
	if user == nil && err == nil {
		return nil, errors.New("Authorization required")
	}
	if err != nil {
		return nil, err
	}

	// parse script
	cmdlist, err := parseScript(script)
	if err != nil {
		return nil, err
	}

	// execute command(s)
	resultset := []interface{}{}
	for _, cmd := range cmdlist {
		r := Exec(cmd, user)
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

func ExecuteCommand(token string, cmd dsl.Command) (*Response, error) {
	user, err := checkUser(token)
	if err != nil {
		return nil, err
	}
	// exec command
	r := Exec(cmd, user)
	return &r, nil
}

func CreateAdminUser(usr, pw string) error {
	err := AuthPlugin.NewUser(usr, pw, true)
	return err
}
