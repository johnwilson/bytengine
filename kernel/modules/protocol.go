package modules

import (
	"encoding/json"
	"net/http"
	"fmt"

	"github.com/gorilla/mux"
)

// ------------ Request ------------ //

type Request interface {
	OnSuccess(interface{}, string)
	OnFailure(string, RequestError)
	GetChannels() (chan [][]byte, chan [][]byte)
}

// Generic Handler to be used by requests
type RequestHandler struct {
	success chan [][]byte
	failure chan [][]byte
}

func (h RequestHandler) OnSuccess(data interface{}, mime string) {
	b := SuccessResponse(data,mime)
	h.success <- b
}

func (h RequestHandler) OnFailure(msg string, err RequestError) {
	b := ErrorResponse(msg, err)
	h.failure <- b
}

func (h RequestHandler) GetChannels() (success, failure chan [][]byte) {
	return h.success, h.failure
}

type LoginRequest struct {
	Username string
	Password string
	RequestHandler	
}

type CommandRequest struct {
	SessionId string
	Script string
	RequestHandler
}

type InfoRequest struct {
	Name string
	RequestHandler
}

type UploadTicketRequest struct {
	Database string
	Path string
	RequestHandler
	SessionId string
}

type DownloadRequest struct {
	Database string
	Path string
	RequestHandler
	SessionId string
}

// upload ticket check
type UploadTCheckRequest struct {
	Ticket string
	RequestHandler
}

func NewUploadTCheckRequest(ticket string) UploadTCheckRequest {
	r := UploadTCheckRequest{ Ticket: ticket }
	r.success = make(chan [][]byte, 1)
	r.failure = make(chan [][]byte, 1)
	return r
}

// upload complete request
type UploadCompleteRequest struct {
	Database string
	UploadFile string
	ContentPath string
	HFileName string // header file name
	Size int
	RequestHandler
}

func NewUploadCompleteRequest(db, upfile, bfsfile, fname string, size int) UploadCompleteRequest {
	r := UploadCompleteRequest{
		Database: db,
		UploadFile: upfile,
		ContentPath: bfsfile,
		HFileName: fname,
		Size: size,
	}
	r.success = make(chan [][]byte, 1)
	r.failure = make(chan [][]byte, 1)
	return r
}

// content delivery/direct access
type DirectAccessRequest struct {
	Database string
	ContentPath string
	AccessType string
	RequestHandler
}

func NewDirectAccessRequest(path, typ, db string) DirectAccessRequest {
	r := DirectAccessRequest{
		Database: db,
		AccessType: typ,
		ContentPath: path,
	}
	r.success = make(chan [][]byte, 1)
	r.failure = make(chan [][]byte, 1)
	return r
}

// ------------ Request Errors ------------ //

type RequestError int

const (
	ProtocolFormatError RequestError = iota
	AuthError
	ExpiredSessionError
	SyntaxError
	CommandError
	ScriptError	
	EngineError
	ItemNotFoundError
	AccessDeniedError
)

// Parse incomming http requests
func ParsePostRequest(r *http.Request) (Request, [][]byte) {
	var err [][]byte
	// get values
	val := mux.Vars(r)
	typ := val["type"]
	r.ParseForm()
	switch typ {
	case "login":
		lr := LoginRequest{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
		}
		lr.success = make(chan [][]byte, 1)
		lr.failure = make(chan [][]byte, 1)
		return lr, err
	case "run":
		cr := CommandRequest{
			SessionId: r.FormValue("ticket"),
			Script: r.FormValue("script"),
		}
		cr.success = make(chan [][]byte, 1)
		cr.failure = make(chan [][]byte, 1)
		return cr, err
	case "upload":
		ur := UploadTicketRequest{
			SessionId: r.FormValue("ticket"),
			Database: r.FormValue("db"),
			Path: r.FormValue("path"),
		}
		ur.success = make(chan [][]byte, 1)
		ur.failure = make(chan [][]byte, 1)
		return ur, err
	case "download":
		dr := DownloadRequest{
			SessionId: r.FormValue("ticket"),
			Database: r.FormValue("db"),
			Path: r.FormValue("path"),
		}
		dr.success = make(chan [][]byte, 1)
		dr.failure = make(chan [][]byte, 1)
		return dr, err
	}
	
	b := ErrorResponse("Invalid request: " + typ, ProtocolFormatError)
	return nil, b
}

func ParseGetRequest(r *http.Request) (Request, [][]byte) {
	var err [][]byte
	// get values
	val := mux.Vars(r)
	typ := val["type"]
	switch typ {
	case "info":
		ir := InfoRequest{
			Name: val["query"],
		}
		ir.success = make(chan [][]byte, 1)
		ir.failure = make(chan [][]byte, 1)
		return ir, err
	}
	
	b := ErrorResponse("Invalid request: " + typ, ProtocolFormatError)
	return nil, b
}

func SuccessResponse(data interface{}, mime string) [][]byte {
	mimeb := []byte(mime)
	var b []byte

	switch mime {
	case "text/plain":
		fallthrough
	case "text/html":
		b = []byte(data.(string))
		break
	default: // application/json
		reply := map[string]interface{}{"status":"ok","data":data}
		j, err := json.Marshal(reply)
		if err != nil {
			msg := fmt.Sprintf("{\"status\":\"error\",\"msg\":\"Error encoding response\",\"code\":\"%d\"}", EngineError)
			return [][]byte{ []byte("application/json"), []byte(msg) }
		}
		b = []byte(j)
	}
	
	return [][]byte{ mimeb, b }
}

// error response
func ErrorResponse(msg string, err RequestError) [][]byte {
	mime := []byte("application/json")
	var b []byte
	reply := map[string]interface{}{"status":"error", "msg":msg, "code":err}
	j, e := json.Marshal(reply)
	if e != nil {
		msg := fmt.Sprintf("{\"status\":\"error\",\"msg\":\"Error encoding response\",\"code\":\"%d\"}", EngineError)
		return [][]byte{ mime, []byte(msg) }
	}
	b = []byte(j)
	return [][]byte{ mime, b }
}