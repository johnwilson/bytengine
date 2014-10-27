package auth

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"regexp"
)

type User struct {
	Username  string   `json:"username"`
	Active    bool     `json:"active"`
	Databases []string `json:"databases"`
	Root      bool     `json:"root"`
}

type ReqMode string

const (
	MROOT  ReqMode = "root"
	MUSER  ReqMode = "user"
	MGUEST ReqMode = "guest"

	PASSWORD_COST = 10 // for bcrypt
)

type Authentication interface {
	Start(config string) error
	ClearAll() error
	Authenticate(usr, pw string) bool
	NewUser(usr, pw string, root bool) error
	ChangeUserPassword(usr, pw string) error
	ChangeUserStatus(usr string, isactive bool) error
	ListUser(rgx string) ([]User, error)
	ChangeUserDbAccess(usr, db string, grant bool) error
	HasDbAccess(usr, db string) bool
	RemoveUser(usr string) error
	UserInfo(u string) (*User, error)
}

func CheckPassword(pw string) error {
	// disallow whitespace
	r, err := regexp.Compile("\\s")
	if err != nil {
		return err
	}
	if r.MatchString(pw) {
		return errors.New("password cannot contain whitespace")
	}
	// minimum length 8 chars
	if len(pw) < 8 {
		return errors.New("password must be at least 8 chars")
	}

	return nil
}

func CheckUsername(usr string) error {
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

func ValidatePassword(pwh, pw []byte) bool {
	err := bcrypt.CompareHashAndPassword(pwh, pw)
	if err != nil {
		return false
	}
	return true
}

func PasswordEncrypt(pw string) ([]byte, error) {
	pw_bytes := []byte(pw)
	pw_encrypt, err := bcrypt.GenerateFromPassword(pw_bytes, PASSWORD_COST)
	if err != nil {
		return nil, err
	}
	return pw_encrypt, nil
}

var plugins = make(map[string]Authentication)

func Register(name string, plugin Authentication) {
	if plugin == nil {
		panic("Authentication Plugin Registration: plugin is nil")
	}
	if _, exists := plugins[name]; exists {
		panic("Authentication Plugin Registration: plugin '" + name + "' already registered")
	}
	plugins[name] = plugin
}

func NewPlugin(pluginName, config string) (plugin Authentication, err error) {
	plugin, ok := plugins[pluginName]
	if !ok {
		err = fmt.Errorf("Authentication Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config)
	if err != nil {
		plugin = nil
	}
	return
}

// Taken from 'gorilla toolkit secure cookie'
func GenerateRandomKey(strength int) []byte {
	buffer := make([]byte, strength)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return nil
	}
	return buffer
}
