package mongo

import (
	//"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/johnwilson/bytengine"
	"github.com/johnwilson/bytengine/filesystem"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

const (
	DatabasePrefix       = "bfs_"
	FileSystemCollection = "bfs"
	CounterCollection    = "bytengine.counters"
)

type FileSystem struct {
	session *mgo.Session
	bstore  bytengine.ByteStore
}

type SimpleResultItem struct {
	Header  NodeHeader  `bson:"__header__"`
	AHeader BytesHeader `bson:"__bytes__"`
	Id      string      `bson:"_id"`
}

type CounterItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

/*
============================================================================
    Private Methods
============================================================================
*/

func makeRootDir() (*Directory, error) {
	id, err := filesystem.NewNodeID()
	if err != nil {
		return nil, err
	}
	dt := filesystem.FormatDatetime(time.Now())
	h := NodeHeader{"/", "Directory", true, dt, ""}
	r := &Directory{h, id}
	return r, nil
}

func (m *FileSystem) existsDocument(p string, c *mgo.Collection) (SimpleResultItem, bool) {
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		// log error
		return ri, false
	}
	return ri, true
}

func (m *FileSystem) copyDirectoryDocument(d *Directory, newprefix, oldprefix, newname string, c *mgo.Collection) error {
	// update parent path prefix with new prefix
	_parent_path := d.Header.Parent
	_parent_path = strings.Replace(_parent_path, oldprefix, newprefix, 1)

	// update header info
	id, err := filesystem.NewNodeID()
	if err != nil {
		return err
	}

	d.Header.Parent = _parent_path
	if newname != "" {
		err = filesystem.ValidateDirName(newname)
		if err != nil {
			return err
		}
		d.Header.Name = newname
	}
	d.Header.Created = filesystem.FormatDatetime(time.Now())
	d.Id = id
	// save to mongodb
	err = c.Insert(&d)
	if err != nil {
		return err
	}

	return nil
}

func (m *FileSystem) copyFileDocument(f *File, newprefix, oldprefix, newname string, c *mgo.Collection) error {
	// update parent path prefix with new prefix
	_parent_path := f.Header.Parent
	_parent_path = strings.Replace(_parent_path, oldprefix, newprefix, 1)

	// update header info
	// both the original and copy will point to the same attachment id
	// in the bst
	id, err := filesystem.NewNodeID()
	if err != nil {
		return err
	}
	f.Header.Parent = _parent_path
	f.Header.Created = filesystem.FormatDatetime(time.Now())
	if newname != "" {
		err = filesystem.ValidateFileName(newname)
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

func (m *FileSystem) findPathQuery(p string) bson.M {
	// build query
	var q bson.M
	if p == "/" {
		q = bson.M{"__header__.parent": "", "__header__.name": "/"}
	} else {
		q = bson.M{"__header__.parent": path.Dir(p), "__header__.name": path.Base(p)}
	}
	return q
}

func (m *FileSystem) findChildrenQuery(p, rgx string) bson.M {
	qre := bson.RegEx{Pattern: rgx, Options: "i"} // case insensitive regex
	q := bson.M{
		"__header__.parent": p,
		"__header__.name":   bson.M{"$regex": qre},
	}
	return q
}

func (m *FileSystem) findAllChildrenQuery(p string) bson.M {
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

func (m *FileSystem) getBFSCollection(db string) *mgo.Collection {
	actual_db := DatabasePrefix + db
	return m.session.DB(actual_db).C(FileSystemCollection)
}

func (m *FileSystem) getCounterCollection(db string) *mgo.Collection {
	actual_db := DatabasePrefix + db
	return m.session.DB(actual_db).C(CounterCollection)
}

/*
============================================================================
    BFS Interface Methods
============================================================================
*/

func (m *FileSystem) Start(config string, b *bytengine.ByteStore) error {
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

func (m *FileSystem) ClearAll() ([]string, error) {
	found := make([]string, 0)

	dbs, err := m.session.DatabaseNames()
	if err != nil {
		return found, err
	}

	for _, db := range dbs {
		if strings.HasPrefix(db, DatabasePrefix) {
			err = m.session.DB(db).DropDatabase()
			if err != nil {
				return found, err
			}
			// drop database from bytestore
			err = m.bstore.DropDatabase(db)
			if err != nil {
				return found, err
			}
			found = append(found, strings.TrimPrefix(db, DatabasePrefix))
		}
	}

	return found, nil
}

func (m *FileSystem) ListDatabase(filter string) ([]string, error) {
	found := make([]string, 0)

	r, err := regexp.Compile(filter)
	if err != nil {
		return found, err
	}

	dbs, err := m.session.DatabaseNames()
	if err != nil {
		return found, err
	}

	for _, db := range dbs {
		if strings.HasPrefix(db, DatabasePrefix) {
			db = strings.Replace(db, DatabasePrefix, "", 1) // remove prefix
			if r.MatchString(db) {
				found = append(found, db)
			}
		}
	}

	return found, nil
}

func (m *FileSystem) CreateDatabase(db string) error {
	err := filesystem.ValidateDbName(db)
	if err != nil {
		return err
	}

	// create mongodb database collection root node
	rn, err := makeRootDir()
	if err != nil {
		return err
	}

	// create mongodb database and collection and insert record
	col := m.getBFSCollection(db)

	err = col.Insert(&rn)
	if err != nil {
		return err
	}

	return nil
}

func (m *FileSystem) DropDatabase(db string) error {
	actual_db := DatabasePrefix + db

	// check if db to be deleted exists
	dbs, err := m.session.DatabaseNames()
	if err != nil {
		return err
	}
	_db_exists := false
	for _, item := range dbs {
		if item == actual_db {
			_db_exists = true
			break
		}
	}
	if !_db_exists {
		err := errors.New(fmt.Sprintf("database '%s' doesn't exist", db))
		return err
	}

	// drop db from mongodb
	err = m.session.DB(actual_db).DropDatabase()
	if err != nil {
		return err
	}

	// drop database from bst
	err = m.bstore.DropDatabase(db)
	if err != nil {
		return err
	}

	return nil
}

func (m *FileSystem) NewDir(p, db string) error {
	// check path
	p = path.Clean(p)
	if p == "/" {
		err := errors.New("root directory already exists")
		return err
	}
	_name := path.Base(p)
	_parent := path.Dir(p)
	err := filesystem.ValidateDirName(_name)
	if err != nil {
		return err
	}
	// check if parent directory exists
	q := m.findPathQuery(_parent)

	// get collection
	c := m.getBFSCollection(db)
	var _parentdir Directory
	// find record
	err = c.Find(q).One(&_parentdir)
	if err != nil {
		return err
	}
	if _parentdir.Header.Type != "Directory" {
		err = fmt.Errorf("directory '%s' couldn't be created: destination isn't a directory.", p)
		return err
	}
	// check if name already taken
	q = m.findPathQuery(p)
	_count, err := c.Find(q).Count()
	if err != nil {
		return err
	}
	if _count > 0 {
		err = fmt.Errorf("directory '%s' already exists", p)
		return err
	}

	// create directory
	id, err := filesystem.NewNodeID()
	if err != nil {
		return err
	}
	dt := filesystem.FormatDatetime(time.Now())
	h := NodeHeader{_name, "Directory", false, dt, _parent}
	_dir := Directory{h, id}
	// insert node into mongodb
	err = c.Insert(&_dir)
	if err != nil {
		return err
	}

	return nil
}

func (m *FileSystem) NewFile(p, db string, j map[string]interface{}) error {
	// check path
	p = path.Clean(p)
	_name := path.Base(p)
	_parent := path.Dir(p)
	err := filesystem.ValidateFileName(_name)
	if err != nil {
		return err
	}
	// check if parent directory exists
	q := m.findPathQuery(_parent)

	// get collection
	c := m.getBFSCollection(db)
	var _parentdir Directory
	// find record
	err = c.Find(q).One(&_parentdir)
	if err != nil {
		err = fmt.Errorf("destination directory not found: %s", err)
		return err
	}
	if _parentdir.Header.Type != "Directory" {
		err = errors.New("destination isn't a directory")
		return err
	}
	// check if name already taken
	q = m.findPathQuery(p)
	_count, err := c.Find(q).Count()
	if err != nil {
		return err
	}
	if _count > 0 {
		err = fmt.Errorf("file '%s' already exists", p)
		return err
	}

	// create file
	id, err := filesystem.NewNodeID()
	if err != nil {
		return err
	}
	dt := filesystem.FormatDatetime(time.Now())
	h := NodeHeader{_name, "File", false, dt, _parent}
	a := BytesHeader{"", "", 0}
	_file := File{h, a, id, j}
	// insert node into mongodb
	err = c.Insert(&_file)
	if err != nil {
		return err
	}

	return nil
}

func (m *FileSystem) ListDir(p, filter, db string) (map[string][]string, error) {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// find path
	q := m.findPathQuery(p)
	n, err := c.Find(q).Count()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		err = fmt.Errorf("path '%s' doesn't exist.", p)
		return nil, err
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
		return nil, err
	}
	res := map[string][]string{
		"dirs":   dirs,
		"files":  files,
		"bfiles": bfiles,
	}

	return res, nil
}

func (m *FileSystem) ReadJson(p, db string, fields []string) (interface{}, error) {
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
			return nil, err
		}
	} else {
		_flds := bson.M{"__header__": 1}
		for _, item := range fields {
			_flds["content."+item] = 1
		}
		err := c.Find(q).Select(_flds).One(&r)
		if err != nil {
			return nil, err
		}
	}

	return r["content"], nil
}

func (m *FileSystem) Delete(p, db string) error {
	// check path
	p = path.Clean(p)
	if p == "/" {
		err := errors.New("root directory can't be deleted")
		return err
	}

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		return err
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
			return err
		}
		// delete all children
		_, err := c.RemoveAll(q)
		if err != nil {
			return err
		}
		// delete attachments from bst
		for _, item := range _attchs {
			err = m.bstore.Delete(db, item)
			if err != nil {
				return err
			}
		}
		// delete directory
		err = c.RemoveId(ri.Id)
		if err != nil {
			return err
		}

	} else {
		if ri.AHeader.Filepointer != "" {
			// delete attachment from bst
			err = m.bstore.Delete(db, ri.AHeader.Filepointer)
			if err != nil {
				return err
			}
		}
		// delete file
		err = c.RemoveId(ri.Id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileSystem) Rename(p, newname, db string) error {
	// check path
	p = path.Clean(p)
	if p == "/" {
		err := errors.New("root directory cannot be renamed.")
		return err
	}

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		return err
	}

	if ri.Header.Type == "Directory" {
		// check if name is valid
		if err = filesystem.ValidateDirName(newname); err != nil {
			return err
		}
		// check if name isn't already in use
		np := path.Join(path.Dir(p), newname)
		q = m.findPathQuery(np)
		_count, err := c.Find(q).Count()
		if err != nil {
			return err
		}
		if _count > 0 {
			err := fmt.Errorf("directory '%s' already exists", np)
			return err
		}
		// get affected parent directories
		q = m.findAllChildrenQuery(p)
		var _dirs []string
		err = c.Find(q).Distinct("__header__.parent", &_dirs)
		if err != nil {
			return err
		}
		for _, item := range _dirs {
			newparent := strings.Replace(item, p, np, 1)
			q = bson.M{"__header__.parent": item}
			uq := bson.M{"$set": bson.M{"__header__.parent": newparent}}
			_, e := c.UpdateAll(q, uq)
			if e != nil {
				return e
			}
		}
		// rename directory by updating field
		q = bson.M{"$set": bson.M{"__header__.name": newname}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}

	} else {
		// check if name is valid
		if err = filesystem.ValidateFileName(newname); err != nil {
			return err
		}
		// check if name isn't already in use
		np := path.Join(path.Dir(p), newname)
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			return e
		}
		if _count > 0 {
			err = fmt.Errorf("file '%s' already exists", np)
			return err
		}
		// rename file by updating field
		q = bson.M{"$set": bson.M{"__header__.name": newname}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileSystem) Move(from, to, db string) error {
	// check path
	from = path.Clean(from) // from
	to = path.Clean(to)     // to
	if from == "/" {
		err := errors.New("root directory can't be moved")
		return err
	}
	// check illegal move operation i.e. moving from parent to sub directory
	if strings.HasPrefix(to, from) {
		err := errors.New("illegal move operation.")
		return err
	}

	// get collection
	c := m.getBFSCollection(db)

	// check if destination dir exists
	_doc_dest, _exists_dest := m.existsDocument(to, c)
	if !_exists_dest {
		err := errors.New("Destination directory doesn't exist")
		return err
	}
	if _doc_dest.Header.Type != "Directory" {
		err := errors.New("Destination must be a directory")
		return err
	}

	// get file or directory if it exists
	q := m.findPathQuery(from)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		return err
	}

	if ri.Header.Type == "Directory" {
		// check if name isn't already in use
		np := path.Join(to, path.Base(from))
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			return e
		}
		if _count > 0 {
			err = fmt.Errorf("directory '%s' already exists", np)
			return err
		}
		// get affected parent directories
		q = m.findAllChildrenQuery(from)
		var _dirs []string
		err = c.Find(q).Distinct("__header__.parent", &_dirs)
		if err != nil {
			return err
		}
		for _, item := range _dirs {
			newparent := strings.Replace(item, from, np, 1)
			q = bson.M{"__header__.parent": item}
			uq := bson.M{"$set": bson.M{"__header__.parent": newparent}}
			_, e := c.UpdateAll(q, uq)
			if e != nil {
				return e
			}
		}
		// move directory by updating parent field
		q = bson.M{"$set": bson.M{"__header__.parent": to}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}

	} else {
		// check if name isn't already in use
		np := path.Join(to, path.Base(from))
		q = m.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			return e
		}
		if _count > 0 {
			err = fmt.Errorf("file '%s' already exists", np)
			return err
		}
		// rename file by updating field
		q = bson.M{"$set": bson.M{"__header__.parent": to}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileSystem) Copy(from, to, db string) error {
	// setup paths
	_from_doc_path := path.Clean(from)
	_from_doc_parent_path := path.Dir(_from_doc_path)
	_to_doc_path := path.Clean(to)
	_to_doc_parent_path := path.Dir(_to_doc_path)
	_to_doc_name := path.Base(_to_doc_path)

	if _from_doc_path == "/" {
		err := errors.New("root directory cannot be copied.")
		return err
	}
	// check illegal copy operation i.e. copy from parent to sub directory
	if strings.HasPrefix(_to_doc_parent_path, _from_doc_path) {
		err := errors.New("illegal copy operation.")
		return err
	}

	// get collection
	c := m.getBFSCollection(db)

	// check if destination dir exists
	_doc_dest, _exists_dest := m.existsDocument(_to_doc_parent_path, c)
	if !_exists_dest {
		err := errors.New("Destination directory doesn't exist")
		return err
	}
	if _doc_dest.Header.Type != "Directory" {
		err := errors.New("Destination must be a directory")
		return err
	}

	// check if item to copy exists
	_doc, _exists := m.existsDocument(_from_doc_path, c)
	if !_exists {
		err := fmt.Errorf("'%s' doesn't exist", _from_doc_path)
		return err
	}

	// check if name isn't already in use
	_, _exists = m.existsDocument(_to_doc_path, c)
	if _exists {
		err := fmt.Errorf("'%s' already exists.", _to_doc_path)
		return err
	}

	if _doc.Header.Type == "Directory" {
		// get full document
		var _main_dir Directory
		err := c.FindId(_doc.Id).One(&_main_dir)
		if err != nil {
			return err
		}

		// copy directory
		err = m.copyDirectoryDocument(&_main_dir, _to_doc_parent_path, _from_doc_parent_path, _to_doc_name, c)
		if err != nil {
			return err
		}

		// get affected dirs
		q := m.findAllChildrenQuery(_from_doc_path)
		q["__header__.type"] = "Directory"
		var _tmpdir Directory
		i := c.Find(q).Iter()
		for i.Next(&_tmpdir) {
			err = m.copyDirectoryDocument(&_tmpdir, _to_doc_path, _from_doc_path, "", c)
			if err != nil {
				return err
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
				return err
			}
		}

	} else {
		// get full document
		var _filedoc File
		err := c.FindId(_doc.Id).One(&_filedoc)
		if err != nil {
			return err
		}

		// copy file
		err = m.copyFileDocument(&_filedoc, _to_doc_parent_path, _from_doc_parent_path, _to_doc_name, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileSystem) Info(p, db string) (map[string]interface{}, error) {
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		return nil, err
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
			return nil, e
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

	return _info, nil
}

func (m *FileSystem) FileAccess(p, db string, protect bool) error {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		return err
	}

	if ri.Header.Type == "Directory" {
		// update directory access by updating field
		q = bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}
		// automatically cascade to sub nodes
		q = m.findAllChildrenQuery(p)
		uq := bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		_, e := c.UpdateAll(q, uq)
		if e != nil {
			return e
		}

	} else {
		// update file access by updating field
		q = bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileSystem) SetCounter(counter, action string, value int64, db string) (int64, error) {
	// update value 'v'
	nv := math.Abs(float64(value))
	value = int64(nv)

	// get collection
	c := m.getCounterCollection(db)

	// check if counter exists
	q := bson.M{"name": counter}
	num, err := c.Find(q).Count()
	if err != nil {
		return 0, err
	}
	// if not exists create new counter
	if num < 1 {
		err = filesystem.ValidateCounterName(counter)
		if err != nil {
			return 0, err
		}

		doc := bson.M{"name": counter, "value": value}
		err = c.Insert(doc)
		if err != nil {
			return 0, err
		}

		return int64(value), nil
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
		err = errors.New("counter action failed")
		return 0, err
	}

	var r struct {
		Name  string
		Value int64
	}
	_, err = c.Find(q).Apply(cq, &r)
	if err != nil {
		return 0, err
	}

	return r.Value, nil
}

func (m *FileSystem) ListCounter(filter, db string) (map[string]int64, error) {
	// get collection
	c := m.getCounterCollection(db)

	list := make(map[string]int64)
	qre := bson.RegEx{Pattern: filter, Options: "i"} // case insensitive regex
	q := bson.M{"name": bson.M{"$regex": qre}}
	iter := c.Find(q).Iter()
	var ci CounterItem
	for iter.Next(&ci) {
		list[ci.Name] = int64(ci.Value)
	}
	err := iter.Close()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (m *FileSystem) WriteBytes(p, ap, db string) (int64, error) {
	var nbytes int64 // number of bytes written

	// check path
	p = path.Clean(p)
	_, err := os.Stat(ap)
	if err != nil {
		return nbytes, err
	}

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		return nbytes, err
	}

	if ri.Header.Type == "Directory" {
		err = errors.New("command only valid for files.")
		return nbytes, err

	} else {
		// if bytes already writen then update else create new
		isnew := true
		if ri.AHeader.Filepointer != "" {
			isnew = false
		}
		// open attachment and add to bst
		file, err := os.Open(ap)
		if err != nil {
			return nbytes, err
		}

		var q bson.M // query

		if isnew {
			info, err := m.bstore.Add(db, file)
			if err != nil {
				return nbytes, err
			}
			nbytes = info["size"].(int64)

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
				return nbytes, err
			}
			nbytes = info["size"].(int64)

			// update file access by updating field
			q = bson.M{
				"$set": bson.M{
					"__bytes__.size": info["size"].(int64),
					"__bytes__.mime": info["mime"].(string),
				}}
		}

		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return nbytes, err
		}
	}

	return nbytes, nil
}

func (m *FileSystem) ReadBytes(fp, db string) (string, error) {
	var id string // bytestore file id
	// check path
	fp = path.Clean(fp)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(fp)
	var ri File
	err := c.Find(q).One(&ri)
	if err != nil {
		return id, err
	}

	if ri.Header.Type == "Directory" {
		err = errors.New("command only valid for files.")
		return id, err

	}
	id = ri.AHeader.Filepointer
	if len(id) == 0 {
		err = errors.New("byte layer is empty")
		return id, err
	}
	return id, nil
}

func (m *FileSystem) DirectAccess(fp, db, layer string) (map[string]interface{}, string, error) {
	var content map[string]interface{} // file json data
	var id string                      // bytestore file id

	// check path
	fp = path.Clean(fp)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(fp)
	var ri File
	err := c.Find(q).One(&ri)
	if err != nil {
		err = errors.New("file not found")
		return content, id, err
	}

	if ri.Header.Type == "Directory" {
		err = errors.New("command only valid for files.")
		return content, id, err

	}

	if !ri.Header.IsPublic {
		err = errors.New("file isn't public")
		return content, id, err
	}
	content = ri.Content
	id = ri.AHeader.Filepointer

	switch layer {
	case "json":
		return content, id, nil
	case "bytes":
		if len(id) == 0 {
			err = errors.New("byte layer is empty")
			return nil, id, err
		}
		return nil, id, nil
	}

	return nil, "", errors.New("data not found")
}

func (m *FileSystem) DeleteBytes(p, db string) error {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file or directory if it exists
	q := m.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		return err
	}

	if ri.Header.Type == "Directory" {
		err = errors.New("command only valid for files.")
		return err

	} else {
		// delete attachment
		if ri.AHeader.Filepointer != "" {
			// delete attachment
			err = m.bstore.Delete(db, ri.AHeader.Filepointer)
			if err != nil && os.IsExist(err) {
				return err
			}
		}
		// update file access by updating field
		q = bson.M{"$set": bson.M{"__bytes__.filepointer": ""}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FileSystem) UpdateJson(p, db string, j map[string]interface{}) error {
	// check path
	p = path.Clean(p)

	// get collection
	c := m.getBFSCollection(db)

	// get file if it exists
	q := m.findPathQuery(p)
	q["__header__.type"] = "File"
	uq := bson.M{"$set": bson.M{"content": j}}
	// update file
	return c.Update(q, uq)
}

func (m *FileSystem) BQLSearch(db string, query map[string]interface{}) (interface{}, error) {
	// check fields and paths
	fields, hasfields := query["fields"].([]string)
	paths, haspaths := query["dirs"].([]string)
	where, haswhere := query["where"].(map[string]interface{})
	limit, haslimit := query["limit"].(int64)
	sort, hassort := query["sort"].([]string)
	_, hascount := query["count"]
	distinct, hasdistinct := query["distinct"].(string)

	if !hasfields && !haspaths {
		err := errors.New("Invalid select query: No fields or document paths.")
		return nil, err
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
			return nil, err
		}
		return count, nil
	}
	// check distinct
	if hasdistinct {
		var distinctlist interface{}
		err := tmp.Distinct(distinct, &distinctlist)
		if err != nil {
			return nil, err
		}
		return distinctlist, nil
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

	return itemlist, nil
}

func (m *FileSystem) BQLSet(db string, query map[string]interface{}) (int, error) {
	var count int // number of items updated

	// check fields and paths
	fields, hasfields := query["fields"].(map[string]interface{})
	incr_fields, hasincr := query["incr"].(map[string]interface{})
	paths, haspaths := query["dirs"].([]string)
	where, haswhere := query["where"].(map[string]interface{})

	if !hasfields && !haspaths {
		err := errors.New("Invalid set command: No fields or document paths.")
		return count, err
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
		return count, err
	}
	count = info.Updated

	return count, nil
}

func (m *FileSystem) BQLUnset(db string, query map[string]interface{}) (int, error) {
	var count int // number of items updated

	// check fields and paths
	fields, hasfields := query["fields"].(map[string]interface{})
	paths, haspaths := query["dirs"].([]string)
	where, haswhere := query["where"].(map[string]interface{})

	if !hasfields && !haspaths {
		err := errors.New("Invalid unset command: No fields or document paths.")
		return count, err
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
		return count, err
	}

	count = info.Updated

	return count, nil
}

func init() {
	bytengine.RegisterFileSystem("mongodb", NewFileSystem())
}
