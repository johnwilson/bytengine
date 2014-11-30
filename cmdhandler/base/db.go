package base

import (
	"github.com/johnwilson/bytengine"
)

// handler for: database.newdir
func DbNewDir(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	if err := eng.FileSystem.NewDir(path, db); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.newfile
func DbNewFile(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	if err := eng.FileSystem.NewFile(path, db, data); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.listdir
func DbListDir(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	db := cmd.Database
	return eng.FileSystem.ListDir(path, filter, db)
}

// handler for: database.rename
func DbRename(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	name := cmd.Args["name"].(string)
	db := cmd.Database
	if err := eng.FileSystem.Rename(path, name, db); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.move
func DbMove(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	if err := eng.FileSystem.Move(path, to, db); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.copy
func DbCopy(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	if err := eng.FileSystem.Copy(path, to, db); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.delete
func DbDelete(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	if err := eng.FileSystem.Delete(path, db); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.info
func DbInfo(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return eng.FileSystem.Info(path, db)
}

// handler for: database.makepublic
func DbMakePublic(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	if err := eng.FileSystem.FileAccess(path, db, false); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.makeprivate
func DbMakePrivate(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	if err := eng.FileSystem.FileAccess(path, db, true); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.readfile
func DbReadFile(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	fields := cmd.Args["fields"].([]string)
	db := cmd.Database
	return eng.FileSystem.ReadJson(path, db, fields)
}

// handler for: database.modfile
func DbModFile(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	if err := eng.FileSystem.UpdateJson(path, db, data); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.deletebytes
func DbDeleteBytes(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	if err := eng.FileSystem.DeleteBytes(path, db); err != nil {
		return false, err
	}
	return true, nil
}

// handler for: database.counter
func DbCounter(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	act := cmd.Args["action"].(string)
	db := cmd.Database
	if act != "list" {
		name := cmd.Args["name"].(string)
		val := cmd.Args["value"].(int64)
		return eng.FileSystem.SetCounter(name, act, val, db)
	}
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return eng.FileSystem.ListCounter(filter, db)
}

// handler for: database.select
func DbSelect(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	db := cmd.Database
	return eng.FileSystem.BQLSearch(db, cmd.Args)
}

// handler for: database.set
func DbSet(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	db := cmd.Database
	return eng.FileSystem.BQLSet(db, cmd.Args)
}

// handler for: database.unset
func DbUnset(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (interface{}, error) {
	db := cmd.Database
	return eng.FileSystem.BQLUnset(db, cmd.Args)
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
