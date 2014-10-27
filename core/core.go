package core

import (
	"errors"
	"fmt"
	"github.com/astaxie/beego/cache"
	"github.com/johnwilson/bytengine/auth"
	"github.com/johnwilson/bytengine/bfs"
	"github.com/johnwilson/bytengine/bst"
	"github.com/johnwilson/bytengine/dsl"
	"github.com/johnwilson/bytengine/ext"
)

type RequestHandler func(cmd dsl.Command, user *auth.User, e *Engine) bfs.BFSResponse

type FilterMethod func(r *bfs.BFSResponse) bfs.BFSResponse

type CommandRouter struct {
	cmdRegistry map[string]RequestHandler
	filters     []ext.DataFilter
}

func (cr *CommandRouter) AddCommandHandler(name string, fn RequestHandler) {
	cr.cmdRegistry[name] = fn
}

func (cr *CommandRouter) Exec(cmd dsl.Command, user *auth.User, e *Engine) bfs.BFSResponse {
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

func (cr *CommandRouter) AddFilters(f ext.DataFilter) {
	cr.filters = append(cr.filters, f)
}

func NewCommandRouter() *CommandRouter {
	cr := CommandRouter{}
	cr.cmdRegistry = map[string]RequestHandler{}
	cr.filters = []ext.DataFilter{}
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
	ResultChannel chan bfs.BFSResponse
}

type Engine struct {
	Router        *CommandRouter
	AuthManager   auth.Authentication
	BFSManager    bfs.BFS
	BStoreManager bst.ByteStore
	CacheManager  cache.Cache
}

func (e Engine) checkUser(token string) (*auth.User, error) {
	// check token
	if len(token) == 0 {
		// anonymous user
		return nil, nil
	}

	exists := e.CacheManager.IsExist(token)
	if !exists {
		return nil, errors.New("invalid auth token")
	}

	// retrieve user
	uname := cache.GetString(e.CacheManager.Get(token))
	if len(uname) == 0 {
		return nil, errors.New("user not found")
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
				if !r.Success() {
					srq.ResultChannel <- r.JSON()
					execerr = true
					break
				}
				resultset = append(resultset, r.Data())
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
