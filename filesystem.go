package bytengine

import (
	"encoding/json"
	"fmt"
	"log"
)

type Status string

const (
	OK    Status = "ok"
	ERROR Status = "error"
)

var bfsPlugins = make(map[string]FileSystem)

type Response struct {
	Status        Status
	StatusMessage string
	Data          interface{}
}

func (r Response) JSON() []byte {
	val := r.Map()
	b, err := json.Marshal(val)
	if err != nil {
		return []byte{}
	}
	return b
}

func (r Response) Map() map[string]interface{} {
	var val map[string]interface{}

	if r.Status == OK {
		val = map[string]interface{}{
			"status": r.Status,
			"data":   r.Data,
		}
	} else {
		val = map[string]interface{}{
			"status": r.Status,
			"msg":    r.StatusMessage,
		}
	}

	return val
}

func (r Response) String() string {
	return string(r.JSON())
}

type FileSystem interface {
	Start(config string, b *ByteStore) error
	ClearAll() (Response, error)
	ListDatabase(filter string) (Response, error)
	CreateDatabase(db string) (Response, error)
	DropDatabase(db string) (Response, error)
	NewDir(p, db string) (Response, error)
	NewFile(p, db string, jsondata map[string]interface{}) (Response, error)
	ListDir(p, filter, db string) (Response, error)
	ReadJson(p, db string, fields []string) (Response, error)
	Delete(p, db string) (Response, error)
	Rename(p, newname, db string) (Response, error)
	Move(from, to, db string) (Response, error)
	Copy(from, to, db string) (Response, error)
	Info(p, db string) (Response, error)
	FileAccess(p, db string, protect bool) (Response, error)
	SetCounter(counter, action string, value int64, db string) (Response, error)
	ListCounter(filter, db string) (Response, error)
	WriteBytes(p, ap, db string) (Response, error)
	ReadBytes(fp, db string) (Response, error)
	DirectAccess(fp, db, layer string) (Response, error)
	DeleteBytes(p, db string) (Response, error)
	UpdateJson(p, db string, j map[string]interface{}) (Response, error)
	BQLSearch(db string, query map[string]interface{}) (Response, error)
	BQLSet(db string, query map[string]interface{}) (Response, error)
	BQLUnset(db string, query map[string]interface{}) (Response, error)
}

func ErrorResponse(err error) Response {
	return Response{
		Status:        ERROR,
		StatusMessage: err.Error(),
		Data:          nil,
	}
}

func OKResponse(d interface{}) Response {
	return Response{
		Status:        OK,
		StatusMessage: "",
		Data:          d,
	}
}

func RegisterFileSystem(name string, plugin FileSystem) {
	if plugin == nil {
		log.Fatal("File System Plugin Registration: plugin is nil")
	}

	if _, exists := bfsPlugins[name]; exists {
		log.Printf("File System Plugin Registration: plugin '%s' already registered", name)
	}
	bfsPlugins[name] = plugin
}

func NewFileSystem(pluginName, config string, b *ByteStore) (plugin FileSystem, err error) {
	plugin, ok := bfsPlugins[pluginName]
	if !ok {
		err = fmt.Errorf("File System Plugin Creation: unknown plugin name %q (forgot to import?)", pluginName)
		return
	}
	err = plugin.Start(config, b)
	if err != nil {
		plugin = nil
	}
	return
}
