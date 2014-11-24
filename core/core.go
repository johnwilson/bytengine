package core

import (
	"errors"
	"fmt"

	"github.com/bitly/go-simplejson"
	"github.com/johnwilson/bytengine/auth"
	bst "github.com/johnwilson/bytengine/bytestore"
	"github.com/johnwilson/bytengine/datafilter"
	"github.com/johnwilson/bytengine/dsl"
	bfs "github.com/johnwilson/bytengine/filesystem"
	"github.com/johnwilson/bytengine/plugin"
	sts "github.com/johnwilson/bytengine/statestore"
)

type RequestHandler func(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response

type FilterMethod func(r *bfs.Response) bfs.Response

// Routes commands to handlers
type Router struct {
	cmdRegistry map[string]RequestHandler
	filters     []datafilter.DataFilter
}

func (cr *Router) Add(name string, fn RequestHandler) {
	cr.cmdRegistry[name] = fn
}

func (cr *Router) Exec(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	// check if command in cmdRegistry
	fn, ok := cr.cmdRegistry[cmd.Name]
	if !ok {
		msg := fmt.Sprintf("Command '%s' not found", cmd.Name)
		return bfs.ErrorResponse(errors.New(msg))
	}

	msg_auth := "User not authorized to execute command"
	// check id admin command
	if cmd.IsAdmin() && !user.Root {
		return bfs.ErrorResponse(errors.New(msg_auth))
	}

	// check user database access
	if cmd.Database != "" && !user.Root {
		dbaccess := false
		for _, item := range user.Databases {
			if cmd.Database == item {
				dbaccess = true
				break
			}
		}
		if !dbaccess {
			return bfs.ErrorResponse(errors.New(msg_auth))
		}
	}

	val := fn(cmd, user, e)
	// check sendto
	if cmd.Filter != "" {
		for _, filtergroup := range cr.filters {
			if filtergroup.Check(cmd.Filter) {
				return filtergroup.Apply(cmd.Filter, &val)
			}
		}
		return bfs.ErrorResponse(fmt.Errorf("Filter '%s' not found", cmd.Filter))
	}
	return val
}

func (cr *Router) AddFilters(f datafilter.DataFilter) {
	cr.filters = append(cr.filters, f)
}

func NewRouter() *Router {
	cr := Router{}
	cr.cmdRegistry = map[string]RequestHandler{}
	cr.filters = []datafilter.DataFilter{}
	return &cr
}

type ScriptRequest struct {
	Text          string
	Token         string
	ResultChannel chan []byte
}

type CommandRequest struct {
	Command       dsl.Command
	Token         string
	ResultChannel chan bfs.Response
}

type Engine struct {
	Router        *Router
	AuthManager   auth.Authentication
	BFSManager    bfs.BFS
	BStoreManager bst.ByteStore
	StateManager  sts.StateStore
}

func (e Engine) checkUser(token string) (*auth.User, error) {
	// check token
	if len(token) == 0 {
		// anonymous user
		return nil, nil
	}

	uname, err := e.StateManager.TokenGet(token)
	if err != nil {
		return nil, errors.New("invalid auth token")
	}

	return e.AuthManager.UserInfo(uname)
}

func (e Engine) parseScript(script string) ([]dsl.Command, error) {
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

func (e Engine) Start(scripts chan *ScriptRequest, commands chan *CommandRequest) {
	for {
		select {
		case srq := <-scripts:
			// check user
			user, err := e.checkUser(srq.Token)
			// check anonymous login
			if user == nil && err == nil {
				srq.ResultChannel <- bfs.ErrorResponse(errors.New("Authorization required")).JSON()
				continue
			}
			if err != nil {
				srq.ResultChannel <- bfs.ErrorResponse(err).JSON()
				continue
			}

			// parse script
			cmdlist, err := e.parseScript(srq.Text)
			if err != nil {
				srq.ResultChannel <- bfs.ErrorResponse(err).JSON()
				continue
			}

			// execute command(s)
			resultset := []interface{}{}
			execerr := false
			for _, cmd := range cmdlist {
				r := e.Router.Exec(cmd, user, &e)
				if r.Status != bfs.OK {
					srq.ResultChannel <- r.JSON()
					execerr = true
					break
				}
				resultset = append(resultset, r.Data)
			}

			if !execerr {
				if len(resultset) > 1 {
					srq.ResultChannel <- bfs.OKResponse(resultset).JSON()
				} else {
					srq.ResultChannel <- bfs.OKResponse(resultset[0]).JSON()
				}
			}
		case crq := <-commands:
			user, err := e.checkUser(crq.Token)
			if err != nil {
				crq.ResultChannel <- bfs.ErrorResponse(err)
				continue
			}
			// exec command
			r := e.Router.Exec(crq.Command, user, &e)
			crq.ResultChannel <- r
		}
	}
}

func CreateDataFilter(config *simplejson.Json) datafilter.DataFilter {
	df, err := plugin.NewDataFilter(DATA_FILTER_PLUGIN, "")
	if err != nil {
		panic(err)
	}
	return df
}

func CreateAuthManager(config *simplejson.Json) auth.Authentication {
	b, err := config.Get("auth").MarshalJSON()
	if err != nil {
		panic(err)
	}
	authM, err := plugin.NewAuthentication(AUTH_PLUGIN, string(b))
	if err != nil {
		panic(err)
	}
	return authM
}

func CreateBSTManager(config *simplejson.Json) bst.ByteStore {
	b, err := config.Get("bst").MarshalJSON()
	if err != nil {
		panic(err)
	}
	bstM, err := plugin.NewByteStore(BST_PLUGIN, string(b))
	if err != nil {
		panic(err)
	}
	return bstM
}

func CreateBFSManager(bstore *bst.ByteStore, config *simplejson.Json) bfs.BFS {
	b, err := config.Get("bfs").MarshalJSON()
	if err != nil {
		panic(err)
	}
	bfsM, err := plugin.NewFileSystem(BFS_PLUGIN, string(b), bstore)
	if err != nil {
		panic(err)
	}
	return bfsM
}

func CreateStateManager(config *simplejson.Json) sts.StateStore {
	b, err := config.Get("state").MarshalJSON()
	if err != nil {
		panic(err)
	}
	stateM, err := plugin.NewStateStore(STATE_PLUGIN, string(b))
	if err != nil {
		panic(err)
	}
	return stateM
}

func WorkerPool(n int, config *simplejson.Json) (chan *ScriptRequest, chan *CommandRequest) {
	queries := make(chan *ScriptRequest)
	commands := make(chan *CommandRequest)

	for i := 0; i < n; i++ {
		authM := CreateAuthManager(config)
		bstM := CreateBSTManager(config)
		bfsM := CreateBFSManager(&bstM, config)
		stateM := CreateStateManager(config)
		df := CreateDataFilter(config)
		router := NewRouter()
		router.AddFilters(df)
		initialize(router)
		eng := Engine{router, authM, bfsM, bstM, stateM}

		go eng.Start(queries, commands)
	}

	return queries, commands
}

func CreateAdminUser(usr, pw string, config *simplejson.Json) error {
	authM := CreateAuthManager(config)
	err := authM.NewUser(usr, pw, true)
	return err
}
