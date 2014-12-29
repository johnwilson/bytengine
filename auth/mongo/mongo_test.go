package mongo

import (
	"github.com/johnwilson/bytengine"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	CONFIG = `
    {
        "addresses":["localhost:27017"],
        "authdb":"",
        "username":"",
        "password":"",
        "userdb":"authdb",
        "timeout":60
    }`
)

func TestUserManagement(t *testing.T) {
	mgauth, err := bytengine.NewAuthentication("mongodb", CONFIG)
	assert.Nil(t, err, "auth not created")

	// initialize db
	err = mgauth.ClearAll()
	assert.Nil(t, err, "database initialization failed")

	// create user
	err = mgauth.NewUser("john", "password", false)
	assert.Nil(t, err, "user not created")

	// authenticate user
	ok := mgauth.Authenticate("john", "wrongpassword")
	assert.False(t, ok, "authentication should have failed")
	ok = mgauth.Authenticate("john", "password")
	assert.True(t, ok, "authentication failed")

	// password update
	err = mgauth.ChangeUserPassword("john", "password2")
	assert.Nil(t, err, "password update failed")
	ok = mgauth.Authenticate("john", "password")
	assert.False(t, ok, "authentication should have failed")
	ok = mgauth.Authenticate("john", "password2")
	assert.True(t, ok, "authentication failed")

	// database access
	err = mgauth.ChangeUserDbAccess("john", "db1", true)
	assert.Nil(t, err, "database access failed")
	ok = mgauth.HasDbAccess("john", "db")
	assert.False(t, ok, "database access failed")
	ok = mgauth.HasDbAccess("john", "db1")
	assert.True(t, ok, "database access failed")

	// system access
	err = mgauth.ChangeUserStatus("john", false)
	assert.Nil(t, err, "user status update failed")
	ok = mgauth.Authenticate("john", "password2")
	assert.False(t, ok, "authentication should have failed")

	// delete user / list users
	l, err := mgauth.ListUser("")
	assert.Len(t, l, 1, "user list error")
	err = mgauth.RemoveUser("john")
	assert.Nil(t, err, "delete user failed")
	l, err = mgauth.ListUser("")
	assert.Len(t, l, 0, "user list error")
}
