package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/astaxie/beego/cache"
	"github.com/johnwilson/bytengine/auth"
	"github.com/johnwilson/bytengine/bfs"
	"github.com/johnwilson/bytengine/core"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: user.new
func handler1(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := e.AuthManager.NewUser(usr, pw, false)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.all
func handler2(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	rgx := cmd.Args["regex"].(string)
	users, err := e.AuthManager.ListUser(rgx)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(users)
}

// handler for: user.about
func handler3(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	usr := cmd.Args["username"].(string)
	info, err := e.AuthManager.UserInfo(usr)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(info)
}

// handler for: user.delete
func handler4(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	usr := cmd.Args["username"].(string)
	err := e.AuthManager.RemoveUser(usr)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.passw
func handler5(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := e.AuthManager.ChangeUserPassword(usr, pw)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.access
func handler6(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	err := e.AuthManager.ChangeUserStatus(usr, grant)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	return bfs.OKResponse(true)
}

// handler for: user.db
func handler7(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
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
func handler8(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	val := map[string]interface{}{
		"username":  user.Username,
		"databases": user.Databases,
		"isroot":    user.Root,
	}
	return bfs.OKResponse(val)
}

// handler for: server.listdb
func handler9(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	filter := cmd.Args["regex"].(string)
	return e.BFSManager.ListDatabase(filter)
}

// handler for: server.newdb
func handler10(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	db := cmd.Args["database"].(string)
	return e.BFSManager.CreateDatabase(db)
}

// handler for: server.init
func handler11(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	return e.BFSManager.ClearAll()
}

// handler for: server.dropdb
func handler12(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	db := cmd.Args["database"].(string)
	return e.BFSManager.DropDatabase(db)
}

// handler for: database.newdir
func handler13(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.NewDir(path, db)
}

// handler for: database.newfile
func handler14(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	return e.BFSManager.NewFile(path, db, data)
}

// handler for: database.listdir
func handler15(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	filter := cmd.Args["regex"].(string)
	db := cmd.Database
	return e.BFSManager.ListDir(path, filter, db)
}

// handler for: database.rename
func handler16(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	name := cmd.Args["name"].(string)
	db := cmd.Database
	return e.BFSManager.Rename(path, name, db)
}

// handler for: database.move
func handler17(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	return e.BFSManager.Move(path, to, db)
}

// handler for: database.copy
func handler18(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	to := cmd.Args["to"].(string)
	db := cmd.Database
	return e.BFSManager.Copy(path, to, db)
}

// handler for: database.delete
func handler19(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.Delete(path, db)
}

// handler for: database.info
func handler20(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.Info(path, db)
}

// handler for: database.makepublic
func handler21(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.FileAccess(path, db, false)
}

// handler for: database.makeprivate
func handler22(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.FileAccess(path, db, true)
}

// handler for: database.readfile
func handler23(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	fields := cmd.Args["fields"].([]string)
	db := cmd.Database
	return e.BFSManager.ReadJson(path, db, fields)
}

// handler for: database.modfile
func handler24(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	data := cmd.Args["data"].(map[string]interface{})
	db := cmd.Database
	return e.BFSManager.UpdateJson(path, db, data)
}

// handler for: database.deletebytes
func handler25(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	path := cmd.Args["path"].(string)
	db := cmd.Database
	return e.BFSManager.DeleteBytes(path, db)
}

// handler for: database.counter
func handler26(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	act := cmd.Args["action"].(string)
	db := cmd.Database
	if act != "list" {
		name := cmd.Args["name"].(string)
		val := cmd.Args["value"].(int64)
		return e.BFSManager.SetCounter(name, act, val, db)
	}
	filter := cmd.Args["regex"].(string)
	return e.BFSManager.ListCounter(filter, db)
}

// handler for: database.select
func handler27(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	db := cmd.Database
	return e.BFSManager.BQLSearch(db, cmd.Args)
}

// handler for: database.set
func handler28(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	db := cmd.Database
	return e.BFSManager.BQLSet(db, cmd.Args)
}

// handler for: database.unset
func handler29(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	db := cmd.Database
	return e.BFSManager.BQLUnset(db, cmd.Args)
}

// handler for: login
func loginHandler(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)

	ok := e.AuthManager.Authenticate(usr, pw)
	if !ok {
		return bfs.ErrorResponse(fmt.Errorf("Authentication failed"))
	}

	key := auth.GenerateRandomKey(16)
	if len(key) == 0 {
		return bfs.ErrorResponse(fmt.Errorf("Token creation failed"))
	}

	token := fmt.Sprintf("%x", key)
	err := e.CacheManager.Put(token, usr, 60)
	if err != nil {
		return bfs.ErrorResponse(fmt.Errorf("Token creation failed"))
	}

	return bfs.OKResponse(token)
}

// handler for: upload ticket
func uploadTicketHandler(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	// check if user is anonymous
	if user == nil {
		return bfs.ErrorResponse(fmt.Errorf("Authorization required"))
	}

	db := cmd.Database
	path := cmd.Args["path"].(string)
	// check if path exists
	r := e.BFSManager.Info(path, db)
	if !r.Success() {
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
	err = e.CacheManager.Put(ticket, string(b), 300)
	if err != nil {
		return bfs.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}

	return bfs.OKResponse(ticket)
}

// handler for: writebytes
func writebytesHandler(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	ticket := cmd.Args["ticket"].(string)
	tmpfile := cmd.Args["tmpfile"].(string)
	// check if ticket exists
	raw := e.CacheManager.Get(ticket)
	if raw == nil {
		os.Remove(tmpfile)
		return bfs.ErrorResponse(fmt.Errorf("Invalid ticket"))
	}
	content := cache.GetString(raw)
	// get ticket value
	var val struct {
		Database string
		Path     string
	}
	b := []byte(content)
	err := json.Unmarshal(b, &val)
	if err != nil {
		os.Remove(tmpfile)
		return bfs.ErrorResponse(fmt.Errorf("Ticket data invalid"))
	}

	r := e.BFSManager.WriteBytes(val.Path, tmpfile, val.Database)
	os.Remove(tmpfile)
	return r
}

// handler for: readbytes
func readbytesHandler(cmd dsl.Command, user *auth.User, e *core.Engine) bfs.BFSResponse {
	// check if user is anonymous
	if user == nil {
		return bfs.ErrorResponse(fmt.Errorf("Authorization required"))
	}

	db := cmd.Database
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	r := e.BFSManager.ReadBytes(path, db)
	if !r.Success() {
		return r
	}

	// get file pointer
	bstoreid := r.Data().(string)
	err := e.BStoreManager.Read(db, bstoreid, w)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(true)
}
