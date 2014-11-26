package core

import (
	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: user.new
func UserNew(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := bytengine.AuthPlugin.NewUser(usr, pw, false)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(true)
}

// handler for: user.all
func UserAll(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	rgx := "."
	val, ok := cmd.Options["regex"]
	if ok {
		rgx = val.(string)
	}
	users, err := bytengine.AuthPlugin.ListUser(rgx)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(users)
}

// handler for: user.about
func UserAbout(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	info, err := bytengine.AuthPlugin.UserInfo(usr)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(info)
}

// handler for: user.delete
func UserDelete(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	err := bytengine.AuthPlugin.RemoveUser(usr)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(true)
}

// handler for: user.passw
func UserPassw(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := bytengine.AuthPlugin.ChangeUserPassword(usr, pw)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(true)
}

// handler for: user.access
func UserAccess(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	err := bytengine.AuthPlugin.ChangeUserStatus(usr, grant)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(true)
}

// handler for: user.db
func UserDb(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	db := cmd.Args["database"].(string)
	err := bytengine.AuthPlugin.ChangeUserDbAccess(usr, db, grant)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}
	return bytengine.OKResponse(true)
}

// handler for: user.whoami
func UserWhoami(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	val := map[string]interface{}{
		"username":  user.Username,
		"databases": user.Databases,
		"root":      user.Root,
	}
	return bytengine.OKResponse(val)
}

func init() {
	bytengine.RegisterCommandHandler("user.new", UserNew)
	bytengine.RegisterCommandHandler("user.all", UserAll)
	bytengine.RegisterCommandHandler("user.about", UserAbout)
	bytengine.RegisterCommandHandler("user.delete", UserDelete)
	bytengine.RegisterCommandHandler("user.passw", UserPassw)
	bytengine.RegisterCommandHandler("user.access", UserAccess)
	bytengine.RegisterCommandHandler("user.db", UserDb)
	bytengine.RegisterCommandHandler("user.whoami", UserWhoami)
}
