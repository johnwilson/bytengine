package base

import (
	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: server.listdb
func ServerListDb(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return eng.FileSystem.ListDatabase(filter)
}

// handler for: server.newdb
func ServerNewDb(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	db := cmd.Args["database"].(string)
	return eng.FileSystem.CreateDatabase(db)
}

// handler for: server.init
func ServerInit(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	return eng.FileSystem.ClearAll()
}

// handler for: server.dropdb
func ServerDropDb(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	db := cmd.Args["database"].(string)
	return eng.FileSystem.DropDatabase(db)
}

func init() {
	bytengine.RegisterCommandHandler("server.listdb", ServerListDb)
	bytengine.RegisterCommandHandler("server.newdb", ServerNewDb)
	bytengine.RegisterCommandHandler("server.init", ServerInit)
	bytengine.RegisterCommandHandler("server.dropdb", ServerDropDb)
}
