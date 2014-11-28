package base

import (
	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: user.new
func UserNew(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := eng.Authentication.NewUser(usr, pw, false)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(true), nil
}

// handler for: user.all
func UserAll(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	rgx := "."
	val, ok := cmd.Options["regex"]
	if ok {
		rgx = val.(string)
	}
	users, err := eng.Authentication.ListUser(rgx)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(users), nil
}

// handler for: user.about
func UserAbout(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	info, err := eng.Authentication.UserInfo(usr)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(info), nil
}

// handler for: user.delete
func UserDelete(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	err := eng.Authentication.RemoveUser(usr)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(true), nil
}

// handler for: user.passw
func UserPassw(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	err := eng.Authentication.ChangeUserPassword(usr, pw)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(true), nil
}

// handler for: user.access
func UserAccess(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	err := eng.Authentication.ChangeUserStatus(usr, grant)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(true), nil
}

// handler for: user.db
func UserDb(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	grant := cmd.Args["grant"].(bool)
	db := cmd.Args["database"].(string)
	err := eng.Authentication.ChangeUserDbAccess(usr, db, grant)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}
	return bytengine.OKResponse(true), nil
}

// handler for: user.whoami
func UserWhoami(cmd dsl.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	val := map[string]interface{}{
		"username":  user.Username,
		"databases": user.Databases,
		"root":      user.Root,
	}
	return bytengine.OKResponse(val), nil
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
