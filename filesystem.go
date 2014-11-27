package bytengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nu7hatch/gouuid"
)

type Status string

const (
	OK    Status = "ok"
	ERROR Status = "error"
)

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

type BFS interface {
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
