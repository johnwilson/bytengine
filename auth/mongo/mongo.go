package mongo

import (
	//"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/johnwilson/bytengine"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
)

type Config struct {
	Addresses    []string      `json:"addresses"`
	Timeout      time.Duration `json:"timeout"`
	AuthDatabase string        `json:"authdb"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
}

func NewAuthentication() *Authentication {
	return &Authentication{}
}

const (
	AUTH_DATABASE   = "bytengine_auth"
	AUTH_COLLECTION = "users"
)

type Authentication struct {
	session  *mgo.Session
	database string
}

type authToken struct {
	Username  string   `bson:"username"`
	Password  string   `bson:"password"`
	Active    bool     `bson:"active"`
	Databases []string `bson:"databases"`
	Root      bool     `bson:"root"`
}

/*
============================================================================
    Private Methods
============================================================================
*/

func (m *Authentication) getCollection() *mgo.Collection {
	return m.session.DB(AUTH_DATABASE).C(AUTH_COLLECTION)
}

/*
============================================================================
    Auth Interface Methods
============================================================================
*/

func (m *Authentication) Start(config string) error {
	var c Config
	err := json.Unmarshal([]byte(config), &c)
	if err != nil {
		return err
	}

	info := &mgo.DialInfo{
		Addrs:    c.Addresses,
		Timeout:  c.Timeout * time.Second,
		Database: c.AuthDatabase,
		Username: c.Username,
		Password: c.Password,
	}
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		return err
	}
	m.session = session
	m.database = AUTH_DATABASE
	return nil
}

func (m *Authentication) ClearAll() error {
	names, err := m.session.DatabaseNames()
	if err != nil {
		return err
	}
	exists := false // database exists
	for _, i := range names {
		if i == AUTH_DATABASE {
			exists = true
			break
		}
	}
	if !exists {
		return nil
	}

	col := m.getCollection()
	err = col.DropCollection()
	if err != nil {
		return err
	}

	return nil
}

func (m *Authentication) Authenticate(usr, pw string) bool {
	// get collection
	col := m.getCollection()

	// build query
	q := map[string]interface{}{"username": usr, "active": true}
	var _token authToken
	err := col.Find(q).One(&_token)
	if err != nil {
		return false
	}

	if ok := bytengine.ValidatePassword([]byte(_token.Password), []byte(pw)); ok {
		return true
	}

	return false
}

func (m *Authentication) NewUser(usr, pw string, root bool) error {
	// usernames are lowercase
	usr = strings.ToLower(usr)

	// check username and password
	err := bytengine.CheckUsername(usr)
	if err != nil {
		return err
	}
	err = bytengine.CheckPassword(pw)
	if err != nil {
		return err
	}

	// check if username available
	// get collection
	col := m.getCollection()

	// build query
	q := map[string]interface{}{"username": usr}
	count, e2 := col.Find(q).Count()
	if e2 != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, e2)
		return errors.New(msg)
	}
	if count > 0 {
		msg := fmt.Sprintf("user %s already exists", usr)
		return errors.New(msg)
	}

	encrypt_pw, err := bytengine.PasswordEncrypt(pw)
	if err != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, err)
		return errors.New(msg)
	}

	// create token
	_token := authToken{
		usr,
		string(encrypt_pw),
		true,
		[]string{},
		root,
	}
	e4 := col.Insert(&_token)
	if err != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, e4)
		return errors.New(msg)
	}

	return nil
}

func (m *Authentication) ChangeUserPassword(usr, pw string) error {
	// validate password
	err := bytengine.CheckPassword(pw)
	if err != nil {
		return err
	}

	// get collection
	col := m.getCollection()

	encrypt_pw, err := bytengine.PasswordEncrypt(pw)
	if err != nil {
		msg := fmt.Sprintf("user %s couldn't be created:\n%s", usr, err)
		return errors.New(msg)
	}

	// build query
	q := map[string]interface{}{"username": usr}
	uq := bson.M{"$set": bson.M{"password": encrypt_pw}}
	err = col.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("user %s password couldn't be created:\n%s", usr, err)
		return errors.New(msg)
	}

	return nil
}

func (m *Authentication) ChangeUserStatus(usr string, isactive bool) error {
	// get collection
	col := m.getCollection()

	// build query
	q := map[string]interface{}{"username": usr}
	uq := bson.M{"$set": bson.M{"active": isactive}}
	err := col.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("user %s status couldn't be updated:\n%s", usr, err)
		return errors.New(msg)
	}

	return nil
}

func (m *Authentication) ListUser(rgx string) ([]string, error) {
	// get collection
	col := m.getCollection()

	// build query
	qre := bson.RegEx{Pattern: rgx, Options: "i"} // case insensitive regex
	q := bson.M{"username": bson.M{"$regex": qre}}

	i := col.Find(q).Iter()
	res := []string{}
	var usr bytengine.User
	for i.Next(&usr) {
		res = append(res, usr.Username)
	}
	err := i.Err()
	if err != nil {
		msg := fmt.Sprintf("user list couldn't be retrieved:\n%s", err)
		return nil, errors.New(msg)
	}

	return res, nil
}

func (m *Authentication) ChangeUserDbAccess(usr, db string, grant bool) error {
	// get collection
	col := m.getCollection()

	// build query
	var uq bson.M
	if grant {
		uq = bson.M{"$addToSet": bson.M{"databases": db}}
	} else {
		uq = bson.M{"$pull": bson.M{"databases": db}}
	}
	q := map[string]interface{}{"username": usr}
	err := col.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("user %s database access couldn't be updated:\n%s", usr, err)
		return errors.New(msg)
	}

	return nil
}

func (m *Authentication) HasDbAccess(usr, db string) bool {
	// get collection
	col := m.getCollection()

	// build query
	q := map[string]interface{}{"username": usr, "databases": db}
	count, e2 := col.Find(q).Count()
	if e2 != nil {
		return false
	}
	if count == 1 {
		return true
	}

	return false
}

func (m *Authentication) RemoveUser(usr string) error {
	// get collection
	col := m.getCollection()

	// build query
	q := map[string]interface{}{"username": usr}
	e := col.Remove(q)
	if e != nil {
		msg := fmt.Sprintf("couldn't remove user %s:\n%s", usr, e)
		return errors.New(msg)
	}

	return nil
}

func (m *Authentication) UserInfo(u string) (*bytengine.User, error) {
	// get collection
	col := m.getCollection()

	// build query
	q := map[string]interface{}{"username": u}

	var usr bytengine.User
	e := col.Find(q).One(&usr)
	if e != nil {
		msg := fmt.Sprintf("couldn't get info for user %s:\n%s", u, e)
		return nil, errors.New(msg)
	}

	return &usr, nil
}

func init() {
	bytengine.RegisterAuthentication("mongodb", NewAuthentication())
}
