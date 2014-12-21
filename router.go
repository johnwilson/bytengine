package bytengine

import (
	"errors"
	"fmt"
	"log"
)

type CommandHandler func(cmd Command, user *User, eng *Engine) (interface{}, error)
type DataFilter func(r interface{}, eng *Engine) (interface{}, error)

var cmdHandlerRegistry = make(map[string]CommandHandler)
var dataFilterRegistry = make(map[string]DataFilter)

func RegisterCommandHandler(name string, fn CommandHandler) {
	if fn == nil {
		log.Fatal("Command Handler registration: handler is nil")
	}

	if _, exists := cmdHandlerRegistry[name]; exists {
		log.Printf("Command Handler registration: handler %q already added", name)
		return
	}
	cmdHandlerRegistry[name] = fn
}

func RegisterDataFilter(name string, fn DataFilter) {
	if fn == nil {
		log.Fatal("Data Filter registration: filter is nil")
	}

	if _, exists := dataFilterRegistry[name]; exists {
		log.Printf("Data Filter registration: filter %q already added", name)
		return
	}
	dataFilterRegistry[name] = fn
}

func (eng *Engine) execute(cmd Command, user *User) (interface{}, error) {
	// check if command in cmdHandlerRegistry
	fn, ok := cmdHandlerRegistry[cmd.Name]
	if !ok {
		err := errors.New(fmt.Sprintf("Command '%s' not found", cmd.Name))
		return nil, err
	}

	err := errors.New("User not authorized to execute command")
	// check id admin command
	if cmd.IsAdmin && !user.Root {
		return nil, err
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
			return nil, err
		}
	}

	val, err := fn(cmd, user, eng)
	if err != nil {
		return nil, err
	}
	// check sendto
	if cmd.Filter != "" {
		df, ok := dataFilterRegistry[cmd.Filter]
		if !ok {
			err := errors.New(fmt.Sprintf("Filter '%s' not found", cmd.Filter))
			return nil, err
		}

		return df(val, eng)
	}
	return val, nil
}
