package modules

import (
	"crypto/rand"
	"fmt"
	"crypto/sha1"
	"io"
	"strings"
	"errors"
	"regexp"
	"encoding/base64"
	"github.com/vmihailenco/redis"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type authToken struct {
	Username string
	Salt string
	Password string
	Active bool
	Databases []string
}

type SystemUser struct {
	Username string `json:"username"`
	Active bool `json:"active"`
	Databases []string `json:"databases"`
}

type ReqMode string

const (
	RootMode ReqMode = "root"
	UserMode ReqMode = "user"
	GuestMode ReqMode = "guest"
)

type AuthManager struct {
	config *Configuration
	mongo *mgo.Session
	redisMan *RedisManager
}

func NewAuthManager(c *Configuration, m *mgo.Session, r *RedisManager) *AuthManager {
	a := &AuthManager{
		config: c,
		mongo: m,
		redisMan: r,
	}
	return a
}

func (auth *AuthManager) Authenticate(usr, pw string) (ReqMode, error) {
	// check if root authentication
	_admin_usr := auth.config.General.Admin
	_admin_pw := auth.config.General.Password
	if usr == _admin_usr && pw == _admin_pw {
		return RootMode, nil
	}

	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	q := map[string]interface{}{"username":usr}
	var _token authToken
	err := col.Find(q).One(&_token)
	if err != nil {
		msg := fmt.Sprintf("authentication failed: user %s not found", usr)
		return GuestMode, errors.New(msg)
	}

	// check if user is active
	if !_token.Active {
		msg := fmt.Sprintf("authentication failed: user %s not active", usr)
		return GuestMode, errors.New(msg)
	}

	// check password using sha1
	h := sha1.New()
	io.WriteString(h, pw + _token.Salt)
	_encrypt_pw := fmt.Sprintf("%x", h.Sum(nil))
	if _encrypt_pw != _token.Password {
		msg := "authentication failed: invalid password"
		return GuestMode, errors.New(msg)
	}

	return UserMode, nil
}

func (auth *AuthManager) NewUser(usr, pw string) error {
	// usernames are lowercase
	usr = strings.ToLower(usr)

	// validate username and password
	err := auth.validateUsername(usr)
	if err != nil {
		return err
	}
	err = auth.validatePassword(pw)
	if err != nil {
		return err
	}

	// check if username available
	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	q := map[string]interface{}{"username":usr}
	count, e2 := col.Find(q).Count()
	if e2 != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, e2)
		return errors.New(msg)
	}
	if count > 0 {
		msg := fmt.Sprintf("user %s already exists", usr)
		return errors.New(msg)
	}

	// create salt
	c := 16
	b := make([]byte, c)
	n, e3 := io.ReadFull(rand.Reader, b)
	if n != len(b) || err != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, e3)
		return errors.New(msg)
	}
	_salt := base64.URLEncoding.EncodeToString(b)

	// create encrypted password
	h := sha1.New()
	io.WriteString(h, pw + _salt)
	_encrypt_pw := fmt.Sprintf("%x", h.Sum(nil))
	
	// create token
	_token := authToken{usr, _salt, _encrypt_pw, true, []string{}}
	e4 := col.Insert(&_token) 
	if err != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, e4)
		return errors.New(msg)
	}

	return nil
}

func (auth *AuthManager) ChangeUserPassword(usr, pw string) error {
	// validate password
	err := auth.validatePassword(pw)
	if err != nil {
		return err
	}

	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// create salt
	c := 16
	b := make([]byte, c)
	n, e3 := io.ReadFull(rand.Reader, b)
	if n != len(b) || e3 != nil {
		msg := fmt.Sprintf("user %s password couldn't be created:\n%s", usr, e3)
		return errors.New(msg)
	}
	_salt := base64.URLEncoding.EncodeToString(b)

	// create encrypted password
	h := sha1.New()
	io.WriteString(h, pw + _salt)
	_encrypt_pw := fmt.Sprintf("%x", h.Sum(nil))

	// build query
	q := map[string]interface{}{"username":usr}
	uq := bson.M{"$set":bson.M{"password":_encrypt_pw, "salt":_salt}}
	err = col.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("user %s password couldn't be created:\n%s", usr, err)
		return errors.New(msg)
	}

	return nil
}

func (auth *AuthManager) ChangeUserStatus(usr string, isactive bool) error {
	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	q := map[string]interface{}{"username":usr}
	uq := bson.M{"$set":bson.M{"active":isactive}}
	err := col.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("user %s status couldn't be updated:\n%s", usr, err)
		return errors.New(msg)
	}

	return nil
}

func (auth *AuthManager) ListUsers() ([]interface{}, error) {
	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	q := map[string]interface{}{}
	i := col.Find(q).Iter()
	res := []interface{}{}
	var usr SystemUser
	for i.Next(&usr) {
		var item = map[string]interface{}{"username":usr.Username, "active":usr.Active}
		res = append(res, item)
	}
	err := i.Err()
	if err != nil {
		msg := fmt.Sprintf("user list couldn't be retrieved:\n%s", err)
		return nil, errors.New(msg)
	}

	return res, nil
}

func (auth *AuthManager) ChangeUserDbAccess(usr, db string, grant bool) error {
	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	var uq bson.M
	if grant {
		uq = bson.M{"$addToSet":bson.M{"databases":db}}
	} else {
		uq = bson.M{"$pull":bson.M{"databases":db}}
	}
	q := map[string]interface{}{"username":usr}
	err := col.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("user %s database access couldn't be updated:\n%s", usr, err)
		return errors.New(msg)
	}

	return nil
}

func (auth *AuthManager) RemoveUser(usr string) error {
	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	q := map[string]interface{}{"username":usr}
	e := col.Remove(q)
	if e != nil {
		msg := fmt.Sprintf("couldn't remove user %s:\n%s", usr, e)
		return errors.New(msg)
	}

	return nil
}

func (auth *AuthManager) UserInfo(u string) (SystemUser, error) {
	var usr SystemUser
	// get collection
	col := auth.mongo.DB(auth.config.Bfs.SystemDb).C(auth.config.Bfs.SecurityCol)

	// build query
	q := map[string]interface{}{"username":u}
	e := col.Find(q).One(&usr)
	if e != nil {
		msg := fmt.Sprintf("couldn't get info for user %s:\n%s", u, e)
		return usr, errors.New(msg)
	}

	return usr, nil
}

func (auth *AuthManager) NewSession(mode ReqMode, usr string) (string, error) {
	// create session key
	_sessionkey, err := makeUUID()
	if err != nil {
		msg := fmt.Sprintf("session couldn't be created.\n%s", err)
		return "", errors.New(msg)
	}
	// redis connection
	rclient := auth.redisMan.DbConnect["auth"]
	_timeout := auth.config.Bfs.SessionTimeout
	
	// create session and set timeout
	_, e := rclient.Pipelined(func(c *redis.PipelineClient){
		c.RPush(_sessionkey, usr)
        c.RPush(_sessionkey, fmt.Sprint(mode))
		c.Expire(_sessionkey, int64(_timeout))
	})
    if e != nil {
		msg := fmt.Sprintf("session couldn't be created.\n%s", e)
		return "", errors.New(msg)
	}

	return _sessionkey, nil
}

func (auth *AuthManager) GetSession(sid string) (string, ReqMode, error) {
	if sid == "" {
		// guest mode
		return "guest", GuestMode, nil
	}
	
	// redis connection
	rclient := auth.redisMan.DbConnect["auth"]
	_timeout := auth.config.Bfs.SessionTimeout

	// get username and access mode
	_data := rclient.LRange(sid, 0, 1)
	e := _data.Err()
	if e != nil {
		msg := fmt.Sprintf("session couldn't be retrieved.\n%s", e)
		return "guest", GuestMode, errors.New(msg)
	}

	if len(_data.Val()) < 2 {
		msg := "session couldn't be retrieved:\ninvalid session key or data"
		return "guest", GuestMode, errors.New(msg)
	}
    
    if _data.Val()[1] != fmt.Sprint(RootMode) {
        // get user info and check account status
        _info, e2 := auth.UserInfo(_data.Val()[0])
        if e2 != nil {
            // delete session
            rclient.Del(sid)
            msg := fmt.Sprintf("session couldn't be retrieved.\n%s", e2)
            return "guest", GuestMode, errors.New(msg)
        }
        if !_info.Active {
            // delete session
            rclient.Del(sid)
            msg := "session couldn't be retrieved: user account inactive"
            return "guest", GuestMode, errors.New(msg)
        }
    }

	// reset sesson timeout
	_update := rclient.Expire(sid, int64(_timeout))
	e = _update.Err()
	if e != nil || _update.Val() == false{
		msg := fmt.Sprintf("session couldn't be retrieved.\n%s", e)
		return "guest", GuestMode, errors.New(msg)
	}

	// get access mode
	var mode ReqMode	
	switch _data.Val()[1] {
	case fmt.Sprint(RootMode):
		mode = RootMode
	case fmt.Sprint(UserMode):
		mode = UserMode
	default:
		mode = GuestMode
	}
	// username
	username := _data.Val()[0]

	return username, mode, nil
}

func (auth *AuthManager) validatePassword(pw string) error {
	// regex verification
	r, err := regexp.Compile("^\\w{8,}$")
	if err != nil {
		return err
	}
	if r.MatchString(pw) {
		return nil
	}
	msg := "password isn't valid."
	return errors.New(msg)
}

func (auth *AuthManager) validateUsername(usr string) error {
	if usr == "guest" {
		return errors.New("username guest already taken")
	}

	// regex verification
	r, err := regexp.Compile("^[a-z]{1}([_]{0,1}[a-zA-Z0-9]{1,})+$")
	if err != nil {
		return err
	}
	if r.MatchString(usr) {
		return nil
	}
	msg := "username isn't valid."
	return errors.New(msg)
}