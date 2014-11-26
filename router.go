package bytengine

import (
	"errors"
	"fmt"

	"github.com/johnwilson/bytengine/dsl"
)

type CommandHandler func(cmd dsl.Command, user *User, eng *Engine) Response
type DataFilter func(r *Response, eng *Engine) Response

var cmdHandlerRegistry = make(map[string]CommandHandler)
var dataFilterRegistry = make(map[string]DataFilter)

func RegisterCommandHandler(name string, fn CommandHandler) {
	if fn == nil {
		panic("Command Handler registration: handler is nil")
	}

	if _, exists := cmdHandlerRegistry[name]; exists {
		panic(fmt.Sprintf("Command Handler registration: handler '%s' already added", name))
	}
	cmdHandlerRegistry[name] = fn
}

func RegisterDataFilter(name string, fn DataFilter) {
	if fn == nil {
		panic("Data Filter registration: filter is nil")
	}

	if _, exists := dataFilterRegistry[name]; exists {
		panic(fmt.Sprintf("Data Filter registration: filter '%s' already added", name))
	}
	dataFilterRegistry[name] = fn
}

func (eng *Engine) Exec(cmd dsl.Command, user *User) Response {
	// check if command in cmdHandlerRegistry
	fn, ok := cmdHandlerRegistry[cmd.Name]
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

	val := fn(cmd, user, eng)
	// check sendto
	if cmd.Filter != "" {
		df, ok := dataFilterRegistry[cmd.Filter]
		if !ok {
			msg := fmt.Sprintf("Filter '%s' not found", cmd.Filter)
			return ErrorResponse(errors.New(msg))
		}

		return df(&val, eng)
	}
	return val
}
