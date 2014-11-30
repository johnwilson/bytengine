package base

import (
	"github.com/johnwilson/bytengine"
)

// handler for: server.listdb
func ServerListDb(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return eng.FileSystem.ListDatabase(filter)
}

// handler for: server.newdb
func ServerNewDb(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	db := cmd.Args["database"].(string)
	if err := eng.FileSystem.CreateDatabase(db); err != nil {
		return nil, err
	}
	return true, nil
}

// handler for: server.init
func ServerInit(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	return eng.FileSystem.ClearAll()
}

// handler for: server.dropdb
func ServerDropDb(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	db := cmd.Args["database"].(string)
	if err := eng.FileSystem.DropDatabase(db); err != nil {
		return nil, err
	}
	return true, nil
}

func init() {
	bytengine.RegisterCommandHandler("server.listdb", ServerListDb)
	bytengine.RegisterCommandHandler("server.newdb", ServerNewDb)
	bytengine.RegisterCommandHandler("server.init", ServerInit)
	bytengine.RegisterCommandHandler("server.dropdb", ServerDropDb)
}
