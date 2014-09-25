package modules

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
	//"log"

	"github.com/nu7hatch/gouuid"
	"github.com/vmihailenco/redis"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// Bytengine Filesystem
type BFS struct {
	config   *Configuration
	mongo    *mgo.Session
	redisMan *RedisManager
}

func NewBFS(c *Configuration, m *mgo.Session, r *RedisManager) *BFS {
	b := &BFS{
		config:   c,
		mongo:    m,
		redisMan: r,
	}
	return b
}

// BFS Node Header
type NodeHeader struct {
	Name     string
	Type     string
	IsPublic bool `json:"ispublic"`
	Created  string
	Parent   string
}

// BFS Attachment Header
type AttachmentHeader struct {
	Filepointer string `json:"filepointer"`
	Size        int64  `json:"size"`
	Mime        string `json:"mime"`
}

// BFS Directory
type Directory struct {
	Header NodeHeader `bson:"__header__"`
	Id     string     `bson:"_id"`
}

func formatDatetime(t *time.Time) (d string) {
	f := "%d:%02d:%02d-%02d:%02d:%02d.%03d"
	dt := fmt.Sprintf(f,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond()/100000)
	return dt
}

func makeUUID() (string, error) {
	tmp, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	id := strings.Replace(tmp.String(), "-", "", -1) // remove dashes
	return id, nil
}

func makeRootDir() (*Directory, error) {
	uuid, err := makeUUID()
	if err != nil {
		msg := fmt.Sprintf("db root dir couldn't be created.\n%s", err)
		return nil, errors.New(msg)
	}
	nw := time.Now()
	dt := formatDatetime(&nw)
	h := NodeHeader{"/", "Directory", true, dt, ""}
	id := fmt.Sprintf("%s:%s", uuid, dt)
	r := &Directory{h, id}
	return r, nil
}

// BFS File
type File struct {
	Header  NodeHeader       `bson:"__header__"`
	AHeader AttachmentHeader `bson:"__attch__"`
	Id      string           `bson:"_id"`
	Content map[string]interface{}
}

func (fs *BFS) IsReservedDb(d string) bool {
	for _, item := range fs.config.Bfs.ReservedDbs {
		if d == item {
			return true
		}
	}
	return false
}

// Clear all databases in mongodb
func (fs *BFS) clearMongodb() (err error) {
	// clear mongodb
	dbs, err := fs.mongo.DatabaseNames()
	if err != nil {
		return err
	}
	for _, item := range dbs {
		// skip reserved dbs
		if !fs.IsReservedDb(item) {
			err := fs.mongo.DB(item).DropDatabase()
			if err != nil {
				return err
			}
		}
	}

	// clear users
	db := fs.mongo.DB(fs.config.Bfs.SystemDb)
	col_list, err := db.CollectionNames()
	if err != nil {
		return err
	}
	for _, item := range col_list {
		if item == fs.config.Bfs.SecurityCol {
			col := db.C(fs.config.Bfs.SecurityCol)
			err = col.DropCollection()
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (fs *BFS) clearRedis() (err error) {
	// any of the clients can be used
	rclient := fs.redisMan.DbConnect["bfs"]
	status := rclient.FlushAll()
	err = status.Err()
	return err
}

func (fs *BFS) clearDirectories() (err error) {
	_dirs := []string{
		fs.config.Bfs.AttachmentsRoot,
		fs.config.Web.UploadDirectory,
	}

	// delete
	for _, item := range _dirs {
		err = os.RemoveAll(item)
		if err != nil {
			return err
		}
	}

	// create
	for _, item := range _dirs {
		err = os.MkdirAll(item, 0775)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *BFS) RebuildServer() (err error) {
	err = fs.clearMongodb()
	if err != nil {
		return err
	}
	err = fs.clearRedis()
	if err != nil {
		return err
	}
	err = fs.clearDirectories()
	if err != nil {
		return err
	}

	return nil
}

func (fs *BFS) ListDatabases(rgx string) (d []interface{}, err error) {
	dbs, err := fs.mongo.DatabaseNames()
	if err != nil {
		return nil, err
	}

	_list := []interface{}{}
	r, err := regexp.Compile(rgx)
	if err != nil {
		return nil, err
	}

	for _, item := range dbs {
		if !fs.IsReservedDb(item) {
			// regex match
			if r.MatchString(item) {
				_list = append(_list, item)
			}
		}
	}
	return _list, nil
}

func (fs *BFS) validateDbName(d string) (err error) {
	d = strings.ToLower(d)
	if fs.IsReservedDb(d) {
		msg := fmt.Sprintf("database name '%s' is reserved.")
		return errors.New(msg)
	}
	// regex verification
	r, err := regexp.Compile("^[a-z][a-z0-9_]{1,20}$")
	if err != nil {
		return err
	}
	if r.MatchString(d) {
		return nil
	}
	msg := fmt.Sprintf("database name '%s' isn't valid.", d)
	return errors.New(msg)
}

func (fs *BFS) MakeDatabase(db string) (err error) {
	err = fs.validateDbName(db)
	if err != nil {
		return err
	}

	// check if name available in attachments dir
	p := path.Join(fs.config.Bfs.AttachmentsRoot, db)
	_, e := os.Stat(p)
	if e == nil {
		msg := "database already exists in attachments dir."
		return errors.New(msg)
	}
	// create new database dir
	err = os.MkdirAll(p, 0775)
	if err != nil {
		msg := fmt.Sprintf("database '%s' couldn't be created:\n%s", db, err)
		return errors.New(msg)
	}
	// create mongodb database collection root node
	rn, e := makeRootDir()
	if e != nil {
		msg := fmt.Sprintf("database '%s' couldn't be created:\n%s", db, e)
		return errors.New(msg)
	}
	// create counter container
	cnter := bson.M{"counters": bson.M{}}
	// create mongodb database and collection and insert record
	col := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	err = col.Insert(&rn, &cnter)
	if err != nil {
		msg := fmt.Sprintf("database '%s' couldn't be created:\n%s", db, err)
		return errors.New(msg)
	}
	return nil
}

func (fs *BFS) RemoveDatabase(db string) (err error) {
	// check if db to be deleted is reserved
	if fs.IsReservedDb(db) {
		return errors.New("system database cannot be deleted.")
	}
	// check if db to be deleted exists
	dbs, err := fs.mongo.DatabaseNames()
	if err != nil {
		return err
	}
	_db_exists := false
	for _, item := range dbs {
		if item == db {
			_db_exists = true
			break
		}
	}
	if !_db_exists {
		msg := fmt.Sprintf("database '%s' doesn't exist", db)
		return errors.New(msg)
	}
	// drop db from mongodb
	err = fs.mongo.DB(db).DropDatabase()
	if err != nil {
		msg := fmt.Sprintf("database '%s' couldn't be deleted.\n%s", db, err)
		return errors.New(msg)
	}

	// delete database from system users database list
	col := fs.mongo.DB(fs.config.Bfs.SystemDb).C(fs.config.Bfs.SecurityCol)
	q := map[string]interface{}{}
	uq := bson.M{"$pull": bson.M{"databases": db}}
	_, e := col.UpdateAll(q, uq)
	if e != nil {
		msg := fmt.Sprintf("database '%s' couldn't be deleted from users database list.\n%s", db, e)
		return errors.New(msg)
	}

	// delete db dir in attachments
	p := path.Join(fs.config.Bfs.AttachmentsRoot, db)
	err = os.RemoveAll(p)
	if err != nil {
		msg := fmt.Sprintf("database '%s' couldn't be deleted:\n%s", db, err)
		return errors.New(msg)
	}

	return nil
}

func (fs *BFS) MakeDir(p, db string) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)
	if p == "/" {
		return errors.New("root directory already exists")
	}
	_name := path.Base(p)
	_parent := path.Dir(p)
	err = fs.validateDirName(_name)
	if err != nil {
		return err
	}
	// check if parent directory exists
	q := fs.findPathQuery(_parent)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)
	var _parentdir Directory
	// find record
	err = c.Find(q).One(&_parentdir)
	if err != nil {
		msg := fmt.Sprintf("directory '%s' couldn't be created: destination directory not found\n%s", p, err)
		return errors.New(msg)
	}
	if _parentdir.Header.Type != "Directory" {
		msg := fmt.Sprintf("directory '%s' couldn't be created: destination isn't a directory.", p)
		return errors.New(msg)
	}
	// check if name already taken
	q = fs.findPathQuery(p)
	_count, err := c.Find(q).Count()
	if err != nil {
		msg := fmt.Sprintf("directory '%s' couldn't be created.\n%s", p, err)
		return errors.New(msg)
	}
	if _count > 0 {
		msg := fmt.Sprintf("directory '%s' already exists", p)
		return errors.New(msg)
	}

	// create directory
	uuid, err := makeUUID()
	if err != nil {
		msg := fmt.Sprintf("directory '%s' couldn't be created.\n%s", p, err)
		return errors.New(msg)
	}
	nw := time.Now()
	dt := formatDatetime(&nw)
	h := NodeHeader{_name, "Directory", false, dt, _parent}
	id := fmt.Sprintf("%s:%s", uuid, dt)
	_dir := Directory{h, id}
	// insert node into mongodb
	err = c.Insert(&_dir)
	if err != nil {
		msg := fmt.Sprintf("directory '%s' couldn't be created.\n%s", p, err)
		return errors.New(msg)
	}

	return nil
}

func (fs *BFS) MakeFile(p, db string, j map[string]interface{}) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)
	_name := path.Base(p)
	_parent := path.Dir(p)
	err = fs.validateFileName(_name)
	if err != nil {
		return err
	}
	// check if parent directory exists
	q := fs.findPathQuery(_parent)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)
	var _parentdir Directory
	// find record
	err = c.Find(q).One(&_parentdir)
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created: destination directory not found\n%s", p, err)
		return errors.New(msg)
	}
	if _parentdir.Header.Type != "Directory" {
		msg := fmt.Sprintf("file '%s' couldn't be created: destination isn't a directory.", p)
		return errors.New(msg)
	}
	// check if name already taken
	q = fs.findPathQuery(p)
	_count, err := c.Find(q).Count()
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created.\n%s", p, err)
		return errors.New(msg)
	}
	if _count > 0 {
		msg := fmt.Sprintf("file '%s' already exists", p)
		return errors.New(msg)
	}

	// create file
	uuid, err := makeUUID()
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created.\n%s", p, err)
		return errors.New(msg)
	}
	nw := time.Now()
	dt := formatDatetime(&nw)
	h := NodeHeader{_name, "File", false, dt, _parent}
	a := AttachmentHeader{"", 0, ""}
	id := fmt.Sprintf("%s:%s", uuid, dt)
	_file := File{h, a, id, j}
	// insert node into mongodb
	err = c.Insert(&_file)
	if err != nil {
		msg := fmt.Sprintf("file '%s' couldn't be created.\n%s", p, err)
		return errors.New(msg)
	}

	return nil
}

type SimpleResultItem struct {
	Header  NodeHeader       `bson:"__header__"`
	AHeader AttachmentHeader `bson:"__attch__"`
	Id      string           `bson:"_id"`
}

func (fs *BFS) DirectoryListing(p, rgx, db string) (map[string][]interface{}, error) {
	// check path
	p = path.Clean(p)
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// find path
	q := fs.findPathQuery(p)
	n, err := c.Find(q).Count()
	if err != nil {
		msg := fmt.Sprintf("error while trying to find the path: %s.\n%s", p, err)
		return nil, errors.New(msg)
	}
	if n != 1 {
		msg := fmt.Sprintf("path '%s' doesn't exist.", p)
		return nil, errors.New(msg)
	}

	// find children
	q = fs.findChildrenQuery(p, rgx)
	i := c.Find(q).Sort("__header__.name").Iter()
	var ri SimpleResultItem
	dirs := []interface{}{}
	files := []interface{}{}
	afiles := []interface{}{} // files with attachments

	for i.Next(&ri) {
		if ri.Header.Type == "Directory" {
			dirs = append(dirs, ri.Header.Name)
		} else {
			if ri.AHeader.Filepointer == "" {
				files = append(files, ri.Header.Name)
			} else {
				afiles = append(afiles, ri.Header.Name)
			}
		}
	}
	err = i.Err()
	if err != nil {
		msg := fmt.Sprintf("error while trying directory listing for: %s\n%s", p, err)
		return nil, errors.New(msg)
	}
	res := map[string][]interface{}{
		"dirs":   dirs,
		"files":  files,
		"afiles": afiles,
	}

	return res, nil
}

func (fs *BFS) GetFileContent(p, db string, fields []string) (bson.M, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file if it exists
	q := fs.findPathQuery(p)
	q["__header__.type"] = "File"

	var r bson.M
	if len(fields) == 0 {
		e := c.Find(q).One(&r)
		if e != nil {
			msg := fmt.Sprintf("file '%s' content couldn't be retrieved.\n%s", p, e)
			return nil, errors.New(msg)
		}
	} else {
		_flds := bson.M{"__header__": 1}
		for _, item := range fields {
			_flds["content."+item] = 1
		}
		e := c.Find(q).Select(_flds).One(&r)
		if e != nil {
			msg := fmt.Sprintf("file '%s' content couldn't be retrieved.\n%s", p, e)
			return nil, errors.New(msg)
		}
	}

	return r["content"].(bson.M), nil
}

func (fs *BFS) Delete(p, db string) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)
	if p == "/" {
		return errors.New("root directory cannot be deleted.")
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
		return errors.New(msg)
	}
	if ri.Header.Type == "Directory" {
		// find all children
		q = fs.findAllChildrenQuery(p)
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
			return errors.New(msg)
		}
		// delete all children
		_, e := c.RemoveAll(q)
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, e)
			return errors.New(msg)
		}
		// delete attachments
		for _, item := range _attchs {
			err = os.Remove(item)
			if err != nil {
				msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
				return errors.New(msg)
			}
		}
		// delete directory
		err = c.RemoveId(ri.Id)
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
			return errors.New(msg)
		}

	} else {
		if ri.AHeader.Filepointer != "" {
			// delete attachment
			err = os.Remove(ri.AHeader.Filepointer)
			if err != nil && os.IsExist(err) {
				msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
				return errors.New(msg)
			}
		}
		// delete file
		err = c.RemoveId(ri.Id)
		if err != nil {
			msg := fmt.Sprintf("path '%s' couldn't be deleted.\n%s", p, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) Rename(p, n, db string) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)
	if p == "/" {
		return errors.New("root directory cannot be renamed.")
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("renaming '%s'failed.\n%s", p, err)
		return errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		// check if name is valid
		if fs.validateDirName(n) != nil {
			msg := fmt.Sprintf("invalid directory name: %s", n)
			return errors.New(msg)
		}
		// check if name isn't already in use
		np := path.Join(path.Dir(p), n)
		q = fs.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, e)
			return errors.New(msg)
		}
		if _count > 0 {
			msg := fmt.Sprintf("directory '%s' already exists", np)
			return errors.New(msg)
		}
		// get affected parent directories
		q = fs.findAllChildrenQuery(p)
		var _dirs []string
		err = c.Find(q).Distinct("__header__.parent", &_dirs)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, err)
			return errors.New(msg)
		}
		for _, item := range _dirs {
			newparent := strings.Replace(item, p, np, 1)
			q = bson.M{"__header__.parent": item}
			uq := bson.M{"$set": bson.M{"__header__.parent": newparent}}
			_, e := c.UpdateAll(q, uq)
			if e != nil {
				msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, e)
				return errors.New(msg)
			}
		}
		// rename directory by updating field
		q = bson.M{"$set": bson.M{"__header__.name": n}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be renamed.\n%s", p, err)
			return errors.New(msg)
		}

	} else {
		// check if name is valid
		if fs.validateFileName(n) != nil {
			msg := fmt.Sprintf("invalid file name: %s", n)
			return errors.New(msg)
		}
		// check if name isn't already in use
		np := path.Join(path.Dir(p), n)
		q = fs.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("file '%s' couldn't be renamed.\n%s", p, e)
			return errors.New(msg)
		}
		if _count > 0 {
			msg := fmt.Sprintf("file '%s' already exists", np)
			return errors.New(msg)
		}
		// rename file by updating field
		q = bson.M{"$set": bson.M{"__header__.name": n}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("file '%s' couldn't be renamed.\n%s", p, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) Move(p, d, db string) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p) // from
	d = path.Clean(d) // to
	if p == "/" {
		return errors.New("root directory cannot be moved.")
	}
	// check illegal move operation i.e. moving from parent to sub directory
	if strings.HasPrefix(d, p) {
		return errors.New("illegal move operation.")
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// check if destination dir exists
	_doc_dest, _exists_dest := fs.existsDocument(d, c)
	if !_exists_dest {
		return errors.New("Destination directory doesn't exist")
	}
	if _doc_dest.Header.Type != "Directory" {
		return errors.New("Destination must be a directory")
	}

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("moving '%s'failed.\n%s", p, err)
		return errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		// check if name isn't already in use
		np := path.Join(d, path.Base(p))
		q = fs.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", p, e)
			return errors.New(msg)
		}
		if _count > 0 {
			msg := fmt.Sprintf("directory '%s' already exists", np)
			return errors.New(msg)
		}
		// get affected parent directories
		q = fs.findAllChildrenQuery(p)
		var _dirs []string
		err = c.Find(q).Distinct("__header__.parent", &_dirs)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", p, err)
			return errors.New(msg)
		}
		for _, item := range _dirs {
			newparent := strings.Replace(item, p, np, 1)
			q = bson.M{"__header__.parent": item}
			uq := bson.M{"$set": bson.M{"__header__.parent": newparent}}
			_, e := c.UpdateAll(q, uq)
			if e != nil {
				msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", p, e)
				return errors.New(msg)
			}
		}
		// move directory by updating parent field
		q = bson.M{"$set": bson.M{"__header__.parent": d}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' couldn't be moved.\n%s", p, err)
			return errors.New(msg)
		}

	} else {
		// check if name isn't already in use
		np := path.Join(d, path.Base(p))
		q = fs.findPathQuery(np)
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("file '%s' couldn't be moved.\n%s", p, e)
			return errors.New(msg)
		}
		if _count > 0 {
			msg := fmt.Sprintf("file '%s' already exists", np)
			return errors.New(msg)
		}
		// rename file by updating field
		q = bson.M{"$set": bson.M{"__header__.parent": d}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("file '%s' couldn't be moved.\n%s", p, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) existsDocument(p string, c *mgo.Collection) (SimpleResultItem, bool) {
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err := c.Find(q).One(&ri)
	if err != nil {
		// log error
		return ri, false
	}
	return ri, true
}

func (fs *BFS) copyDirectoryDocument(d *Directory, newprefix, oldprefix, newname string, c *mgo.Collection) error {
	// update parent path prefix with new prefix
	_parent_path := d.Header.Parent
	_parent_path = strings.Replace(_parent_path, oldprefix, newprefix, 1)

	// update header info
	uuid, err := makeUUID()
	if err != nil {
		return err
	}

	nw := time.Now()
	dt := formatDatetime(&nw)
	id := fmt.Sprintf("%s:%s", uuid, dt)
	d.Header.Parent = _parent_path
	if newname != "" {
		err = fs.validateDirName(newname)
		if err != nil {
			return err
		}
		d.Header.Name = newname
	}
	d.Header.Created = dt
	d.Id = id
	// save to mongodb
	err = c.Insert(&d)
	if err != nil {
		return err
	}

	return nil
}

func (fs *BFS) copyFileDocument(f *File, newprefix, oldprefix, newname string, c *mgo.Collection) error {
	// update parent path prefix with new prefix
	_parent_path := f.Header.Parent
	_parent_path = strings.Replace(_parent_path, oldprefix, newprefix, 1)

	// update header info
	uuid, err := makeUUID()
	if err != nil {
		return err
	}

	nw := time.Now()
	dt := formatDatetime(&nw)
	id := fmt.Sprintf("%s:%s", uuid, dt)
	f.Header.Parent = _parent_path
	f.Header.Created = dt
	if newname != "" {
		err = fs.validateFileName(newname)
		if err != nil {
			return err
		}
		f.Header.Name = newname
	}
	f.Id = id

	// check if file has an attachment and copy if true
	_attch_path := f.AHeader.Filepointer
	if _attch_path != "" {
		_attch_dir := path.Dir(_attch_path)
		_new_attch_path := path.Join(_attch_dir, id)
		_, err = exec.Command("cp", _attch_path, _new_attch_path).Output()
		if err != nil {
			return err
		}
		// set new attachment filepointer
		f.AHeader.Filepointer = _new_attch_path
	}

	// save to mongodb
	err = c.Insert(&f)
	if err != nil {
		return err
	}

	return nil
}

func (fs *BFS) Copy(p, d, db string) error {
	// check database access
	e := fs.checkDbAccess(db)
	if e != nil {
		return e
	}

	// setup paths
	_from_doc_path := path.Clean(p)
	_from_doc_parent_path := path.Dir(_from_doc_path)
	_to_doc_path := path.Clean(d)
	_to_doc_parent_path := path.Dir(_to_doc_path)
	_to_doc_name := path.Base(_to_doc_path)

	if _from_doc_path == "/" {
		return errors.New("root directory cannot be copied.")
	}
	// check illegal copy operation i.e. copy from parent to sub directory
	if strings.HasPrefix(_to_doc_parent_path, _from_doc_path) {
		return errors.New("illegal copy operation.")
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// check if destination dir exists
	_doc_dest, _exists_dest := fs.existsDocument(_to_doc_parent_path, c)
	if !_exists_dest {
		return errors.New("Destination directory doesn't exist")
	}
	if _doc_dest.Header.Type != "Directory" {
		return errors.New("Destination must be a directory")
	}

	// check if item to copy exists
	_doc, _exists := fs.existsDocument(_from_doc_path, c)
	if !_exists {
		return errors.New(fmt.Sprintf("'%s' doesn't exist", p))
	}

	// check if name isn't already in use
	_, _exists = fs.existsDocument(_to_doc_path, c)
	if _exists {
		return errors.New(fmt.Sprintf("'%s' already exists.", d))
	}

	if _doc.Header.Type == "Directory" {
		// get full document
		var _main_dir Directory
		err := c.FindId(_doc.Id).One(&_main_dir)
		if err != nil {
			msg := fmt.Sprintf("copying '%s'failed.\n%s", p, err)
			return errors.New(msg)
		}

		// copy directory
		err = fs.copyDirectoryDocument(&_main_dir, _to_doc_parent_path, _from_doc_parent_path, _to_doc_name, c)
		if err != nil {
			txt := "sub-directory '%s' in directory '%s' couldn't be copied.\n%s"
			msg := fmt.Sprintf(txt, _main_dir.Header.Name, _main_dir.Header.Parent, err)
			return errors.New(msg)
		}

		// get affected dirs
		q := fs.findAllChildrenQuery(p)
		q["__header__.type"] = "Directory"
		var _tmpdir Directory
		i := c.Find(q).Iter()
		for i.Next(&_tmpdir) {
			err = fs.copyDirectoryDocument(&_tmpdir, _to_doc_path, _from_doc_path, "", c)
			if err != nil {
				txt := "sub-directory '%s' in directory '%s' couldn't be copied.\n%s"
				msg := fmt.Sprintf(txt, _tmpdir.Header.Name, _tmpdir.Header.Parent, err)
				return errors.New(msg)
			}
		}

		// get affected files
		q = fs.findAllChildrenQuery(p)
		q["__header__.type"] = "File"
		var _tmpfile File
		i = c.Find(q).Iter()
		for i.Next(&_tmpfile) {
			err = fs.copyFileDocument(&_tmpfile, _to_doc_path, _from_doc_path, "", c)
			if err != nil {
				txt := "file '%s' in directory '%s' couldn't be copied.\n%s"
				msg := fmt.Sprintf(txt, _tmpfile.Header.Name, _tmpfile.Header.Parent, err)
				return errors.New(msg)
			}
		}

	} else {
		// get full document
		var _filedoc File
		err := c.FindId(_doc.Id).One(&_filedoc)
		if err != nil {
			msg := fmt.Sprintf("copying '%s'failed.\n%s", p, err)
			return errors.New(msg)
		}

		// copy file
		err = fs.copyFileDocument(&_filedoc, _to_doc_parent_path, _from_doc_parent_path, _to_doc_name, c)
		if err != nil {
			txt := "file '%s' in directory '%s' couldn't be copied.\n%s"
			msg := fmt.Sprintf(txt, _filedoc.Header.Name, _filedoc.Header.Parent, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) Info(p, db string) (interface{}, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return nil, errors.New(msg)
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
		q = fs.findChildrenQuery(p, ".")
		_count, e := c.Find(q).Count()
		if e != nil {
			msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, e)
			return nil, errors.New(msg)
		}
		_info["type"] = _type
		_info["content_count"] = _count

	} else {
		_type := "file"
		_info["type"] = _type
		if ri.AHeader.Filepointer != "" {
			_attch := bson.M{
				"mime": ri.AHeader.Mime,
				"size": ri.AHeader.Size,
			}
			_info["attachment"] = _attch
		}
	}

	return _info, nil
}

func (fs *BFS) ChangeAccess(p, db string, protect bool) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		// update directory access by updating field
		q = bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("directory '%s' access couldn't be updated.\n%s", p, err)
			return errors.New(msg)
		}
		// automatically cascade to sub nodes
		q = fs.findAllChildrenQuery(p)
		uq := bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		_, e := c.UpdateAll(q, uq)
		if e != nil {
			msg := fmt.Sprintf("directory '%s' access couldn't be updated.\n%s", p, e)
			return errors.New(msg)
		}

	} else {
		// update file access by updating field
		q = bson.M{"$set": bson.M{"__header__.ispublic": !protect}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("file '%s' access couldn't be updated.\n%s", p, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) Counter(c, a string, v int64, db string) (interface{}, error) {
	// update value 'v'
	nv := math.Abs(float64(v))
	v = int64(nv)

	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}

	// get collection
	col := fs.mongo.DB(db).C(fs.config.Bfs.CounterCol)

	// check if counter exists
	q := bson.M{"name": c}
	num, err := col.Find(q).Count()
	if err != nil {
		return nil, err
	}
	// if not exists create new counter
	if num < 1 {
		err = fs.validateCounterName(c)
		if err != nil {
			return nil, err
		}

		doc := bson.M{"name": c, "value": v}
		err = col.Insert(doc)
		if err != nil {
			return nil, err
		}

		return v, nil
	}

	var cq mgo.Change
	switch a {
	case "incr":
		cq = mgo.Change{
			Update:    bson.M{"$inc": bson.M{"value": v}},
			ReturnNew: true,
		}
		break
	case "decr":
		cq = mgo.Change{
			Update:    bson.M{"$inc": bson.M{"value": -v}},
			ReturnNew: true,
		}
		break
	case "reset":
		cq = mgo.Change{
			Update:    bson.M{"$set": bson.M{"value": v}},
			ReturnNew: true,
		}
		break
	default: // shouldn't reach here
		return nil, errors.New("Invalid counter action.")
	}

	var r interface{}
	_, err = col.Find(q).Apply(cq, &r)
	if err != nil {
		return nil, err
	}

	return r.(bson.M)["value"], nil
}

type counterItem struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

func (fs *BFS) CounterList(rgx, db string) (interface{}, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}

	// get collection
	col := fs.mongo.DB(db).C(fs.config.Bfs.CounterCol)

	list := []counterItem{}
	qre := bson.RegEx{Pattern: rgx, Options: "i"} // case insensitive regex
	q := bson.M{"name": bson.M{"$regex": qre}}
	iter := col.Find(q).Iter()
	var ci counterItem
	for iter.Next(&ci) {
		list = append(list, ci)
	}
	err = iter.Close()
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (fs *BFS) AddAttachment(fp, ap, fn, db string) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	fp = path.Clean(fp)
	_, err = os.Stat(ap)
	if err != nil {
		return errors.New("Attachment not found.")
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(fp)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", fp, err)
		return errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		return errors.New("command only valid for files.")

	} else {
		// create attachment path
		nap := path.Join(fs.config.Bfs.AttachmentsRoot, db, ri.Id)
		_, err = exec.Command("mv", ap, nap).Output()
		if err != nil {
			return err
		}

		// try and get uploaded file mime type
		tmpfile, err := os.Open(nap)
		if err != nil {
			return err
		}
		defer tmpfile.Close()

		// read 1024 bytes to enable mime type retrieval
		mimebuffer := make([]byte, 1024)
		_, err = tmpfile.Read(mimebuffer)
		if err != nil {
			return err
		}
		mime := http.DetectContentType(mimebuffer)

		// if mime is 'text/plain' try and get exact mime from file extension
		mimelist := map[string]string{
			".js":  "text/javascript",
			".css": "text/css",
		}
		prefix := "text/plain;"
		if strings.HasPrefix(mime, prefix) {
			ext := path.Ext(fn)
			mval, exists := mimelist[ext]
			if exists {
				mime = strings.Replace(mime, prefix, mval, 1)
			}
		}

		// get total file size
		f_info, _ := tmpfile.Stat()
		size := f_info.Size()

		// update file access by updating field
		q = bson.M{
			"$set": bson.M{
				"__attch__.filepointer": nap,
				"__attch__.mime":        mime,
				"__attch__.size":        size,
			}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("attachment for file '%s' couldn't be added.\n%s", fp, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) GetAttachment(fp, db string) (AttachmentHeader, error) {
	var header AttachmentHeader
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return header, err
	}

	// check path
	fp = path.Clean(fp)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(fp)
	var ri File
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", fp, err)
		return header, errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		return header, errors.New("command only valid for files.")

	}
	header = ri.AHeader
	return header, nil
}

func (fs *BFS) NewUploadRequestTicket(p, db string) (string, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return "", err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return "", errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		return "", errors.New("command only valid for files.")

	}

	// create upload key
	_uploadkey, err := makeUUID()
	if err != nil {
		msg := fmt.Sprintf("upload ticket couldn't be created.\n%s", err)
		return "", errors.New(msg)
	}

	_timeout := fs.config.Bfs.UploadTimeout
	rclient := fs.redisMan.DbConnect["upload"]

	// create entry and set timeout
	// upload temp file path creation
	up_f := path.Join(fs.config.Web.UploadDirectory, db+"_"+_uploadkey)

	// [0] database; [1] content path; [2] tmp file path
	_, e := rclient.Pipelined(func(c *redis.PipelineClient) {
		c.RPush(_uploadkey, db)
		c.RPush(_uploadkey, p)
		c.RPush(_uploadkey, up_f)
		c.Expire(_uploadkey, int64(_timeout))
	})
	if e != nil {
		msg := fmt.Sprintf("upload ticket couldn't be created.\n%s", e)
		return "", errors.New(msg)
	}

	return _uploadkey, nil
}

func (fs *BFS) UploadRequestTicketInfo(t string) ([]string, error) {
	// redis connection
	rclient := fs.redisMan.DbConnect["upload"]

	// get info
	// [0] database; [1] content path; [2] tmp file path
	_data := rclient.LRange(t, 0, 2)
	e := _data.Err()
	if e != nil {
		msg := fmt.Sprintf("upload ticket couldn't be retrieved.", e)
		return []string{}, errors.New(msg)
	}
	if len(_data.Val()) < 3 {
		msg := fmt.Sprintf("upload ticket couldn't be retrieved.", e)
		return []string{}, errors.New(msg)
	}

	return _data.Val(), nil
}

func (fs *BFS) RemoveAttachment(p, db string) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri SimpleResultItem
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve info for '%s'.\n%s", p, err)
		return errors.New(msg)
	}

	if ri.Header.Type == "Directory" {
		return errors.New("command only valid for files.")

	} else {
		// delete attachment
		if ri.AHeader.Filepointer != "" {
			// delete attachment
			err = os.Remove(ri.AHeader.Filepointer)
			if err != nil && os.IsExist(err) {
				msg := fmt.Sprintf("attachment for file '%s' couldn't be deleted.\n%s", p, err)
				return errors.New(msg)
			}
		}
		// update file access by updating field
		q = bson.M{"$set": bson.M{"__attch__.filepointer": "", "__attch__.mime": "", "__attch__.size": 0}}
		err = c.UpdateId(ri.Id, q)
		if err != nil {
			msg := fmt.Sprintf("attachment for file '%s' couldn't be deleted.\n%s", p, err)
			return errors.New(msg)
		}
	}

	return nil
}

func (fs *BFS) OverwriteFileContent(p, db string, j map[string]interface{}) error {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file if it exists
	q := fs.findPathQuery(p)
	q["__header__.type"] = "File"
	uq := bson.M{"$set": bson.M{"content": j}}
	// update file
	err = c.Update(q, uq)
	if err != nil {
		msg := fmt.Sprintf("file '%s' content couldn't be updated.\n%s", p, err)
		return errors.New(msg)
	}

	return nil
}

func (fs *BFS) DirectAccess(p, t, db string) (interface{}, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}

	// check path
	p = path.Clean(p)

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// get file or directory if it exists
	q := fs.findPathQuery(p)
	var ri File
	err = c.Find(q).One(&ri)
	if err != nil {
		msg := fmt.Sprintf("couldn't retrieve content at '%s'.\n%s", p, err)
		return nil, errors.New(msg)
	}

	// check if content can be served
	if !ri.Header.IsPublic {
		return nil, errors.New("content isn't available for public access.")
	}

	if ri.Header.Type != "File" {
		return nil, errors.New("action only valid for files.")
	}

	// determine which part of the file to serve: data or binary
	switch t {
	case "fd":
		var content interface{}
		err = c.FindId(ri.Id).Select(bson.M{"content": 1}).One(&content)
		if err != nil {
			msg := fmt.Sprintf("couldn't retrieve content at '%s'.\n%s", p, err)
			return nil, errors.New(msg)
		}
		return content.(bson.M)["content"], nil
	case "fa":
		return ri.AHeader, nil
	default:
		break
	}

	return nil, errors.New("invlid data request type: " + t)
}

func (fs *BFS) BQLSearch(db string, q bson.M) (interface{}, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}
	// check fields and paths
	fields, hasfields := q["fields"].([]string)
	paths, haspaths := q["dirs"].([]string)
	where, haswhere := q["where"].(map[string]interface{})
	limit, haslimit := q["limit"].(int64)
	sort, hassort := q["sort"].([]string)
	_, hascount := q["count"]
	distinct, hasdistinct := q["distinct"].(string)

	if !hasfields && !haspaths {
		return nil, errors.New("Invalid select query: No fields or document paths.")
	}

	// build query
	query := bson.M{
		"__header__.parent": bson.M{"$in": paths},
		"__header__.type":   "File"} // make sure return item is file
	if haswhere {
		_and := where["and"].([]map[string]interface{})
		if len(_and) > 0 {
			query["$and"] = _and
		}
		_or := where["or"].([]map[string]interface{})
		if len(_or) > 0 {
			query["$or"] = _or
		}
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// run query
	tmp := c.Find(query)
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
		err = tmp.Distinct(distinct, &distinctlist)
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

func (fs *BFS) BQLSet(db string, q bson.M) (interface{}, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}
	// check fields and paths
	fields, hasfields := q["fields"].(map[string]interface{})
	incr_fields, hasincr := q["incr"].(map[string]interface{})
	paths, haspaths := q["dirs"].([]string)
	where, haswhere := q["where"].(map[string]interface{})

	if !hasfields && !haspaths {
		return nil, errors.New("Invalid set command: No fields or document paths.")
	}

	// build query
	query := bson.M{"__header__.parent": bson.M{"$in": paths}}
	if haswhere {
		_and := where["and"].([]map[string]interface{})
		if len(_and) > 0 {
			query["$and"] = _and
		}
		_or := where["or"].([]map[string]interface{})
		if len(_or) > 0 {
			query["$or"] = _or
		}
	}
	// build update query
	uquery := bson.M{"$set": fields}
	if hasincr {
		uquery["$inc"] = incr_fields
	}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// run query
	info, err := c.UpdateAll(query, uquery)
	if err != nil {
		return nil, err
	}

	return info.Updated, nil
}

func (fs *BFS) BQLUnset(db string, q bson.M) (interface{}, error) {
	// check database access
	err := fs.checkDbAccess(db)
	if err != nil {
		return nil, err
	}
	// check fields and paths
	fields, hasfields := q["fields"].(map[string]interface{})
	paths, haspaths := q["dirs"].([]string)
	where, haswhere := q["where"].(map[string]interface{})

	if !hasfields && !haspaths {
		return nil, errors.New("Invalid unset command: No fields or document paths.")
	}

	// build query
	query := bson.M{"__header__.parent": bson.M{"$in": paths}}
	if haswhere {
		_and := where["and"].([]map[string]interface{})
		if len(_and) > 0 {
			query["$and"] = _and
		}
		_or := where["or"].([]map[string]interface{})
		if len(_or) > 0 {
			query["$or"] = _or
		}
	}
	// build update query
	uquery := bson.M{"$unset": fields}

	// get collection
	c := fs.mongo.DB(db).C(fs.config.Bfs.ContentCol)

	// run query
	info, err := c.UpdateAll(query, uquery)
	if err != nil {
		return nil, err
	}

	return info.Updated, nil
}

func (fs *BFS) findPathQuery(p string) bson.M {
	// build query
	var q bson.M
	if p == "/" {
		q = bson.M{"__header__.parent": "", "__header__.name": "/"}
	} else {
		q = bson.M{"__header__.parent": path.Dir(p), "__header__.name": path.Base(p)}
	}
	return q
}

func (fs *BFS) findChildrenQuery(p, rgx string) bson.M {
	qre := bson.RegEx{Pattern: rgx, Options: "i"} // case insensitive regex
	q := bson.M{
		"__header__.parent": p,
		"__header__.name":   bson.M{"$regex": qre},
	}
	return q
}

func (fs *BFS) findAllChildrenQuery(p string) bson.M {
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

func (fs *BFS) checkDbAccess(db string) error {
	// check if database exists and not reserved
	if fs.IsReservedDb(db) {
		msg := fmt.Sprintf("connot perform action on system database.")
		return errors.New(msg)
	}
	dbs, err := fs.ListDatabases(".")
	if err != nil {
		return err
	}
	_db_exists := false
	for _, item := range dbs {
		if db == item {
			_db_exists = true
			break
		}
	}
	if !_db_exists {
		msg := fmt.Sprintf("database '%s' doesn't exist.", db)
		return errors.New(msg)
	}

	return nil
}

func (fs *BFS) validateDirName(d string) error {
	msg := fmt.Sprintf("directory name '%s' isn't valid.", d)
	r, err := regexp.Compile("^[a-zA-Z0-9][a-zA-Z0-9_\\-]{0,}$")
	if err != nil {
		return errors.New(msg)
	}
	if r.MatchString(d) {
		return nil
	}
	return errors.New(msg)
}

func (fs *BFS) validateCounterName(c string) error {
	msg := fmt.Sprintf("counter name '%s' isn't valid.", c)
	r, err := regexp.Compile("[a-zA-Z0-9_\\.\\-]+")
	if err != nil {
		return errors.New(msg)
	}
	match := r.FindString(c)
	if match != c {
		return errors.New(msg)
	}
	return nil
}

func (fs *BFS) validateFileName(f string) error {
	msg := fmt.Sprintf("file name '%s' isn't valid.", f)
	r, err := regexp.Compile("^\\w[\\w\\-]{0,}(\\.[a-zA-Z0-9]+)*$")
	if err != nil {
		return errors.New(msg)
	}
	if r.MatchString(f) {
		return nil
	}
	return errors.New(msg)
}
