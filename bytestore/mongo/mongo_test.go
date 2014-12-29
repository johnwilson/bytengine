package mongo

import (
	"github.com/johnwilson/bytengine"
	"io/ioutil"
	"os"
	"testing"
)

func TestMongoBST(t *testing.T) {
	// create test file
	txt := "Hello from bst!"
	fpath := "/tmp/bytengine_bst_test.txt"
	err := ioutil.WriteFile(fpath, []byte(txt), 0777)
	if err != nil {
		t.Fatal(err)
	}

	// create bst client
	b, err := bytengine.NewByteStore(
		"mongodb",
		`{
            "addresses":["localhost:27017"],
            "authdb":"",
            "username":"",
            "password":"",
            "storedb":"bytestore",
            "timeout":60
        }`,
	)
	if err != nil {
		t.Fatal(err)
	}

	db := "bst_test"
	// open test file
	f, err := os.Open(fpath)
	if err != nil {
		t.Fatal(err)
	}
	// add to store
	info, err := b.Add(db, f)
	if err != nil {
		t.Fatal(err)
	}

	// read from store
	fpath2 := "/tmp/bytengine_bst_down.txt"
	f2, err := os.Create(fpath2)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Read(db, info["name"].(string), f2)
	if err != nil {
		t.Fatal(err)
	}
	// check downloaded file data
	data, err := ioutil.ReadFile(fpath2)
	if err != nil {
		t.Fatal(err)
	}
	if txt != string(data) {
		t.Fatal("Content in files is different")
	}

	// delete from store
	err = b.Delete(db, info["name"].(string))
	if err != nil {
		t.Fatal(err)
	}

	// drop database
	err = b.DropDatabase(db)
	if err != nil {
		t.Fatal(err)
	}
}
