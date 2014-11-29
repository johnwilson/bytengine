package base

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/auth"
)

const (
	KeyStrength = 16 // random key strength
)

// handler for: login
func LoginHandler(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	usr := cmd.Args["username"].(string)
	pw := cmd.Args["password"].(string)
	duration := cmd.Args["duration"].(int64)

	ok := eng.Authentication.Authenticate(usr, pw)
	if !ok {
		err := fmt.Errorf("Authentication failed")
		return bytengine.ErrorResponse(err), err
	}

	key := auth.GenerateRandomKey(KeyStrength)
	if len(key) == 0 {
		err := fmt.Errorf("Token creation failed")
		return bytengine.ErrorResponse(err), err
	}

	token := fmt.Sprintf("%x", key)
	err := eng.StateStore.TokenSet(token, usr, 60*duration)
	if err != nil {
		err := fmt.Errorf("Token creation failed")
		return bytengine.ErrorResponse(err), err
	}

	return bytengine.OKResponse(token), nil
}

// handler for: upload ticket
func UploadTicketHandler(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	// check if user is anonymous
	if user == nil {
		err := fmt.Errorf("Authorization required")
		return bytengine.ErrorResponse(err), err
	}

	db := cmd.Database
	path := cmd.Args["path"].(string)
	duration := cmd.Args["duration"].(int64)
	// check if path exists
	r, err := eng.FileSystem.Info(path, db)
	if err != nil {
		return r, err
	}
	// create ticket
	key := auth.GenerateRandomKey(KeyStrength)
	if len(key) == 0 {
		err := fmt.Errorf("Ticket creation failed")
		return bytengine.ErrorResponse(err), err
	}

	ticket := fmt.Sprintf("%x", key)
	val := map[string]string{
		"database": db,
		"path":     path,
	}
	b, err := json.Marshal(val)
	if err != nil {
		err := fmt.Errorf("Ticket creation failed")
		return bytengine.ErrorResponse(err), err
	}
	err = eng.StateStore.CacheSet(ticket, string(b), 60*duration)
	if err != nil {
		err := fmt.Errorf("Ticket creation failed")
		return bytengine.ErrorResponse(err), err
	}

	return bytengine.OKResponse(ticket), nil
}

// handler for: writebytes
func WritebytesHandler(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	ticket := cmd.Args["ticket"].(string)
	tmpfile := cmd.Args["tmpfile"].(string)
	// get ticket
	content, err := eng.StateStore.CacheGet(ticket)
	if err != nil {
		os.Remove(tmpfile)
		err := fmt.Errorf("Invalid ticket")
		return bytengine.ErrorResponse(err), err
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
		err := fmt.Errorf("Ticket data invalid")
		return bytengine.ErrorResponse(err), err
	}

	r, err := eng.FileSystem.WriteBytes(val.Path, tmpfile, val.Database)
	os.Remove(tmpfile)
	return r, err
}

// handler for: readbytes
func ReadbytesHandler(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	// check if user is anonymous
	if user == nil {
		err := fmt.Errorf("Authorization required")
		return bytengine.ErrorResponse(err), err
	}

	db := cmd.Database
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	r, err := eng.FileSystem.ReadBytes(path, db)
	if err != nil {
		return r, err
	}

	// get file pointer
	bstoreid := r.Data.(string)
	err = eng.ByteStore.Read(db, bstoreid, w)
	if err != nil {
		return bytengine.ErrorResponse(err), err
	}

	return bytengine.OKResponse(true), nil
}

// handler for: direct access
func DirecaccessHandler(cmd bytengine.Command, user *bytengine.User, eng *bytengine.Engine) (bytengine.Response, error) {
	db := cmd.Args["database"].(string)
	w := cmd.Args["writer"].(io.Writer)
	path := cmd.Args["path"].(string)
	layer := cmd.Args["layer"].(string)
	r, err := eng.FileSystem.DirectAccess(path, db, layer)
	if err != nil {
		return r, err
	}

	switch layer {
	case "json":
		// write json
		_, err := w.Write(r.JSON())
		if err != nil {
			return bytengine.ErrorResponse(err), err
		}
	case "bytes":
		// get file pointer
		bstoreid := r.Data.(string)
		err := eng.ByteStore.Read(db, bstoreid, w)
		if err != nil {
			return bytengine.ErrorResponse(err), err
		}
	}
	return bytengine.OKResponse(true), nil
}

func init() {
	bytengine.RegisterCommandHandler("login", LoginHandler)
	bytengine.RegisterCommandHandler("uploadticket", UploadTicketHandler)
	bytengine.RegisterCommandHandler("writebytes", WritebytesHandler)
	bytengine.RegisterCommandHandler("readbytes", ReadbytesHandler)
	bytengine.RegisterCommandHandler("directaccess", DirecaccessHandler)
}
