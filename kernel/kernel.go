package kernel

import (
	"log"
	"os"
	"path"
	"fmt"
	"strings"
	"net/http"
	"encoding/json"
	
	"github.com/johnwilson/bytengine/kernel/modules"
	
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

// Global vars
//var Eng *CommandEngine
var EngReqChan chan modules.Request

// web vars
var decoder = schema.NewDecoder()
var ROOT_DIR string

func renderView(view string) []byte {
	var b []byte

	// load file
	view_f := path.Join(ROOT_DIR,"templates",view)
	file, err := os.Open(view_f)
	if err != nil {
		return b
	}
	defer file.Close()

	// get file size
	stat, err := file.Stat()
	if err != nil {
		return b
	}

	// read file
	b = make([]byte, stat.Size())
	_, err = file.Read(b)
	if err != nil {
		return b
	}

	return b
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	b := renderView("index.html")
	w.Header().Set("Content-Type","text/html")
	w.Write(b)
}

func documentationHandler(w http.ResponseWriter, r *http.Request) {
	b := renderView("documentation.html")
	w.Header().Set("Content-Type","text/html")
	w.Write(b)
}

func commandsHandler(w http.ResponseWriter, r *http.Request) {
	b := renderView("commands.html")
	w.Header().Set("Content-Type","text/html")
	w.Write(b)
}

func staticFileHandler(w http.ResponseWriter, r *http.Request) {
	_prefix := "/static/"
	_path := path.Clean(r.URL.Path)
	if !strings.HasPrefix(_path, _prefix) {
		http.Error(w, "", http.StatusNotFound)
		return
	}	
	_path = path.Join(ROOT_DIR, _path)
	http.ServeFile(w, r, _path)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","text/plain")
	fmt.Fprintf(w, "pong!")
}

func readUploadData(outpath string, r *http.Request) (int, error) {
	total := 0 // total bytes
	maxbytes := 102400 // hard limit of 100kb just for test

	// create read buffer
	var bsize int64 = 16 * 1024 // 16 kb
	buffer := make([]byte, bsize)

	// get stream
	mr, err := r.MultipartReader()
	if err != nil {
		return total, err
	}
	in_f, err := mr.NextPart()
	if err != nil {
		return total, err
	}
	defer in_f.Close()

	// create output file
	out_f, err := os.OpenFile(outpath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return total, err
	}
	defer out_f.Close()

	// start reading/writing
	
	for {
		// read
		n, err := in_f.Read(buffer)
		if n == 0 { break }
		if err != nil {
			return total, err
		}
		// update total bytes
		total += n
		if total > maxbytes {
			return total, fmt.Errorf("exceeded maximum file size of %d bytes",maxbytes)
		}
		// write
		n, err = out_f.Write(buffer[:n])
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func writeDownloadData(filep, mime string, size int64, w http.ResponseWriter) {
	// create read buffer
	var bsize int64 = 16 * 1024 // 16 kb
	buffer := make([]byte, bsize)

	err_msg := "Error reading content. Contact admin."

	// open attachment
	out_f, err := os.Open(filep)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err_msg, http.StatusNotFound)
		return
	}
	defer out_f.Close()

	// set response headers
	w.Header().Set("Content-Type",mime)
	w.Header().Set("Content-Length",fmt.Sprintf("%d",int64(size)))

	// start reading/writing
	for {
		// read
		n, err := out_f.Read(buffer)
		if n == 0 { break }
		if err != nil {
			fmt.Println(err)
			http.Error(w, err_msg, http.StatusNotFound)
			return
		}
		// write
		n, err = w.Write(buffer[:n])
		if err != nil {
			fmt.Println(err)
			http.Error(w, err_msg, http.StatusNotFound)
			return
		}
	}
	// finish
	return
}

func uploadAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	// get ticket from url
	vars := mux.Vars(r)
	ticket := vars["ticket"]
	req := modules.NewUploadTCheckRequest(ticket)    
    success, failure := req.GetChannels()
    EngReqChan <- req
    for {
        select {
        case msg := <- success:        	
        	// deserialize ticket info
        	b := msg[1]
        	var tmp interface{}
        	err := json.Unmarshal(b,&tmp)
        	if err != nil {
        		b2 := modules.ErrorResponse("Upload failed: ticket not verified", modules.EngineError)
        		w.Header().Set("Content-Type",string(msg[0]))
                w.WriteHeader(http.StatusBadRequest)
        		w.Write(b2[1])
        		return
        	}
        	// upload data to temp file
        	ticket_info := tmp.(map[string]interface{})["data"].([]interface{})
        	db := ticket_info[0].(string) // database to save to
        	bpath := ticket_info[1].(string) // bfs content path to attach to
        	tpath := ticket_info[2].(string) // uploaded file path

        	total, err := readUploadData(tpath, r)
        	if err != nil {
        		b2 := modules.ErrorResponse("Upload failed: error read/write data", modules.EngineError)
        		w.Header().Set("Content-Type",string(msg[0]))
                w.WriteHeader(http.StatusBadRequest)
        		w.Write(b2[1])
        		return
        	}
        	// send upload complete request
        	req2 := modules.NewUploadCompleteRequest(db, tpath, bpath, total)
        	success2, failure2 := req2.GetChannels()
    		EngReqChan <- req2
    		for {
    			select {
    				case msg2 := <- success2:
    					w.Header().Set("Content-Type",string(msg2[0]))
            			w.Write(msg2[1])
            			return
    				case msg2 := <- failure2:
    					w.Header().Set("Content-Type",string(msg2[0]))
                        w.WriteHeader(http.StatusBadRequest) // error 400
			            w.Write(msg2[1])
			            return
    			}
    		}
            return
        case msg := <- failure:
        	w.Header().Set("Content-Type",string(msg[0]))
            w.WriteHeader(http.StatusBadRequest) // error 400
            w.Write(msg[1])
            return
        }
    }
}

func runCommandHandler(w http.ResponseWriter, r *http.Request) {
	req, b := modules.ParsePostRequest(r)
    if len(b) > 0 {
        w.Header().Set("Content-Type",string(b[0]))
    	w.Write(b[1])
        return
    }

    success, failure := req.GetChannels()
    EngReqChan <- req
    for {
        select {
        case msg := <- success:
        	switch req.(type) {
        	case modules.DownloadRequest:
        		var j map[string]interface{}
        		err := json.Unmarshal(msg[1], &j)
        		if err != nil {
                    w.Header().Set("Content-Type",string(msg[0]))
        			w.Write([]byte(""))
		            return
        		}
        		filep := j["data"].(map[string]interface{})["filepointer"].(string)
        		size := j["data"].(map[string]interface{})["size"].(float64)
        		mime := j["data"].(map[string]interface{})["mime"].(string)

        		writeDownloadData(filep, mime, int64(size), w)

				return
        	default:
        		w.Header().Set("Content-Type",string(msg[0]))
	            w.Write(msg[1])
	            return
        	}        	
        case msg := <- failure:
            w.Header().Set("Content-Type",string(msg[0]))
        	w.Write(msg[1])
            return
        }
    }
}

// GET requests
func getRequestHandler(w http.ResponseWriter, r *http.Request) {
	req, b := modules.ParseGetRequest(r)
    if len(b) > 0 {
        w.WriteHeader(http.StatusBadRequest) // error 400
        w.Write(b[1])
        return
    }

    success, failure := req.GetChannels()
    EngReqChan <- req
    for {
        select {
        case msg := <- success:
        	w.Header().Set("Content-Type",string(msg[0]))
            w.Write(msg[1])
            return
        case msg := <- failure:
            w.WriteHeader(http.StatusBadRequest) // error 400
            w.Write(msg[1])
            return
        }
    }
}

func contentDileveryHandler(w http.ResponseWriter, r *http.Request){
	// get values from url
	vars := mux.Vars(r)
	db := vars["database"]
	typ := vars["type"]
	path := "/" + vars["path"]
	
	req := modules.NewDirectAccessRequest(path, typ, db)
	success, failure := req.GetChannels()
    EngReqChan <- req
    for {
        select {
        case msg := <- success:
        	if typ == "fa" {
        		var j map[string]interface{}
        		err := json.Unmarshal(msg[1], &j)
        		if err != nil {
        			w.WriteHeader(http.StatusNotFound) // error 400
		            w.Write([]byte(""))
		            return
        		}
        		filep := j["data"].(map[string]interface{})["filepointer"].(string)
        		size := j["data"].(map[string]interface{})["size"].(float64)
        		mime := j["data"].(map[string]interface{})["mime"].(string)

        		writeDownloadData(filep, mime, int64(size), w)

				return
        	}
        	w.Header().Set("Content-Type",string(msg[0]))
            w.Write(msg[1])
            return
        case msg := <- failure:
        	w.WriteHeader(http.StatusBadRequest) // error 400
            w.Write(msg[1])
            return
        }
    }	
}

// Build url router
func buildRoutes() (*mux.Router, error) {
	// create router
	r := mux.NewRouter()
	
	// API urls
	r.HandleFunc("/bfs/prq/{type}", runCommandHandler).Methods("POST","HEAD")
	r.HandleFunc("/bfs/upload/{ticket}", uploadAttachmentHandler).Methods("POST","HEAD")
	r.HandleFunc("/cds/{type:fd|fa}/{database}/{path:[\\w\\.0-9/_\\-]+}", contentDileveryHandler).Methods("GET","HEAD")
	
	/*
		Info Url
		=================
		/info/version --> bytengine server version
	*/
	r.HandleFunc("/bfs/grq/{type}/{query:[\\w\\.0-9_@]*}", getRequestHandler).Methods("GET","HEAD")
	
	// Web UI
	r.HandleFunc("/", welcomeHandler).Methods("GET","HEAD")
	r.HandleFunc("/ping", pingHandler).Methods("GET","HEAD")
	r.HandleFunc("/docs", documentationHandler).Methods("GET","HEAD")
	r.HandleFunc("/commands", commandsHandler).Methods("GET","HEAD")

	// Static files	
	r.HandleFunc("/static/{path:[\\w\\.0-9/_\\-]+}",staticFileHandler).Methods("GET","HEAD")

	return r, nil
}

func Run(configfile string) {
	// start command engine
	EngReqChan = make(chan modules.Request)
	engine := modules.NewCommandEngine(configfile)
    go engine.Run(EngReqChan)

	// set root directory
	ROOT_DIR = path.Join(path.Dir(configfile),"..","core","web")

	// create router
	r, err := buildRoutes()
	if err != nil {
		log.Panic(err)
	}

	// create app handler
	http.Handle("/", r)	
	log.Fatal(http.ListenAndServe(engine.SysInit.WebUrl(), nil))
}