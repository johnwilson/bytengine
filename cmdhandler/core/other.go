package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/dsl"
)

// handler for: login
func LoginHandler(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	duration := cmd.Args["duration"].(int64)

	ok := bytengine.AuthPlugin.Authenticate(usr, pw)
	if !ok {
		return bytengine.ErrorResponse(fmt.Errorf("Authentication failed"))
	}

	key := bytengine.GenerateRandomKey(16)
	if len(key) == 0 {
		return bytengine.ErrorResponse(fmt.Errorf("Token creation failed"))
	}

	token := fmt.Sprintf("%x", key)
	err := bytengine.StateStorePlugin.TokenSet(token, usr, 60*duration)
	if err != nil {
		return bytengine.ErrorResponse(fmt.Errorf("Token creation failed"))
	}

	return bytengine.OKResponse(token)
}

// handler for: upload ticket
func UploadTicketHandler(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	// check if user is anonymous
	if user == nil {
		return bytengine.ErrorResponse(fmt.Errorf("Authorization required"))
	}

	db := cmd.Database
	path := cmd.Args["path"].(string)
	duration := cmd.Args["duration"].(int64)
	// check if path exists
	r := bytengine.FileSystemPlugin.Info(path, db)
	if r.Status != bytengine.OK {
		return r
	}
	// create ticket
	key := bytengine.GenerateRandomKey(16)
	if len(key) == 0 {
		return bytengine.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}

	ticket := fmt.Sprintf("%x", key)
	val := map[string]string{
		"database": db,
		"path":     path,
	}
	b, err := json.Marshal(val)
	if err != nil {
		return bytengine.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}
	err = bytengine.StateStorePlugin.CacheSet(ticket, string(b), 60*duration)
	if err != nil {
		return bytengine.ErrorResponse(fmt.Errorf("Ticket creation failed"))
	}

	return bytengine.OKResponse(ticket)
}

// handler for: writebytes
func WritebytesHandler(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	ticket := cmd.Args["ticket"].(string)
	tmpfile := cmd.Args["tmpfile"].(string)
	// get ticket
	content, err := bytengine.StateStorePlugin.CacheGet(ticket)
	if err != nil {
		os.Remove(tmpfile)
		return bytengine.ErrorResponse(fmt.Errorf("Invalid ticket"))
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
		return bytengine.ErrorResponse(fmt.Errorf("Ticket data invalid"))
	}

	r := bytengine.FileSystemPlugin.WriteBytes(val.Path, tmpfile, val.Database)
	os.Remove(tmpfile)
	return r
}

// handler for: readbytes
func ReadbytesHandler(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	// check if user is anonymous
	if user == nil {
		return bytengine.ErrorResponse(fmt.Errorf("Authorization required"))
	}

	db := cmd.Database
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	r := bytengine.FileSystemPlugin.ReadBytes(path, db)
	if r.Status != bytengine.OK {
		return r
	}

	// get file pointer
	bstoreid := r.Data.(string)
	err := bytengine.ByteStorePlugin.Read(db, bstoreid, w)
	if err != nil {
		return bytengine.ErrorResponse(err)
	}

	return bytengine.OKResponse(true)
}

// handler for: direct access
func DirecaccessHandler(cmd dsl.Command, user *bytengine.User) bytengine.Response {
	db := cmd.Args["database"].(string)
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	layer := cmd.Args["layer"].(string)
	r := bytengine.FileSystemPlugin.DirectAccess(path, db, layer)
	if r.Status != bytengine.OK {
		return r
	}

	switch layer {
	case "json":
		// write json
		_, err := w.Write(r.JSON())
		if err != nil {
			return bytengine.ErrorResponse(err)
		}
	case "bytes":
		// get file pointer
		bstoreid := r.Data.(string)
		err := bytengine.ByteStorePlugin.Read(db, bstoreid, w)
		if err != nil {
			return bytengine.ErrorResponse(err)
		}
	}
	return bytengine.OKResponse(true)
}

func init() {
	bytengine.RegisterCommandHandler("login", LoginHandler)
	bytengine.RegisterCommandHandler("uploadticket", UploadTicketHandler)
	bytengine.RegisterCommandHandler("writebytes", WritebytesHandler)
	bytengine.RegisterCommandHandler("readbytes", ReadbytesHandler)
	bytengine.RegisterCommandHandler("directaccess", DirecaccessHandler)
}
