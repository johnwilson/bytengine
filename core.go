package bytengine

import (
	"errors"
	"fmt"

	"github.com/bitly/go-simplejson"
	"github.com/johnwilson/bytengine/dsl"
)

type RequestHandler func(cmd dsl.Command, user *User, e *Engine) Response

type FilterMethod func(r *Response) Response

// Routes commands to handlers
type Router struct {
	cmdRegistry map[string]RequestHandler
	filters     []DataFilter
}

func (cr *Router) Add(name string, fn RequestHandler) {
	cr.cmdRegistry[name] = fn
}

func (cr *Router) Exec(cmd dsl.Command, user *User, e *Engine) Response {
	// check if command in cmdRegistry
	fn, ok := cr.cmdRegistry[cmd.Name]
	if !ok {
		msg := fmt.Sprintf("Command '%s' not found", cmd.Name)
		return ErrorResponse(errors.New(msg))
	}

	msg_auth := "User not authorized to execute command"
	// check id admin command
	if cmd.IsAdmin() && !user.Root {
		return ErrorResponse(errors.New(msg_auth))
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
			return ErrorResponse(errors.New(msg_auth))
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
		return ErrorResponse(fmt.Errorf("Filter '%s' not found", cmd.Filter))
	}
	return val
}

func (cr *Router) AddFilters(f DataFilter) {
	cr.filters = append(cr.filters, f)
}

func NewRouter() *Router {
	cr := Router{}
	cr.cmdRegistry = map[string]RequestHandler{}
	cr.filters = []DataFilter{}
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
	ResultChannel chan Response
}

type Engine struct {
	Router        *Router
	AuthManager   Authentication
	BFSManager    BFS
	BStoreManager ByteStore
	StateManager  StateStore
}

func (e Engine) checkUser(token string) (*User, error) {
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
				srq.ResultChannel <- ErrorResponse(errors.New("Authorization required")).JSON()
				continue
			}
			if err != nil {
				srq.ResultChannel <- ErrorResponse(err).JSON()
				continue
			}

			// parse script
			cmdlist, err := e.parseScript(srq.Text)
			if err != nil {
				srq.ResultChannel <- ErrorResponse(err).JSON()
				continue
			}

			// execute command(s)
			resultset := []interface{}{}
			execerr := false
			for _, cmd := range cmdlist {
				r := e.Router.Exec(cmd, user, &e)
				if r.Status != OK {
					srq.ResultChannel <- r.JSON()
					execerr = true
					break
				}
				resultset = append(resultset, r.Data)
			}

			if !execerr {
				if len(resultset) > 1 {
					srq.ResultChannel <- OKResponse(resultset).JSON()
				} else {
					srq.ResultChannel <- OKResponse(resultset[0]).JSON()
				}
			}
		case crq := <-commands:
			user, err := e.checkUser(crq.Token)
			if err != nil {
				crq.ResultChannel <- ErrorResponse(err)
				continue
			}
			// exec command
			r := e.Router.Exec(crq.Command, user, &e)
			crq.ResultChannel <- r
		}
	}
}

func CreateDataFilter(plugin string, config *simplejson.Json) DataFilter {
	df, err := NewDataFilter(plugin, "")
	if err != nil {
		panic(err)
	}
	return df
}

func CreateAuthManager(plugin string, config *simplejson.Json) Authentication {
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

func CreateBSTManager(plugin string, config *simplejson.Json) ByteStore {
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

func CreateBFSManager(plugin string, bstore *ByteStore, config *simplejson.Json) BFS {
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

func CreateStateManager(plugin string, config *simplejson.Json) StateStore {
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

func CreateAdminUser(plugin, usr, pw string, config *simplejson.Json) error {
	authM := CreateAuthManager(plugin, config)
	err := authM.NewUser(usr, pw, true)
	return err
}
