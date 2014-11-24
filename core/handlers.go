package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/johnwilson/bytengine/auth"
	"github.com/johnwilson/bytengine/dsl"
	bfs "github.com/johnwilson/bytengine/filesystem"
)

// handler for: user.new
func userNew(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := e.AuthManager.NewUser(usr, pw, false)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.all
func userAll(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	rgx := "."
	val, ok := cmd.Options["regex"]
	if ok {
		rgx = val.(string)
	}
	users, err := e.AuthManager.ListUser(rgx)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(users)
}

// handler for: user.about
func userAbout(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	info, err := e.AuthManager.UserInfo(usr)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(info)
}

// handler for: user.delete
func userDelete(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	err := e.AuthManager.RemoveUser(usr)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.passw
func userPassw(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := e.AuthManager.ChangeUserPassword(usr, pw)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.access
func userAccess(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	err := e.AuthManager.ChangeUserStatus(usr, grant)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.db
func userDb(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	db := cmd.Args["database"].(string)
	err := e.AuthManager.ChangeUserDbAccess(usr, db, grant)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.whoami
func userWhoami(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	val := map[string]interface{}{
		"username":  user.Username,
		"databases": user.Databases,
		"root":      user.Root,
	}
	return bfs.OKResponse(val)
}

// handler for: server.listdb
func serverListDb(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return e.BFSManager.ListDatabase(filter)
}

// handler for: server.newdb
func serverNewDb(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	db := cmd.Args["database"].(string)
	return e.BFSManager.CreateDatabase(db)
}

// handler for: server.init
func serverInit(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	return e.BFSManager.ClearAll()
}

// handler for: server.dropdb
func serverDropDb(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	db := cmd.Args["database"].(string)
	return e.BFSManager.DropDatabase(db)
}

// handler for: database.newdir
func dbNewDir(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.NewDir(path, db)
}

// handler for: database.newfile
func dbNewFile(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	return e.BFSManager.NewFile(path, db, data)
}

// handler for: database.listdir
func dbListDir(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	db := cmd.Database
	return e.BFSManager.ListDir(path, filter, db)
}

// handler for: database.rename
func dbRename(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	name := cmd.Args["name"].(string)
	db := cmd.Database
	return e.BFSManager.Rename(path, name, db)
}

// handler for: database.move
func dbMove(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	return e.BFSManager.Move(path, to, db)
}

// handler for: database.copy
func dbCopy(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	return e.BFSManager.Copy(path, to, db)
}

// handler for: database.delete
func dbDelete(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.Delete(path, db)
}

// handler for: database.info
func dbInfo(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.Info(path, db)
}

// handler for: database.makepublic
func dbMakePublic(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.FileAccess(path, db, false)
}

// handler for: database.makeprivate
func dbMakePrivate(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.FileAccess(path, db, true)
}

// handler for: database.readfile
func dbReadFile(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	fields := cmd.Args["fields"].([]string)
	db := cmd.Database
	return e.BFSManager.ReadJson(path, db, fields)
}

// handler for: database.modfile
func dbModFile(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	return e.BFSManager.UpdateJson(path, db, data)
}

// handler for: database.deletebytes
func dbDeleteBytes(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.DeleteBytes(path, db)
}

// handler for: database.counter
func dbCounter(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	act := cmd.Args["action"].(string)
	db := cmd.Database
	if act != "list" {
		name := cmd.Args["name"].(string)
		val := cmd.Args["value"].(int64)
		return e.BFSManager.SetCounter(name, act, val, db)
	}
	filter := "."
	val, ok := cmd.Options["regex"]
	if ok {
		filter = val.(string)
	}
	return e.BFSManager.ListCounter(filter, db)
}

// handler for: database.select
func dbSelect(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	db := cmd.Database
	return e.BFSManager.BQLSearch(db, cmd.Args)
}

// handler for: database.set
func dbSet(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	db := cmd.Database
	return e.BFSManager.BQLSet(db, cmd.Args)
}

// handler for: database.unset
func dbUnset(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	db := cmd.Database
	return e.BFSManager.BQLUnset(db, cmd.Args)
}

// handler for: login
func loginHandler(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	duration := cmd.Args["duration"].(int64)

	ok := e.AuthManager.Authenticate(usr, pw)
	if !ok {
		return bfs.ErrorResponse(fmt.Errorf("Authentication failed"))
	}

	key := auth.GenerateRandomKey(16)
	if len(key) == 0 {
		return bfs.ErrorResponse(fmt.Errorf("Token creation failed"))
	}

	token := fmt.Sprintf("%x", key)
	err := e.StateManager.TokenSet(token, usr, 60*duration)
	if err != nil {
		return bfs.ErrorResponse(fmt.Errorf("Token creation failed"))
	}

	return bfs.OKResponse(token)
}

// handler for: upload ticket
func uploadTicketHandler(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	// check if user is anonymous
	if user == nil {
		return bfs.ErrorResponse(fmt.Errorf("Authorization required"))
	}

	db := cmd.Database
	path := cmd.Args["path"].(string)
	duration := cmd.Args["duration"].(int64)
	// check if path exists
	r := e.BFSManager.Info(path, db)
	if r.Status != bfs.OK {
		return r
	}
	// create ticket
	key := auth.GenerateRandomKey(16)
	if len(key) == 0 {
		return bfs.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}

	ticket := fmt.Sprintf("%x", key)
	val := map[string]string{
		"database": db,
		"path":     path,
	}
	b, err := json.Marshal(val)
	if err != nil {
		return bfs.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}
	err = e.StateManager.CacheSet(ticket, string(b), 60*duration)
	if err != nil {
		return bfs.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}

	return bfs.OKResponse(ticket)
}

// handler for: writebytes
func writebytesHandler(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	ticket := cmd.Args["ticket"].(string)
	tmpfile := cmd.Args["tmpfile"].(string)
	// get ticket
	content, err := e.StateManager.CacheGet(ticket)
	if err != nil {
		os.Remove(tmpfile)
		return bfs.ErrorResponse(fmt.Errorf("Invalid ticket"))
	}
	// get ticket value
	var val struct {
		Database string
		Path     string
	}
	b := []byte(content)
	err = json.Unmarshal(b, &val)
	if err != nil {
		os.Remove(tmpfile)
		return bfs.ErrorResponse(fmt.Errorf("Ticket data invalid"))
	}

	r := e.BFSManager.WriteBytes(val.Path, tmpfile, val.Database)
	os.Remove(tmpfile)
	return r
}

// handler for: readbytes
func readbytesHandler(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	// check if user is anonymous
	if user == nil {
		return bfs.ErrorResponse(fmt.Errorf("Authorization required"))
	}

	db := cmd.Database
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	r := e.BFSManager.ReadBytes(path, db)
	if r.Status != bfs.OK {
		return r
	}

	// get file pointer
	bstoreid := r.Data.(string)
	err := e.BStoreManager.Read(db, bstoreid, w)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(true)
}

// handler for: direct access
func direcaccessHandler(cmd dsl.Command, user *auth.User, e *Engine) bfs.Response {
	db := cmd.Args["database"].(string)
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	layer := cmd.Args["layer"].(string)
	r := e.BFSManager.DirectAccess(path, db, layer)
	if r.Status != bfs.OK {
		return r
	}

	switch layer {
	case "json":
		// write json
		_, err := w.Write(r.JSON())
		if err != nil {
			return bfs.ErrorResponse(err)
		}
	case "bytes":
		// get file pointer
		bstoreid := r.Data.(string)
		err := e.BStoreManager.Read(db, bstoreid, w)
		if err != nil {
			return bfs.ErrorResponse(err)
		}
	}
	return bfs.OKResponse(true)
}
