package mongo

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/johnwilson/bytengine/dsl"
	bfs "github.com/johnwilson/bytengine/filesystem"
	"github.com/johnwilson/bytengine/plugin"
	_ "github.com/johnwilson/bytengine/plugin/bytestore/diskv"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

const (
	BFS_CONFIG = `
    {
        "addresses":["localhost:27017"],
        "authdb":"",
        "username":"",
        "password":"",
        "timeout":60
    }`
	BSTORE_CONFIG = `
    {
        "rootdir":"/tmp/diskv_data",
        "cachesize": 1
    }`
)

func TestDatabaseManagement(t *testing.T) {
	// get bst plugin
	bstore, err := plugin.NewByteStore("diskv", BSTORE_CONFIG)
	assert.Nil(t, err, "bst not created")
	// get bfs plugin
	mfs, err := plugin.NewFileSystem("mongodb", BFS_CONFIG, &bstore)
	assert.Nil(t, err, "bfs not created")

	// Clear all
	rep := mfs.ClearAll()
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// Create databases
	rep = mfs.CreateDatabase("db1")
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.CreateDatabase("db2")
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// List databases
	rep = mfs.ListDatabase("")
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	list := rep.Data.([]string)
	assert.Len(t, list, 2, "all databases not created")

	db1_found := false
	db2_found := false
	for _, db := range list {
		switch db {
		case "db1":
			db1_found = true
		case "db2":
			db2_found = true
		default:
			continue
		}
	}
	assert.True(t, db1_found, "db1 not created")
	assert.True(t, db2_found, "db2 not created")

	// Delete database
	rep = mfs.DropDatabase("db2")
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// check database list
	rep = mfs.ListDatabase("")
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	list = rep.Data.([]string)
	assert.Len(t, list, 1, "database not deleted")

	db1_found = false
	db2_found = false
	for _, db := range list {
		switch db {
		case "db1":
			db1_found = true
		case "db2":
			db2_found = true
		default:
			continue
		}
	}
	assert.True(t, db1_found, "db1 not found after bfs.dropdatabase")
	assert.False(t, db2_found, "db2 not deleted from bfs")
}

func TestContentManagement(t *testing.T) {
	// get bst plugin
	bstore, err := plugin.NewByteStore("diskv", BSTORE_CONFIG)
	assert.Nil(t, err, "bst not created")
	// get bfs plugin
	mfs, err := plugin.NewFileSystem("mongodb", BFS_CONFIG, &bstore)
	assert.Nil(t, err, "bfs not created")

	// set database
	db := "db1"

	// create directories
	rep := mfs.NewDir("/var", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.NewDir("/var/www", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// create file
	rep = mfs.NewFile("/var/www/index.html", db, map[string]interface{}{})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// update file
	data := map[string]interface{}{
		"title": "welcome",
		"body":  "Hello world!",
	}
	rep = mfs.UpdateJson("/var/www/index.html", db, data)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// read file
	rep = mfs.ReadJson("/var/www/index.html", db, []string{"title", "body"})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok := rep.Data.(bson.M)
	assert.True(t, ok, "couldn't cast file content to bson.M")
	assert.Equal(t, val["title"], "welcome", "incorrect file content: title")
	assert.Equal(t, val["body"], "Hello world!", "incorrect file content: body")

	// copy file
	rep = mfs.Copy("/var/www/index.html", "/var/www/index_copy.html", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// directory listing
	rep = mfs.ListDir("/var/www", "", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val2, ok := rep.Data.(map[string][]string)
	assert.True(t, ok, "couldn't cast directory listing to map[string][]string")
	l := val2["files"]
	assert.Len(t, l, 2, "file copy failed")

	// copy directory
	rep = mfs.Copy("/var/www", "/www", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// directory listing
	rep = mfs.ListDir("/www", "", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val2, ok = rep.Data.(map[string][]string)
	assert.True(t, ok, "couldn't cast directory listing to map[string][]string")
	l = val2["files"]
	assert.Len(t, l, 2, "directory copy failed")

	// read copied file contents
	rep = mfs.ReadJson("/www/index_copy.html", db, []string{"title", "body"})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok = rep.Data.(bson.M)
	assert.True(t, ok, "couldn't cast file content to bson.M")
	assert.Equal(t, val["title"], "welcome", "incorrect file content: title")
	assert.Equal(t, val["body"], "Hello world!", "incorrect file content: body")
}

func TestCounters(t *testing.T) {
	// get bst plugin
	bstore, err := plugin.NewByteStore("diskv", BSTORE_CONFIG)
	assert.Nil(t, err, "bst not created")
	// get bfs plugin
	mfs, err := plugin.NewFileSystem("mongodb", BFS_CONFIG, &bstore)
	assert.Nil(t, err, "bfs not created")

	// set database
	db := "db1"

	rep := mfs.SetCounter("users", "incr", 1, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok := rep.Data.(int64)
	assert.True(t, ok, "couldn't cast search result into int")
	assert.Equal(t, val, 1, "counter create failed")

	rep = mfs.SetCounter("users", "decr", 1, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok = rep.Data.(int64)
	assert.True(t, ok, "couldn't cast search result into int")
	assert.Equal(t, val, 0, "counter create failed")

	rep = mfs.SetCounter("users", "reset", 5, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok = rep.Data.(int64)
	assert.True(t, ok, "couldn't cast search result into int")
	assert.Equal(t, val, 5, "counter create failed")

	rep = mfs.SetCounter("user1.likes", "incr", 1, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.SetCounter("car.users", "incr", 1, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	rep = mfs.ListCounter("", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val2, ok := rep.Data.([]CounterItem)
	assert.True(t, ok, "couldn't cast search result into []interface")
	assert.Len(t, val2, 3, "counter list failed")

	rep = mfs.ListCounter("^user", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val2, ok = rep.Data.([]CounterItem)
	assert.True(t, ok, "couldn't cast search result into []interface")
	assert.Len(t, val2, 2, "counter list failed")
}

func TestSearch(t *testing.T) {
	// get bst plugin
	bstore, err := plugin.NewByteStore("diskv", BSTORE_CONFIG)
	assert.Nil(t, err, "bst not created")
	// get bfs plugin
	mfs, err := plugin.NewFileSystem("mongodb", BFS_CONFIG, &bstore)
	assert.Nil(t, err, "bfs not created")

	// set database
	db := "db1"

	// create dir and add files
	rep := mfs.NewDir("/users", db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.NewFile("/users/u1", db, map[string]interface{}{
		"name":    "john",
		"age":     34,
		"country": "ghana",
	})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.NewFile("/users/u2", db, map[string]interface{}{
		"name":    "jason",
		"age":     18,
		"country": "ghana",
	})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.NewFile("/users/u3", db, map[string]interface{}{
		"name": "juliette",
		"age":  18,
	})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.NewFile("/users/u4", db, map[string]interface{}{
		"name":    "michelle",
		"age":     21,
		"country": "uk",
	})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	rep = mfs.NewFile("/users/u5", db, map[string]interface{}{
		"name":    "dennis",
		"age":     22,
		"country": "france",
	})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// parse script and run
	script := `@test.select "name" "age" in /users where "country" in ["ghana"]`
	cmd, err := dsl.NewParser().Parse(script)
	assert.Nil(t, err, "couldn't parse script")
	rep = mfs.BQLSearch(db, cmd[0].Args)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok := rep.Data.([]interface{})
	assert.True(t, ok, "couldn't cast search result into []interface")
	assert.Len(t, val, 2, "search failed")

	script = `
    @test.select "name" "age" in /users
    where regex("name","i") == "^j\\w*n$"`
	cmd, err = dsl.NewParser().Parse(script)
	assert.Nil(t, err, "couldn't parse script")
	rep = mfs.BQLSearch(db, cmd[0].Args)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok = rep.Data.([]interface{})
	assert.True(t, ok, "couldn't cast search result into []interface")
	assert.Len(t, val, 2, "search failed")

	script = `
    @test.select "name" "age" in /users
    where exists("country") == true`
	cmd, err = dsl.NewParser().Parse(script)
	assert.Nil(t, err, "couldn't parse script")
	rep = mfs.BQLSearch(db, cmd[0].Args)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok = rep.Data.([]interface{})
	assert.True(t, ok, "couldn't cast search result into []interface")
	assert.Len(t, val, 4, "search failed")
}

func TestSetUnset(t *testing.T) {
	// get bst plugin
	bstore, err := plugin.NewByteStore("diskv", BSTORE_CONFIG)
	assert.Nil(t, err, "bst not created")
	// get bfs plugin
	mfs, err := plugin.NewFileSystem("mongodb", BFS_CONFIG, &bstore)
	assert.Nil(t, err, "bfs not created")

	// set database
	db := "db1"

	// parse script and run
	script := `
    @test.set "country"={"name":"ghana","major_cities":["kumasi","accra"]}
    in /users
    where "country" == "ghana"
    `
	cmd, err := dsl.NewParser().Parse(script)
	assert.Nil(t, err, "couldn't parse script")
	rep := mfs.BQLSet(db, cmd[0].Args)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok := rep.Data.(int)
	assert.True(t, ok, "couldn't cast search result into int")
	assert.Equal(t, val, 2, "set data failed")

	rep = mfs.ReadJson("/users/u1", db, []string{})
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	data, ok := rep.Data.(bson.M)
	assert.True(t, ok, "couldn't cast file content to bson.M")
	country, ok := data["country"].(bson.M)
	assert.True(t, ok, "couldn't cast file content to bson.M")
	assert.Equal(t, country["name"], "ghana", "incorrect file content update")

	script = `
    @test.unset "country"
    in /users
    where exists("country") == true
    `
	cmd, err = dsl.NewParser().Parse(script)
	assert.Nil(t, err, "couldn't parse script")
	rep = mfs.BQLUnset(db, cmd[0].Args)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val, ok = rep.Data.(int)
	assert.True(t, ok, "couldn't cast search result into int")
	assert.Equal(t, val, 4, "unset data failed")

	script = `@test.select "name" in /users where exists("country") == false`
	cmd, err = dsl.NewParser().Parse(script)
	assert.Nil(t, err, "couldn't parse script")
	rep = mfs.BQLSearch(db, cmd[0].Args)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	val2, ok := rep.Data.([]interface{})
	assert.True(t, ok, "couldn't cast search result into []interface")
	assert.Len(t, val2, 5, "search failed")
}

func TestAttachmentManagement(t *testing.T) {
	// get bst plugin
	bstore, err := plugin.NewByteStore("diskv", BSTORE_CONFIG)
	assert.Nil(t, err, "bst not created")
	// get bfs plugin
	mfs, err := plugin.NewFileSystem("mongodb", BFS_CONFIG, &bstore)
	assert.Nil(t, err, "bfs not created")

	// set database
	db := "db1"

	// create test file
	txt := "Hello from bst!"
	fpath := "/tmp/bfs_attach.txt"
	err = ioutil.WriteFile(fpath, []byte(txt), 0777)
	assert.Nil(t, err, "test file not created")

	data := map[string]interface{}{
		"title": "bfs test file",
		"type":  ".txt",
	}
	bfs_path := "/file_with_attachment"
	rep := mfs.NewFile(bfs_path, db, data)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// add to bfs
	rep = mfs.WriteBytes(bfs_path, fpath, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)

	// read from store
	fpath2 := "/tmp/bfs_attach_down.txt"
	f2, err := os.Create(fpath2)
	assert.Nil(t, err, "download test file not created")

	rep = mfs.ReadBytes(bfs_path, db)
	assert.Equal(t, bfs.OK, rep.Status, rep.StatusMessage)
	bstore_id, ok := rep.Data.(string)
	assert.True(t, ok, "couldn't cast bst id into string")

	bstore.Read(db, bstore_id, f2)
	f2.Close()

	// check downloaded file data
	fdata, err := ioutil.ReadFile(fpath2)
	assert.Nil(t, err, "download test file couldn't be opened")
	assert.Equal(t, txt, string(fdata), "attachment file content has changed")
}
