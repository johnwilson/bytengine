package mongo

import (
	//"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/johnwilson/bytengine/bfs"
	"github.com/johnwilson/bytengine/bst"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

// BFS Node Header
type NodeHeader struct {
	Name     string `bson:"name"`
	Type     string `bson:"type"`
	IsPublic bool   `bson:"ispublic"`
	Created  string `bson:"created"`
	Parent   string `bson:"parent"`
}

// BFS Bytes Header
type BytesHeader struct {
	Filepointer string `bson:"filepointer"`
	Mime        string `bson:"mime"`
	Size        int64  `bson:"size"`
}

// BFS Directory
type Directory struct {
	Header NodeHeader `bson:"__header__"`
	Id     string     `bson:"_id"`
}

// BFS File
type File struct {
	Header  NodeHeader             `bson:"__header__"`
	AHeader BytesHeader            `bson:"__bytes__"`
	Id      string                 `bson:"_id"`
	Content map[string]interface{} `bson:"content"`
}

type Config struct {
	Addresses    []string      `json:"addresses"`
	Timeout      time.Duration `json:"timeout"`
	AuthDatabase string        `json:"authdb"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
}

func NewMongodbBFS() *MongodbBFS {
	return &MongodbBFS{}
}

const (
	DB_PREFIX          = "bfs_"
	BFS_COLLECTION     = "bfs"
	COUNTER_COLLECTION = "bfs.counters"
)

type MongodbBFS struct {
	session *mgo.Session
	bstore  bst.ByteStore
}

type SimpleResultItem struct {
	Header  NodeHeader  `bson:"__header__"`
	AHeader BytesHeader `bson:"__bytes__"`
	Id      string      `bson:"_id"`
}

type CounterItem struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

/*
============================================================================
    Private Methods
============================================================================
*/

func makeRootDir() (*Directory, error) {
	id, err := bfs.NewNodeID()
	if err != nil {
		return nil, err
	}
	dt := bfs.FormatDatetime(time.Now())
	h := NodeHeader{"/", "Directory", true, dt, ""}
	r := &Directory{h, id}
	return r, nil
}

func (m *MongodbBFS) existsDocument(p string, c *mgo.Collection) (SimpleResultItem, bool) {
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		// log error
		return ri, false
	}
	return ri, true
}

func (m *MongodbBFS) copyDirectoryDocument(d *Directory, newprefix, oldprefix, newname string, c *mgo.Collection) error {
	// update parent path prefix with new prefix
	_parent_path := d.Header.Parent
	_parent_path = strings.Replace(_parent_path, oldprefix, newprefix, 1)

	// update header info
	id, err := bfs.NewNodeID()
	if err != nil {
		return err
	}

	d.Header.Parent = _parent_path
	if newname != "" {
		err = bfs.ValidateDirName(newname)
		if err != nil {
			return err
		}
		d.Header.Name = newname
	}
	d.Header.Created = bfs.FormatDatetime(time.Now())
	d.Id = id
	// save to mongodb
	err = c.Insert(&d)
	if err != nil {
		return err
	}

	return nil
}

func (m *MongodbBFS) copyFileDocument(f *File, newprefix, oldprefix, newname string, c *mgo.Collection) error {
	// update parent path prefix with new prefix
	_parent_path := f.Header.Parent
	_parent_path = strings.Replace(_parent_path, oldprefix, newprefix, 1)

	// update header info
	// both the original and copy will point to the same attachment id
	// in the bst
	id, err := bfs.NewNodeID()
	if err != nil {
		return err
	}
	f.Header.Parent = _parent_path
	f.Header.Created = bfs.FormatDatetime(time.Now())
	if newname != "" {
		err = bfs.ValidateFileName(newname)
		if err != nil {
			return err
		}
		f.Header.Name = newname
	}
	f.Id = id

	// save to mongodb
	err = c.Insert(&f)
	if err != nil {
		return err
	}

	return nil
}

func (m *MongodbBFS) findPathQuery(p string) bson.M {
	// build query
	var q bson.M
	if p == "/" {
		q = bson.M{"__header__.parent": "", "__header__.name": "/"}
	} else {
		q = bson.M{"__header__.parent": path.Dir(p), "__header__.name": path.Base(p)}
	}
	return q
}

func (m *MongodbBFS) findChildrenQuery(p, rgx string) bson.M {
	qre := bson.RegEx{Pattern: rgx, Options: "i"} // case insensitive regex
	q := bson.M{
		"__header__.parent": p,
		"__header__.name":   bson.M{"$regex": qre},
	}
	return q
}

func (m *MongodbBFS) findAllChildrenQuery(p string) bson.M {
	// pattern
	var r string
	if p == "/" {
		r = "^/"
	} else {
		r = fmt.Sprintf("^%s($|/)", p)
	}
	q := bson.M{"__header__.parent": bson.RegEx{r, "i"}}
	return q
}

func (m *MongodbBFS) getBFSCollection(db string) *mgo.Collection {
	actual_db := DB_PREFIX + db
	return m.session.DB(actual_db).C(BFS_COLLECTION)
}

func (m *MongodbBFS) getCounterCollection(db string) *mgo.Collection {
	actual_db := DB_PREFIX + db
	return m.session.DB(actual_db).C(COUNTER_COLLECTION)
}

/*
============================================================================
    BFS Interface Methods
============================================================================
*/

func (m *MongodbBFS) Start(config string, b *bst.ByteStore) error {
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
	m.bstore = *b
	return nil
}

func (m *MongodbBFS) ClearAll() bfs.BFSResponse {
	dbs, err := m.session.DatabaseNames()
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	found := make([]string, 0)
	for _, db := range dbs {
		if strings.HasPrefix(db, DB_PREFIX) {
			err = m.session.DB(db).DropDatabase()
			if err != nil {
				return bfs.ErrorResponse(err)
			}
			// drop database from bst
			err = m.bstore.DropDatabase(db)
			if err != nil {
				return bfs.ErrorResponse(err)
			}
			found = append(found, db)
		}
	}

	return bfs.OKResponse(found)
}

func (m *MongodbBFS) ListDatabase(filter string) bfs.BFSResponse {
	r, err := regexp.Compile(filter)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	dbs, err := m.session.DatabaseNames()
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	found := make([]string, 0)
	for _, db := range dbs {
		if strings.HasPrefix(db, DB_PREFIX) {
			db = strings.Replace(db, DB_PREFIX, "", 1) // remove prefix
			if r.MatchString(db) {
				found = append(found, db)
			}
		}
	}

	return bfs.OKResponse(found)
}

func (m *MongodbBFS) CreateDatabase(db string) bfs.BFSResponse {
	err := bfs.ValidateDbName(db)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	// create mongodb database collection root node
	rn, err := makeRootDir()
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	// create mongodb database and collection and insert record
	col := m.getBFSCollection(db)

	err = col.Insert(&rn)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) DropDatabase(db string) bfs.BFSResponse {
	actual_db := DB_PREFIX + db

	// check if db to be deleted exists
	dbs, err := m.session.DatabaseNames()
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	_db_exists := false
	for _, item := range dbs {
		if item == actual_db {
			_db_exists = true
			break
		}
	}
	if !_db_exists {
		msg := fmt.Sprintf("database '%s' doesn't exist", db)
		return bfs.ErrorResponse(errors.New(msg))
	}

	// drop db from mongodb
	err = m.session.DB(actual_db).DropDatabase()
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	// drop database from bst
	err = m.bstore.DropDatabase(db)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) NewDir(p, db string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)
	if p == "/" {
		return bfs.ErrorResponse(errors.New("root directory already exists"))
	}
	_name := path.Base(p)
	_parent := path.Dir(p)
	err := bfs.ValidateDirName(_name)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	// check if parent directory exists
	q := m.findPathQuery(_parent)

	// get collection
	c := m.getBFSCollection(db)
	var _parentdir Directory
	// find record
	err = c.Find(q).One(&_parentdir)
	if err != nil {
		fmt.Println("here")
		return bfs.ErrorResponse(err)
	}
	if _parentdir.Header.Type != "Directory" {
		msg := fmt.Sprintf("directory '%s' couldn't be created: destination isn't a directory.", p)
		return bfs.ErrorResponse(errors.New(msg))
	}
	// check if name already taken
	q = m.findPathQuery(p)
	_count, err := c.Find(q).Count()
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	if _count > 0 {
		msg := fmt.Sprintf("directory '%s' already exists", p)
		return bfs.ErrorResponse(errors.New(msg))
	}

	// create directory
	id, err := bfs.NewNodeID()
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	dt := bfs.FormatDatetime(time.Now())
	h := NodeHeader{_name, "Directory", false, dt, _parent}
	_dir := Directory{h, id}
	// insert node into mongodb
	err = c.Insert(&_dir)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) NewFile(p, db string, j map[string]interface{}) bfs.BFSResponse {
	// check path
	p = path.Clean(p)
	_name := path.Base(p)
	_parent := path.Dir(p)
	err := bfs.ValidateFileName(_name)
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	// check if parent directory exists
	q := m.findPathQuery(_parent)

	// get collection
	c := m.getBFSCollection(db)
	var _parentdir Directory
	// find record
	err = c.Find(q).One(&_parentdir)
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created: destination directory not found\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}
	if _parentdir.Header.Type != "Directory" {
		msg := fmt.Sprintf("file '%s' couldn't be created: destination isn't a directory.", p)
		return bfs.ErrorResponse(errors.New(msg))
	}
	// check if name already taken
	q = m.findPathQuery(p)
	_count, err := c.Find(q).Count()
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}
	if _count > 0 {
		msg := fmt.Sprintf("file '%s' already exists", p)
		return bfs.ErrorResponse(errors.New(msg))
	}

	// create file
	id, err := bfs.NewNodeID()
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}
	dt := bfs.FormatDatetime(time.Now())
	h := NodeHeader{_name, "File", false, dt, _parent}
	a := BytesHeader{"", "", 0}
	_file := File{h, a, id, j}
	// insert node into mongodb
	err = c.Insert(&_file)
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) ListDir(p, filter, db string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// find path
	q := m.findPathQuery(p)
	n, err := c.Find(q).Count()
	if err != nil {
		msg := fmt.Sprintf("error while trying to find the path: %s.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}
	if n != 1 {
		msg := fmt.Sprintf("path '%s' doesn't exist.", p)
		return bfs.ErrorResponse(errors.New(msg))
	}

	// find children
	q = m.findChildrenQuery(p, filter)
	i := c.Find(q).Sort("__header__.name").Iter()
	var ri SimpleResultItem
	dirs := make([]string, 0)
	files := make([]string, 0)
	bfiles := make([]string, 0) // files with attachments

	for i.Next(&ri) {
		if ri.Header.Type == "Directory" {
			dirs = append(dirs, ri.Header.Name)
		} else {
			if ri.AHeader.Filepointer == "" {
				files = append(files, ri.Header.Name)
			} else {
				bfiles = append(bfiles, ri.Header.Name)
			}
		}
	}
	err = i.Err()
	if err != nil {
		msg := fmt.Sprintf("error while trying directory listing for: %s\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}
	res := map[string][]string{
		"dirs":   dirs,
		"files":  files,
		"bfiles": bfiles,
	}

	return bfs.OKResponse(res)
}

func (m *MongodbBFS) ReadJson(p, db string, fields []string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file if it exists
	q := m.findPathQuery(p)
	q["__header__.type"] = "File"

	var r bson.M
	if len(fields) == 0 {
		err := c.Find(q).One(&r)
		if err != nil {
			msg := fmt.Sprintf("file '%s' content couldn't be retrieved.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	} else {
		_flds := bson.M{"__header__": 1}
		for _, item := range fields {
			_flds["content."+item] = 1
		}
		err := c.Find(q).Select(_flds).One(&r)
		if err != nil {
			msg := fmt.Sprintf("file '%s' content couldn't be retrieved.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(r["content"])
}

func (m *MongodbBFS) Delete(p, db string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)
	if p == "/" {
		return bfs.ErrorResponse(errors.New("root directory cannot be deleted."))
	}

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}
	if ri.Header.Type == "Directory" {
		// find all children
		q = m.findAllChildrenQuery(p)
		i := c.Find(q).Iter()
		var ri2 SimpleResultItem
		_attchs := []string{} // list of all attachments paths
		for i.Next(&ri2) {
			if ri2.Header.Type == "File" && ri2.AHeader.Filepointer != "" {
				_attchs = append(_attchs, ri2.AHeader.Filepointer)
			}
		}
		err = i.Err()
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// delete all children
		_, err := c.RemoveAll(q)
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// delete attachments from bst
		for _, item := range _attchs {
			err = m.bstore.Delete(db, item)
			if err != nil {
				msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}
		// delete directory
		err = c.RemoveId(ri.Id)
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}

	} else {
		if ri.AHeader.Filepointer != "" {
			// delete attachment from bst
			err = m.bstore.Delete(db, ri.AHeader.Filepointer)
			if err != nil {
				msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}
		// delete file
		err = c.RemoveId(ri.Id)
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) Rename(p, newname, db string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)
	if p == "/" {
		return bfs.ErrorResponse(errors.New("root directory cannot be renamed."))
	}

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("renaming '%s'failed.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	if ri.Header.Type == "Directory" {
		// check if name is valid
		if bfs.ValidateDirName(newname) != nil {
			msg := fmt.Sprintf("invalid directory name: %s", newname)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// check if name isn't already in use
		np := path.Join(path.Dir(p), newname)
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, e)
			return bfs.ErrorResponse(errors.New(msg))
		}
		if _count > 0 {
			msg := fmt.Sprintf("directory '%s' already exists", np)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// get affected parent directories
		q = m.findAllChildrenQuery(p)
		var _dirs []string
		err = c.Find(q).Distinct("__header__.parent", &_dirs)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
		for _, item := range _dirs {
			newparent := strings.Replace(item, p, np, 1)
			q = bson.M{"__header__.parent": item}
			uq := bson.M{"$set": bson.M{"__header__.parent": newparent}}
			_, e := c.UpdateAll(q, uq)
			if e != nil {
				msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, e)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}
		// rename directory by updating field
		q = bson.M{"$set": bson.M{"__header__.name": newname}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}

	} else {
		// check if name is valid
		if bfs.ValidateFileName(newname) != nil {
			msg := fmt.Sprintf("invalid file name: %s", newname)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// check if name isn't already in use
		np := path.Join(path.Dir(p), newname)
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("file '%s' couldn't be renamed.\n%s", p, e)
			return bfs.ErrorResponse(errors.New(msg))
		}
		if _count > 0 {
			msg := fmt.Sprintf("file '%s' already exists", np)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// rename file by updating field
		q = bson.M{"$set": bson.M{"__header__.name": newname}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("file '%s' couldn't be renamed.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) Move(from, to, db string) bfs.BFSResponse {
	// check path
	from = path.Clean(from) // from
	to = path.Clean(to)     // to
	if from == "/" {
		return bfs.ErrorResponse(errors.New("root directory cannot be moved."))
	}
	// check illegal move operation i.e. moving from parent to sub directory
	if strings.HasPrefix(to, from) {
		return bfs.ErrorResponse(errors.New("illegal move operation."))
	}

	// get collection
	c := m.getBFSCollection(db)

	// check if destination dir exists
	_doc_dest, _exists_dest := m.existsDocument(to, c)
	if !_exists_dest {
		return bfs.ErrorResponse(errors.New("Destination directory doesn't exist"))
	}
	if _doc_dest.Header.Type != "Directory" {
		return bfs.ErrorResponse(errors.New("Destination must be a directory"))
	}

	// get file or directory if it exists
	q := m.findPathQuery(from)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("moving '%s'failed.\n%s", from, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	if ri.Header.Type == "Directory" {
		// check if name isn't already in use
		np := path.Join(to, path.Base(from))
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", from, e)
			return bfs.ErrorResponse(errors.New(msg))
		}
		if _count > 0 {
			msg := fmt.Sprintf("directory '%s' already exists", np)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// get affected parent directories
		q = m.findAllChildrenQuery(from)
		var _dirs []string
		err = c.Find(q).Distinct("__header__.parent", &_dirs)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", from, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
		for _, item := range _dirs {
			newparent := strings.Replace(item, from, np, 1)
			q = bson.M{"__header__.parent": item}
			uq := bson.M{"$set": bson.M{"__header__.parent": newparent}}
			_, e := c.UpdateAll(q, uq)
			if e != nil {
				msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", from, e)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}
		// move directory by updating parent field
		q = bson.M{"$set": bson.M{"__header__.parent": to}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", from, err)
			return bfs.ErrorResponse(errors.New(msg))
		}

	} else {
		// check if name isn't already in use
		np := path.Join(to, path.Base(from))
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("file '%s' couldn't be moved.\n%s", from, e)
			return bfs.ErrorResponse(errors.New(msg))
		}
		if _count > 0 {
			msg := fmt.Sprintf("file '%s' already exists", np)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// rename file by updating field
		q = bson.M{"$set": bson.M{"__header__.parent": to}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("file '%s' couldn't be moved.\n%s", from, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) Copy(from, to, db string) bfs.BFSResponse {
	// setup paths
	_from_doc_path := path.Clean(from)
	_from_doc_parent_path := path.Dir(_from_doc_path)
	_to_doc_path := path.Clean(to)
	_to_doc_parent_path := path.Dir(_to_doc_path)
	_to_doc_name := path.Base(_to_doc_path)

	if _from_doc_path == "/" {
		return bfs.ErrorResponse(errors.New("root directory cannot be copied."))
	}
	// check illegal copy operation i.e. copy from parent to sub directory
	if strings.HasPrefix(_to_doc_parent_path, _from_doc_path) {
		return bfs.ErrorResponse(errors.New("illegal copy operation."))
	}

	// get collection
	c := m.getBFSCollection(db)

	// check if destination dir exists
	_doc_dest, _exists_dest := m.existsDocument(_to_doc_parent_path, c)
	if !_exists_dest {
		return bfs.ErrorResponse(errors.New("Destination directory doesn't exist"))
	}
	if _doc_dest.Header.Type != "Directory" {
		return bfs.ErrorResponse(errors.New("Destination must be a directory"))
	}

	// check if item to copy exists
	_doc, _exists := m.existsDocument(_from_doc_path, c)
	if !_exists {
		return bfs.ErrorResponse(errors.New(fmt.Sprintf("'%s' doesn't exist", _from_doc_path)))
	}

	// check if name isn't already in use
	_, _exists = m.existsDocument(_to_doc_path, c)
	if _exists {
		return bfs.ErrorResponse(errors.New(fmt.Sprintf("'%s' already exists.", _to_doc_path)))
	}

	if _doc.Header.Type == "Directory" {
		// get full document
		var _main_dir Directory
		err := c.FindId(_doc.Id).One(&_main_dir)
		if err != nil {
			msg := fmt.Sprintf("copying '%s'failed.\n%s", _from_doc_path, err)
			return bfs.ErrorResponse(errors.New(msg))
		}

		// copy directory
		err = m.copyDirectoryDocument(&_main_dir, _to_doc_parent_path, _from_doc_parent_path, _to_doc_name, c)
		if err != nil {
			txt := "sub-directory '%s' in directory '%s' couldn't be copied.\n%s"
			msg := fmt.Sprintf(txt, _main_dir.Header.Name, _main_dir.Header.Parent, err)
			return bfs.ErrorResponse(errors.New(msg))
		}

		// get affected dirs
		q := m.findAllChildrenQuery(_from_doc_path)
		q["__header__.type"] = "Directory"
		var _tmpdir Directory
		i := c.Find(q).Iter()
		for i.Next(&_tmpdir) {
			err = m.copyDirectoryDocument(&_tmpdir, _to_doc_path, _from_doc_path, "", c)
			if err != nil {
				txt := "sub-directory '%s' in directory '%s' couldn't be copied.\n%s"
				msg := fmt.Sprintf(txt, _tmpdir.Header.Name, _tmpdir.Header.Parent, err)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}

		// get affected files
		q = m.findAllChildrenQuery(_from_doc_path)
		q["__header__.type"] = "File"
		var _tmpfile File
		i = c.Find(q).Iter()
		for i.Next(&_tmpfile) {
			err = m.copyFileDocument(&_tmpfile, _to_doc_path, _from_doc_path, "", c)
			if err != nil {
				txt := "file '%s' in directory '%s' couldn't be copied.\n%s"
				msg := fmt.Sprintf(txt, _tmpfile.Header.Name, _tmpfile.Header.Parent, err)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}

	} else {
		// get full document
		var _filedoc File
		err := c.FindId(_doc.Id).One(&_filedoc)
		if err != nil {
			msg := fmt.Sprintf("copying '%s'failed.\n%s", _from_doc_path, err)
			return bfs.ErrorResponse(errors.New(msg))
		}

		// copy file
		err = m.copyFileDocument(&_filedoc, _to_doc_parent_path, _from_doc_parent_path, _to_doc_name, c)
		if err != nil {
			txt := "file '%s' in directory '%s' couldn't be copied.\n%s"
			msg := fmt.Sprintf(txt, _filedoc.Header.Name, _filedoc.Header.Parent, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) Info(p, db string) bfs.BFSResponse {
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	var _info map[string]interface{}
	// info elements
	_name := ri.Header.Name
	_created := ri.Header.Created
	_parent := ri.Header.Parent
	_public := ri.Header.IsPublic

	_info = bson.M{
		"name":    _name,
		"created": _created,
		"public":  _public,
		"parent":  _parent,
	}

	if ri.Header.Type == "Directory" {
		_type := "directory"
		// count child nodes
		q = m.findChildrenQuery(p, ".")
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, e)
			return bfs.ErrorResponse(errors.New(msg))
		}
		_info["type"] = _type
		_info["content_count"] = _count

	} else {
		_type := "file"
		_info["type"] = _type
		if ri.AHeader.Filepointer != "" {
			_info["attachment"] = true
		}
	}

	return bfs.OKResponse(_info)
}

func (m *MongodbBFS) FileAccess(p, db string, protect bool) bfs.BFSResponse {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	if ri.Header.Type == "Directory" {
		// update directory access by updating field
		q = bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' access couldn't be updated.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
		// automatically cascade to sub nodes
		q = m.findAllChildrenQuery(p)
		uq := bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		_, e := c.UpdateAll(q, uq)
		if e != nil {
			msg := fmt.Sprintf("directory '%s' access couldn't be updated.\n%s", p, e)
			return bfs.ErrorResponse(errors.New(msg))
		}

	} else {
		// update file access by updating field
		q = bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("file '%s' access couldn't be updated.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) SetCounter(counter, action string, value int64, db string) bfs.BFSResponse {
	// update value 'v'
	nv := math.Abs(float64(value))
	value = int64(nv)

	// get collection
	c := m.getCounterCollection(db)

	// check if counter exists
	q := bson.M{"name": counter}
	num, err := c.Find(q).Count()
	if err != nil {
		return bfs.ErrorResponse(err)
	}
	// if not exists create new counter
	if num < 1 {
		err = bfs.ValidateCounterName(counter)
		if err != nil {
			return bfs.ErrorResponse(err)
		}

		doc := bson.M{"name": counter, "value": value}
		err = c.Insert(doc)
		if err != nil {
			return bfs.ErrorResponse(err)
		}

		return bfs.OKResponse(value)
	}

	var cq mgo.Change
	switch action {
	case "incr":
		cq = mgo.Change{
			Update:    bson.M{"$inc": bson.M{"value": value}},
			ReturnNew: true,
		}
		break
	case "decr":
		cq = mgo.Change{
			Update:    bson.M{"$inc": bson.M{"value": -value}},
			ReturnNew: true,
		}
		break
	case "reset":
		cq = mgo.Change{
			Update:    bson.M{"$set": bson.M{"value": value}},
			ReturnNew: true,
		}
		break
	default: // shouldn't reach here
		return bfs.ErrorResponse(errors.New("Invalid counter action."))
	}

	var r interface{}
	_, err = c.Find(q).Apply(cq, &r)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(r.(bson.M)["value"])
}

func (m *MongodbBFS) ListCounter(filter, db string) bfs.BFSResponse {
	// get collection
	c := m.getCounterCollection(db)

	list := []CounterItem{}
	qre := bson.RegEx{Pattern: filter, Options: "i"} // case insensitive regex
	q := bson.M{"name": bson.M{"$regex": qre}}
	iter := c.Find(q).Iter()
	var ci CounterItem
	for iter.Next(&ci) {
		list = append(list, ci)
	}
	err := iter.Close()
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(list)
}

func (m *MongodbBFS) WriteBytes(p, ap, db string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)
	_, err := os.Stat(ap)
	if err != nil {
		return bfs.ErrorResponse(errors.New("File not found."))
	}

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	if ri.Header.Type == "Directory" {
		return bfs.ErrorResponse(errors.New("command only valid for files."))

	} else {
		// if bytes already writen then update else create new
		isnew := true
		if ri.AHeader.Filepointer != "" {
			isnew = false
		}
		// open attachment and add to bst
		file, err := os.Open(ap)
		if err != nil {
			return bfs.ErrorResponse(err)
		}

		var q bson.M // query

		if isnew {
			info, err := m.bstore.Add(db, file)
			if err != nil {
				return bfs.ErrorResponse(err)
			}
			// update file access by updating field
			q = bson.M{
				"$set": bson.M{
					"__bytes__.filepointer": info["name"].(string),
					"__bytes__.size":        info["size"].(int64),
					"__bytes__.mime":        info["mime"].(string),
				}}
		} else {
			info, err := m.bstore.Update(db, ri.AHeader.Filepointer, file)
			if err != nil {
				return bfs.ErrorResponse(err)
			}
			// update file access by updating field
			q = bson.M{
				"$set": bson.M{
					"__bytes__.size": info["size"].(int64),
					"__bytes__.mime": info["mime"].(string),
				}}
		}

		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("attachment for file '%s' couldn't be added.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) ReadBytes(fp, db string) bfs.BFSResponse {
	// check path
	fp = path.Clean(fp)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(fp)
	var ri File
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", fp, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	if ri.Header.Type == "Directory" {
		return bfs.ErrorResponse(errors.New("command only valid for files."))

	}
	id := ri.AHeader.Filepointer
	/*err = m.bstore.Read(db, id, file)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", fp, err)
		return bfs.ErrorResponse(errors.New(msg))
	}*/
	return bfs.OKResponse(id)
}

func (m *MongodbBFS) DeleteBytes(p, db string) bfs.BFSResponse {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	if ri.Header.Type == "Directory" {
		return bfs.ErrorResponse(errors.New("command only valid for files."))

	} else {
		// delete attachment
		if ri.AHeader.Filepointer != "" {
			// delete attachment
			err = m.bstore.Delete(db, ri.AHeader.Filepointer)
			if err != nil && os.IsExist(err) {
				msg := fmt.Sprintf("attachment for file '%s' couldn't be deleted.\n%s", p, err)
				return bfs.ErrorResponse(errors.New(msg))
			}
		}
		// update file access by updating field
		q = bson.M{"$set": bson.M{"__bytes__.filepointer": ""}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("attachment for file '%s' couldn't be deleted.\n%s", p, err)
			return bfs.ErrorResponse(errors.New(msg))
		}
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) UpdateJson(p, db string, j map[string]interface{}) bfs.BFSResponse {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file if it exists
	q := m.findPathQuery(p)
	q["__header__.type"] = "File"
	uq := bson.M{"$set": bson.M{"content": j}}
	// update file
	err := c.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("file '%s' content couldn't be updated.\n%s", p, err)
		return bfs.ErrorResponse(errors.New(msg))
	}

	return bfs.OKResponse(true)
}

func (m *MongodbBFS) BQLSearch(db string, query map[string]interface{}) bfs.BFSResponse {
	// check fields and paths
	fields, hasfields := query["fields"].([]string)
	paths, haspaths := query["dirs"].([]string)
	where, haswhere := query["where"].(map[string]interface{})
	limit, haslimit := query["limit"].(int64)
	sort, hassort := query["sort"].([]string)
	_, hascount := query["count"]
	distinct, hasdistinct := query["distinct"].(string)

	if !hasfields && !haspaths {
		return bfs.ErrorResponse(errors.New("Invalid select query: No fields or document paths."))
	}

	// build mongodb query
	q := bson.M{
		"__header__.parent": bson.M{"$in": paths},
		"__header__.type":   "File"} // make sure return item is file
	if haswhere {
		_and := where["and"].([]map[string]interface{})
		if len(_and) > 0 {
			q["$and"] = _and
		}
		_or := where["or"].([]map[string]interface{})
		if len(_or) > 0 {
			q["$or"] = _or
		}
	}

	// get collection
	c := m.getBFSCollection(db)

	// run query
	tmp := c.Find(q)
	// check count
	if hascount {
		count, err := tmp.Count()
		if err != nil {
			return bfs.ErrorResponse(err)
		}
		return bfs.OKResponse(count)
	}
	// check distinct
	if hasdistinct {
		var distinctlist interface{}
		err := tmp.Distinct(distinct, &distinctlist)
		if err != nil {
			return bfs.ErrorResponse(err)
		}
		return bfs.OKResponse(distinctlist)
	}
	// check limit
	if haslimit {
		tmp = tmp.Limit(int(limit))
	}
	// check sort
	if hassort {
		tmp = tmp.Sort(sort...)
	}
	// filter fields
	if hasfields && len(fields) > 0 {
		_flds := bson.M{"__header__": 1}
		for _, item := range fields {
			_flds[item] = 1
		}
		tmp = tmp.Select(_flds)
	}

	// get results
	var item bson.M
	itemlist := []interface{}{}
	i := tmp.Iter()
	for i.Next(&item) {
		_parent := item["__header__"].(bson.M)["parent"].(string)
		_name := item["__header__"].(bson.M)["name"].(string)
		_path := path.Join(_parent, _name)
		_data := item["content"].(bson.M)
		itemlist = append(itemlist, bson.M{"path": _path, "content": _data})
	}

	return bfs.OKResponse(itemlist)
}

func (m *MongodbBFS) BQLSet(db string, query map[string]interface{}) bfs.BFSResponse {
	// check fields and paths
	fields, hasfields := query["fields"].(map[string]interface{})
	incr_fields, hasincr := query["incr"].(map[string]interface{})
	paths, haspaths := query["dirs"].([]string)
	where, haswhere := query["where"].(map[string]interface{})

	if !hasfields && !haspaths {
		return bfs.ErrorResponse(errors.New("Invalid set command: No fields or document paths."))
	}

	// build query
	q := bson.M{"__header__.parent": bson.M{"$in": paths}, "__header__.type": "File"}
	if haswhere {
		_and := where["and"].([]map[string]interface{})
		if len(_and) > 0 {
			q["$and"] = _and
		}
		_or := where["or"].([]map[string]interface{})
		if len(_or) > 0 {
			q["$or"] = _or
		}
	}
	// build update query
	uquery := bson.M{"$set": fields}
	if hasincr {
		uquery["$inc"] = incr_fields
	}

	// get collection
	c := m.getBFSCollection(db)

	// run query
	info, err := c.UpdateAll(q, uquery)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(info.Updated)
}

func (m *MongodbBFS) BQLUnset(db string, query map[string]interface{}) bfs.BFSResponse {
	// check fields and paths
	fields, hasfields := query["fields"].(map[string]interface{})
	paths, haspaths := query["dirs"].([]string)
	where, haswhere := query["where"].(map[string]interface{})

	if !hasfields && !haspaths {
		return bfs.ErrorResponse(errors.New("Invalid unset command: No fields or document paths."))
	}

	// build query
	q := bson.M{"__header__.parent": bson.M{"$in": paths}, "__header__.type": "File"}
	if haswhere {
		_and := where["and"].([]map[string]interface{})
		if len(_and) > 0 {
			q["$and"] = _and
		}
		_or := where["or"].([]map[string]interface{})
		if len(_or) > 0 {
			q["$or"] = _or
		}
	}
	// build update query
	uq := bson.M{"$unset": fields}

	// get collection
	c := m.getBFSCollection(db)

	// run query
	info, err := c.UpdateAll(q, uq)
	if err != nil {
		return bfs.ErrorResponse(err)
	}

	return bfs.OKResponse(info.Updated)
}

func init() {
	bfs.Register("mongodb", NewMongodbBFS())
}
