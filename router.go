package bytengine

import (
	"errors"
	"fmt"

	"github.com/johnwilson/bytengine/dsl"
)

var commandRegistry = make(map[string]CommandHandler)
var datafilters = make([]DataFilter, 1)

func RegisterCommandHandler(name string, fn CommandHandler) {
	if fn == nil {
		panic("Command Handler Addition: handler is nil")
	}

	if _, exists := commandRegistry[name]; exists {
		panic(fmt.Sprintf("Command Handler Addition: handler '%s' already added", name))
	}
	commandRegistry[name] = fn
}

func RegisterFilters(f DataFilter) {
	if f == nil {
		panic("Data Filter Addition: data filter is nil")
	}
	datafilters = append(datafilters, f)
}

func Exec(cmd dsl.Command, user *User) Response {
	// check if command in commandRegistry
	fn, ok := commandRegistry[cmd.Name]
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

	val := fn(cmd, user)
	// check sendto
	if cmd.Filter != "" {
		for _, filtergroup := range datafilters {
			if filtergroup.Check(cmd.Filter) {
				return filtergroup.Apply(cmd.Filter, &val)
			}
		}
		return ErrorResponse(fmt.Errorf("Filter '%s' not found", cmd.Filter))
	}
	return val
}
