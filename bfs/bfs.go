package bfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/johnwilson/bytengine/bst"
	"github.com/nu7hatch/gouuid"
	"regexp"
	"strings"
	"time"
)

type BFSResponse struct {
	content map[string]interface{}
}

func (r BFSResponse) Success() bool {
	status, ok := r.content["status"].(string)
	if !ok || status != "ok" {
		return false
	}
	return true
}

func (r BFSResponse) ErrorMessage() string {
	if !r.Success() {
		return r.content["msg"].(string)
	}
	return ""
}

func (r BFSResponse) Data() interface{} {
	data, ok := r.content["data"]
	if !ok {
		return nil
	}
	return data
}

func (r BFSResponse) JSON() []byte {
	b, err := json.Marshal(r.content)
	if err != nil {
		return []byte{}
	}
	return b
}

func (r BFSResponse) String() string {
	return string(r.JSON())
}

type BFS interface {
	Start(config string, b *bst.ByteStore) error
	ClearAll() BFSResponse
	ListDatabase(filter string) BFSResponse
	CreateDatabase(db string) BFSResponse
	DropDatabase(db string) BFSResponse
	NewDir(p, db string) BFSResponse
	NewFile(p, db string, jsondata map[string]interface{}) BFSResponse
	ListDir(p, filter, db string) BFSResponse
	ReadJson(p, db string, fields []string) BFSResponse
	Delete(p, db string) BFSResponse
	Rename(p, newname, db string) BFSResponse
	Move(from, to, db string) BFSResponse
	Copy(from, to, db string) BFSResponse
	Info(p, db string) BFSResponse
	FileAccess(p, db string, protect bool) BFSResponse
	SetCounter(counter, action string, value int64, db string) BFSResponse
	ListCounter(filter, db string) BFSResponse
	WriteBytes(p, ap, db string) BFSResponse
	ReadBytes(fp, db string) BFSResponse
	DirectAccess(fp, db, layer string) BFSResponse
	DeleteBytes(p, db string) BFSResponse
	UpdateJson(p, db string, j map[string]interface{}) BFSResponse
	BQLSearch(db string, query map[string]interface{}) BFSResponse
	BQLSet(db string, query map[string]interface{}) BFSResponse
	BQLUnset(db string, query map[string]interface{}) BFSResponse
}

func ValidateDbName(d string) error {
	d = strings.ToLower(d)

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

func ValidateDirName(d string) error {
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

func ValidateCounterName(c string) error {
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

func ValidateFileName(f string) error {
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

func FormatDatetime(t time.Time) string {
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

func NewNodeID() (string, error) {
	tmp, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	id := strings.Replace(tmp.String(), "-", "", -1) // remove dashes
	return id, nil
}

func ErrorResponse(err error) BFSResponse {
	content := map[string]interface{}{
		"status": "error",
		"msg":    err.Error(),
	}
	return BFSResponse{content}
}

func OKResponse(d interface{}) BFSResponse {
	content := map[string]interface{}{
		"status": "ok",
		"data":   d,
	}
	return BFSResponse{content}
}

var plugins = make(map[string]BFS)

func Register(name string, plugin BFS) {
	if plugin == nil {
		panic("BFS Plugin Registration: plugin is nil")
	}
	if _, exists := plugins[name]; exists {
		panic("BFS Plugin Registration: plugin '" + name + "' already registered")
	}
	plugins[name] = plugin
}

func NewPlugin(pluginName, config string, b *bst.ByteStore) (plugin BFS, err error) {
	plugin, ok := plugins[pluginName]
	if !ok {
		err = fmt.Errorf("BFS Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config, b)
	if err != nil {
		plugin = nil
	}
	return
}
