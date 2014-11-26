package core

import (
	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: database.newdir
func DbNewDir(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.NewDir(path, db)
}

// handler for: database.newfile
func DbNewFile(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	return eng.FileSystemPlugin.NewFile(path, db, data)
}

// handler for: database.listdir
func DbListDir(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	db := cmd.Database
	return eng.FileSystemPlugin.ListDir(path, filter, db)
}

// handler for: database.rename
func DbRename(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	name := cmd.Args["name"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.Rename(path, name, db)
}

// handler for: database.move
func DbMove(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.Move(path, to, db)
}

// handler for: database.copy
func DbCopy(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.Copy(path, to, db)
}

// handler for: database.delete
func DbDelete(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.Delete(path, db)
}

// handler for: database.info
func DbInfo(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.Info(path, db)
}

// handler for: database.makepublic
func DbMakePublic(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.FileAccess(path, db, false)
}

// handler for: database.makeprivate
func DbMakePrivate(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.FileAccess(path, db, true)
}

// handler for: database.readfile
func DbReadFile(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	fields := cmd.Args["fields"].([]string)
	db := cmd.Database
	return eng.FileSystemPlugin.ReadJson(path, db, fields)
}

// handler for: database.modfile
func DbModFile(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	return eng.FileSystemPlugin.UpdateJson(path, db, data)
}

// handler for: database.deletebytes
func DbDeleteBytes(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystemPlugin.DeleteBytes(path, db)
}

// handler for: database.counter
func DbCounter(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	act := cmd.Args["action"].(string)
	db := cmd.Database
	if act != "list" {
		name := cmd.Args["name"].(string)
		val := cmd.Args["value"].(int64)
		return eng.FileSystemPlugin.SetCounter(name, act, val, db)
	}
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return eng.FileSystemPlugin.ListCounter(filter, db)
}

// handler for: database.select
func DbSelect(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	db := cmd.Database
	return eng.FileSystemPlugin.BQLSearch(db, cmd.Args)
}

// handler for: database.set
func DbSet(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	db := cmd.Database
	return eng.FileSystemPlugin.BQLSet(db, cmd.Args)
}

// handler for: database.unset
func DbUnset(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) bytengine.Response {
	db := cmd.Database
	return eng.FileSystemPlugin.BQLUnset(db, cmd.Args)
}

func init() {
	bytengine.RegisterCommandHandler("database.newdir", DbNewDir)
	bytengine.RegisterCommandHandler("database.newfile", DbNewFile)
	bytengine.RegisterCommandHandler("database.listdir", DbListDir)
	bytengine.RegisterCommandHandler("database.rename", DbRename)
	bytengine.RegisterCommandHandler("database.move", DbMove)
	bytengine.RegisterCommandHandler("database.copy", DbCopy)
	bytengine.RegisterCommandHandler("database.delete", DbDelete)
	bytengine.RegisterCommandHandler("database.info", DbInfo)
	bytengine.RegisterCommandHandler("database.makepublic", DbMakePublic)
	bytengine.RegisterCommandHandler("database.makeprivate", DbMakePrivate)
	bytengine.RegisterCommandHandler("database.readfile", DbReadFile)
	bytengine.RegisterCommandHandler("database.modfile", DbModFile)
	bytengine.RegisterCommandHandler("database.deletebytes", DbDeleteBytes)
	bytengine.RegisterCommandHandler("database.counter", DbCounter)
	bytengine.RegisterCommandHandler("database.select", DbSelect)
	bytengine.RegisterCommandHandler("database.set", DbSet)
	bytengine.RegisterCommandHandler("database.unset", DbUnset)
}
