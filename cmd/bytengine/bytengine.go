package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/codegangsta/cli"
	"github.com/gin-gonic/gin"
	"github.com/johnwilson/bytengine"
)

var (
	EngineRequestChan chan *EngineRequest
	Configuration     *Config
)

const (
	Version = "0.2.3"
)

type EngineRequest struct {
	Token        string
	Script       string
	Command      *bytengine.Command
	ResponseChan chan EngineResponse
}

type EngineResponse struct {
	Response interface{}
	Error    error
}

type Config struct {
	Bytengine json.RawMessage
	Workers   int
	Port      int
	Address   string
	Timeout   ConfigTimeout
}

type ConfigTimeout struct {
	AuthToken    int64
	UploadTicket int64
}

func errorResponse(err error) []byte {
	val := map[string]string{
		"status": "error",
		"msg":    err.Error(),
	}

	b, e := json.Marshal(val)
	if e != nil {
		return []byte{}
	}
	return b
}

func okResponse(data interface{}) []byte {
	val := map[string]interface{}{
		"status": "ok",
		"data":   data,
	}

	b, e := json.Marshal(val)
	if e != nil {
		return []byte{}
	}
	return b
}

func Worker(config *Config, requests chan *EngineRequest) {
	// create engine and start
	engine := bytengine.NewEngine()
	err := engine.Start(config.Bytengine)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	for req := range requests {
		// check if script request or command request
		if req.Command != nil {
			r, err := engine.ExecuteCommand(req.Token, *req.Command)
			rep := EngineResponse{r, err}
			req.ResponseChan <- rep
		} else {
			r, err := engine.ExecuteScript(req.Token, req.Script)
			rep := EngineResponse{r, err}
			req.ResponseChan <- rep
		}
	}
}

func WorkerPool(workers int, config *Config) chan *EngineRequest {
	requests := make(chan *EngineRequest)
	for i := 0; i < workers; i++ {
		go Worker(config, requests)
	}

	return requests
}

func welcomeHandler(ctx *gin.Context) {
	msg := fmt.Sprintf(`{"bytengine":"welcome","version":"%s"}`, Version)
	ctx.Data(200, "application/json", []byte(msg))
}

func runScriptHandler(ctx *gin.Context) {
	var form struct {
		Token string `form:"token" binding:"required"`
		Query string `form:"query" binding:"required"`
	}

	ok := ctx.Bind(&form)
	if !ok {
		data := errorResponse(errors.New("Missing parameters"))
		ctx.Data(400, "application/json", data)
		return
	}

	req := EngineRequest{
		Token:        form.Token,
		Script:       form.Query,
		ResponseChan: make(chan EngineResponse),
	}
	EngineRequestChan <- &req
	rep := <-req.ResponseChan
	if rep.Error != nil {
		data := errorResponse(rep.Error)
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", okResponse(rep.Response))
}

func getTokenHandler(ctx *gin.Context) {
	var form struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}
	ok := ctx.Bind(&form)
	if !ok {
		data := errorResponse(errors.New("Missing parameters"))
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := bytengine.Command{
		Name:    "login",
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = form.Username
	cmd.Args["password"] = form.Password

	duration := Configuration.Timeout.AuthToken // in minutes
	cmd.Args["duration"] = duration

	req := EngineRequest{
		Token:        "",
		Command:      &cmd,
		ResponseChan: make(chan EngineResponse),
	}
	EngineRequestChan <- &req
	rep := <-req.ResponseChan
	if rep.Error != nil {
		data := errorResponse(rep.Error)
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", okResponse(rep.Response))
}

func getUploadTicketHandler(ctx *gin.Context) {
	var form struct {
		Token    string `form:"token" binding:"required"`
		Database string `form:"database" binding:"required"`
		Path     string `form:"path" binding:"required"`
	}
	ok := ctx.Bind(&form)
	if !ok {
		data := errorResponse(errors.New("Missing parameters"))
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := bytengine.Command{
		Name:    "uploadticket",
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = form.Database
	cmd.Args["path"] = form.Path

	duration := Configuration.Timeout.UploadTicket // in minutes
	cmd.Args["duration"] = duration

	req := EngineRequest{
		Token:        form.Token,
		Command:      &cmd,
		ResponseChan: make(chan EngineResponse),
	}
	EngineRequestChan <- &req
	rep := <-req.ResponseChan
	if rep.Error != nil {
		data := errorResponse(rep.Error)
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", okResponse(rep.Response))
}

func uploadFileHelper(max int, ctx *gin.Context) (string, int, error) {
	total := 0             // total bytes read/written
	maxbytes := 1024 * max // maximum upload size in bytes

	tmpfile, err := ioutil.TempFile("", "bytengine_upload_") // upload file name from header
	if err != nil {
		return "", 0, err
	}
	defer tmpfile.Close()

	// create read buffer
	var bsize int64 = 16 * 1024 // 16 kb
	buffer := make([]byte, bsize)

	// get stream
	mr, err := ctx.Request.MultipartReader()
	if err != nil {
		return "", 0, err
	}
	in_f, err := mr.NextPart()
	if err != nil {
		return "", 0, err
	}
	defer in_f.Close()

	// start reading/writing

	for {
		// read
		n, err := in_f.Read(buffer)
		if n == 0 {
			break
		}
		if err != nil {
			return "", 0, err
		}
		// update total bytes
		total += n
		if total > maxbytes {
			return "", 0, fmt.Errorf("exceeded maximum file size of %d bytes", maxbytes)
		}
		// write
		n, err = tmpfile.Write(buffer[:n])
		if err != nil {
			return "", 0, err
		}
	}

	return tmpfile.Name(), total, nil
}

func uploadFileHandler(ctx *gin.Context) {
	ticket := ctx.Params.ByName("ticket")
	filename, _, err := uploadFileHelper(300, ctx)
	if err != nil {
		data := errorResponse(fmt.Errorf("upload failed: %s", err.Error()))
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := bytengine.Command{
		Name:    "writebytes",
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["ticket"] = ticket
	cmd.Args["tmpfile"] = filename

	req := EngineRequest{
		Token:        "",
		Command:      &cmd,
		ResponseChan: make(chan EngineResponse),
	}
	EngineRequestChan <- &req
	rep := <-req.ResponseChan
	if rep.Error != nil {
		data := errorResponse(rep.Error)
		ctx.Data(400, "application/json", data)
		return
	}

	ctx.Data(200, "application/json", okResponse(rep.Response))
}

func downloadFileHandler(ctx *gin.Context) {
	var form struct {
		Token    string `form:"token" binding:"required"`
		Database string `form:"database" binding:"required"`
		Path     string `form:"path" binding:"required"`
	}
	ok := ctx.Bind(&form)
	if !ok {
		data := errorResponse(errors.New("Missing parameters"))
		ctx.Data(400, "application/json", data)
		return
	}

	cmd := bytengine.Command{
		Name:    "readbytes",
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = form.Database
	cmd.Args["path"] = form.Path
	cmd.Args["writer"] = ctx.Writer

	ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
	req := EngineRequest{
		Token:        form.Token,
		Command:      &cmd,
		ResponseChan: make(chan EngineResponse),
	}
	EngineRequestChan <- &req
	rep := <-req.ResponseChan
	if rep.Error != nil {
		data := errorResponse(rep.Error)
		ctx.String(400, string(data))
		return
	}
}

func directaccessHandler(ctx *gin.Context) {
	db := ctx.Params.ByName("database")
	path := ctx.Params.ByName("path")
	layer := ctx.Params.ByName("layer")

	cmd := bytengine.Command{
		Name:    "directaccess",
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["database"] = db
	cmd.Args["path"] = path
	cmd.Args["layer"] = layer
	cmd.Args["writer"] = ctx.Writer

	if layer == "json" {
		ctx.Writer.Header().Set("Content-Type", "application/json")
	} else {
		ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
	}

	req := EngineRequest{
		Token:        "",
		Command:      &cmd,
		ResponseChan: make(chan EngineResponse),
	}
	EngineRequestChan <- &req
	rep := <-req.ResponseChan
	if rep.Error != nil {
		data := errorResponse(rep.Error)
		ctx.String(404, string(data))
		return
	}
}

func readConfigFile(pth string) error {
	b, err := ioutil.ReadFile(pth)
	if err != nil {
		return err
	}

	Configuration = &Config{}
	return json.Unmarshal(b, Configuration)
}

func main() {
	app := cli.NewApp()

	createadminCmd := cli.Command{
		Name: "createadmin",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "u", Value: "", Usage: "username"},
			cli.StringFlag{Name: "p", Value: "", Usage: "password"},
			cli.StringFlag{Name: "c", Value: "config.json"},
		},
		Action: func(c *cli.Context) {
			usr := c.String("u")
			pw := c.String("p")
			pth := c.String("c")

			err := readConfigFile(pth)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}

			// create and start bytengine
			engine := bytengine.NewEngine()
			err = engine.Start(Configuration.Bytengine)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}

			err = engine.CreateAdminUser(usr, pw)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}
			fmt.Println("...done")
		},
	}

	run := cli.Command{
		Name: "run",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "c", Value: "config.json"},
		},
		Action: func(c *cli.Context) {
			// get configuration file/info
			pth := c.String("c")
			err := readConfigFile(pth)
			if err != nil {
				fmt.Println("Error: ", err)
				os.Exit(1)
			}

			addr := Configuration.Address
			port := Configuration.Port
			workers := Configuration.Workers

			// start worker pool
			EngineRequestChan = WorkerPool(workers, Configuration)

			// setup routes
			router := gin.Default()
			router.GET("/", welcomeHandler)
			router.POST("/bfs/query", runScriptHandler)
			router.POST("/bfs/token", getTokenHandler)
			router.POST("/bfs/uploadticket", getUploadTicketHandler)
			router.POST("/bfs/writebytes/:ticket", uploadFileHandler)
			router.POST("/bfs/readbytes", downloadFileHandler)
			router.GET("/bfs/direct/:layer/:database/*path", directaccessHandler)

			router.Run(fmt.Sprintf("%s:%d", addr, port))
		},
	}
	app.Commands = []cli.Command{createadminCmd, run}
	app.Run(os.Args)
}
