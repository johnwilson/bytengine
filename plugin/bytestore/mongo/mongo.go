package mongo

import (
	"encoding/json"
	bst "github.com/johnwilson/bytengine/bytestore"
	"github.com/johnwilson/bytengine/plugin"
	"github.com/nu7hatch/gouuid"
	"gopkg.in/mgo.v2"
	"io"
	"os"
	"strings"
	"time"
)

const (
	BLOB_DATABASE = "bytengine_bst"
)

type Config struct {
	Addresses    []string      `json:"addresses"`
	Timeout      time.Duration `json:"timeout"`
	AuthDatabase string        `json:"authdb"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
}

type MongoBST struct {
	session  *mgo.Session
	database string
}

func (m *MongoBST) save(db, filename string, file *os.File) (map[string]interface{}, error) {
	gfile, err := m.session.DB(m.database).GridFS(db).Create(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	_, err = io.Copy(gfile, file)
	if err != nil {
		return nil, err
	}
	err = gfile.Close()
	if err != nil {
		return nil, err
	}
	info, err := bst.GetFileInfo(file.Name())
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (m *MongoBST) Start(config string) error {
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
	m.database = BLOB_DATABASE
	return nil
}

func (m *MongoBST) Add(db string, file *os.File) (map[string]interface{}, error) {
	defer file.Close()
	tmp, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	filename := tmp.String()
	info, err := m.save(db, filename, file)
	if err != nil {
		return nil, err
	}
	info["name"] = filename
	return info, nil
}

func (m *MongoBST) Update(db, filename string, file *os.File) (map[string]interface{}, error) {
	defer file.Close()
	info, err := m.save(db, filename, file)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (m *MongoBST) Delete(db, filename string) error {
	err := m.session.DB(m.database).GridFS(db).Remove(filename)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoBST) Read(db, filename string, file io.Writer) error {
	out := file
	gfile, err := m.session.DB(m.database).GridFS(db).Open(filename)
	if err != nil {
		return err
	}
	defer gfile.Close()
	b := make([]byte, 1024)
	for {
		n, err := gfile.Read(b)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		_, err = out.Write(b[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MongoBST) DropDatabase(db string) error {
	exists := false
	list, err := m.session.DB(m.database).CollectionNames()
	if err != nil {
		return err
	}
	for _, item := range list {
		if strings.HasPrefix(item, db+".") {
			exists = true
		}
	}
	// simply return if database doesn't exist
	if !exists {
		return nil
	}

	err = m.session.DB(m.database).C(db + ".chunks").DropCollection()
	if err != nil {
		return err
	}
	err = m.session.DB(m.database).C(db + ".files").DropCollection()
	if err != nil {
		return err
	}
	return nil
}

func NewMongoBST() *MongoBST {
	return &MongoBST{}
}

func init() {
	plugin.Register("mongodb", NewMongoBST())
}
