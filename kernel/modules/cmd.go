package modules

import (
	"fmt"
	"errors"
)

// ############## Command ############## //

type Command struct {
	Name string
	Database string
	Args map[string] interface{}
}

func NewCommand(name string) Command {
	cmd := Command {
		Name:name,
		Args:map[string]interface{} {},
	}
	return cmd
}

// ############## Job ############## //

type JobStatus int

const (
	JPending JobStatus = iota
	JParsing
	JProcessing
	JCompeted
	JFailed
)

type Job struct {
	CommandQueue *List
	Req Request
	Engine *CommandEngine
	Status JobStatus
	// add authentication, bfs modules
}

// ############## Runtime Functions ############## //

type RuntimeFunction struct {
	Name string // command name
	Modes []ReqMode // list allowed access modes
	Handler func(*Command, *SystemUser, *ReqMode, *CommandEngine) (interface{}, error) // function called
}

// ------------ whoami ------------- //

var f1 = &RuntimeFunction {
	Name: "whoami",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		return u.Username, nil
	},
}

// ------------ server.init ------------- //

var f2 = &RuntimeFunction {
	Name: "initserver",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		e := eng.Bfs.RebuildServer()
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.newdb ------------- //

var f3 = &RuntimeFunction {
	Name: "newdb",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		e := eng.Bfs.MakeDatabase(cmd.Args["db"].(string))
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.listdb ------------- //

var f4 = &RuntimeFunction {
	Name: "listdb",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		if *m == RootMode {
			_dbs, e := eng.Bfs.ListDatabases()
			if e != nil {
				return nil, e
			}
			return _dbs, nil
		}
		return u.Databases, nil
	},
}

// ------------ server.dropdb ------------- //

var f5 = &RuntimeFunction {
	Name: "dropdb",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		e := eng.Bfs.RemoveDatabase(cmd.Args["db"].(string))
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.newuser ------------- //

var f6 = &RuntimeFunction {
	Name: "newuser",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		un := cmd.Args["username"].(string)
		pw := cmd.Args["password"].(string)
		e := eng.Auth.NewUser(un,pw)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.listuser ------------- //

var f7 = &RuntimeFunction {
	Name: "listuser",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		_list, e := eng.Auth.ListUsers()
		if e != nil {
			return nil, e
		}
		return _list, nil
	},
}

// ------------ server.userinfo ------------- //

var f8 = &RuntimeFunction {
	Name: "userinfo",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		_info, e := eng.Auth.UserInfo(cmd.Args["username"].(string))
		if e != nil {
			return nil, e
		}
		return _info, nil
	},
}

// ------------ server.dropuser ------------- //

var f9 = &RuntimeFunction {
	Name: "dropuser",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		e := eng.Auth.RemoveUser(cmd.Args["username"].(string))
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.newpass ------------- //

var f10 = &RuntimeFunction {
	Name: "newpass",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		un := cmd.Args["username"].(string)
		pw := cmd.Args["password"].(string)
		e := eng.Auth.ChangeUserPassword(un,pw)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.sysaccess ------------- //

var f11 = &RuntimeFunction {
	Name: "sysaccess",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		un := cmd.Args["username"].(string)
		grt := cmd.Args["grant"].(bool)
		e := eng.Auth.ChangeUserStatus(un,grt)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ server.userdb ------------- //

var f12 = &RuntimeFunction {
	Name: "userdb",
	Modes: []ReqMode{RootMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		un := cmd.Args["username"].(string)
		grt := cmd.Args["grant"].(bool)
		db := cmd.Args["database"].(string)
		e := eng.Auth.ChangeUserDbAccess(un,db,grt)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.newdir ------------- //

var f13 = &RuntimeFunction {
	Name: "newdir",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		e := eng.Bfs.MakeDir(pth,db)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.newfile ------------- //

var f14 = &RuntimeFunction {
	Name: "newfile",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		data := cmd.Args["data"].(map[string]interface{})
		e := eng.Bfs.MakeFile(pth,db,data)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.listdir ------------- //

var f15 = &RuntimeFunction {
	Name: "listdir",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		_list, e := eng.Bfs.DirectoryListing(pth,db)
		if e != nil {
			return nil, e
		}
		return _list, nil
	},
}

// ------------ @db.rename ------------- //

var f16 = &RuntimeFunction {
	Name: "rename",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		nm := cmd.Args["name"].(string)
		db := cmd.Database
		e := eng.Bfs.Rename(pth,nm,db)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.move ------------- //

var f17 = &RuntimeFunction {
	Name: "move",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		to := cmd.Args["to"].(string)
		db := cmd.Database
		e := eng.Bfs.Move(pth,to,db)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.copy ------------- //

var f18 = &RuntimeFunction {
	Name: "copy",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		to := cmd.Args["to"].(string)
		db := cmd.Database
		e := eng.Bfs.Copy(pth,to,db)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.delete ------------- //

var f19 = &RuntimeFunction {
	Name: "delete",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		e := eng.Bfs.Delete(pth,db)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.info ------------- //

var f20 = &RuntimeFunction {
	Name: "info",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		_info, e := eng.Bfs.Info(pth,db)
		if e != nil {
			return nil, e
		}
		return _info, nil
	},
}

// ------------ @db.makepublic ------------- //

var f21 = &RuntimeFunction {
	Name: "makepublic",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		e := eng.Bfs.ChangeAccess(pth,db,false)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.makeprivate ------------- //

var f22 = &RuntimeFunction {
	Name: "makeprivate",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		e := eng.Bfs.ChangeAccess(pth,db,true)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.readfile ------------- //

var f23 = &RuntimeFunction {
	Name: "readfile",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		flds := cmd.Args["fields"].([]string)
		db := cmd.Database
		_content, e := eng.Bfs.GetFileContent(pth,db,flds)
		if e != nil {
			return nil, e
		}
		return _content, nil
	},
}

// ------------ @db.modfile ------------- //

var f24 = &RuntimeFunction {
	Name: "modfile",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		data := cmd.Args["data"].(map[string]interface{})
		db := cmd.Database
		e := eng.Bfs.OverwriteFileContent(pth,db,data)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.deletebinary ------------- //

var f25 = &RuntimeFunction {
	Name: "deletebinary",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		pth := cmd.Args["path"].(string)
		db := cmd.Database
		e := eng.Bfs.RemoveAttachment(pth,db)
		if e != nil {
			return nil, e
		}
		return 1, nil
	},
}

// ------------ @db.counter ------------- //

var f26 = &RuntimeFunction {
	Name: "counter",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		act := cmd.Args["action"].(string)
		db := cmd.Database
		if act != "list" {
			nm := cmd.Args["name"].(string)
			val := cmd.Args["value"].(int64)
			_data, e := eng.Bfs.Counter(nm,act,val,db)
			if e != nil {
				return nil, e
			}
			return _data, nil
		}

		_data, e := eng.Bfs.CounterList(db)
		if e != nil {
			return nil, e
		}
		return _data, nil
		
	},
}

// ------------ @db.select ------------- //

var f27 = &RuntimeFunction {
	Name: "select",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		db := cmd.Database
		_data, e := eng.Bfs.BQLSearch(db,cmd.Args)
		if e != nil {
			return nil, e
		}
		return _data, nil
	},
}

// ------------ @db.set ------------- //

var f28 = &RuntimeFunction {
	Name: "set",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		db := cmd.Database
		_data, e := eng.Bfs.BQLSet(db,cmd.Args)
		if e != nil {
			return nil, e
		}
		return _data, nil
	},
}

// ------------ @db.unset ------------- //

var f29 = &RuntimeFunction {
	Name: "unset",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		db := cmd.Database
		_data, e := eng.Bfs.BQLUnset(db,cmd.Args)
		if e != nil {
			return nil, e
		}
		return _data, nil
	},
}

// ------------ pipe >> ------------- //

var f30 = &RuntimeFunction {
	Name: "pipe",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		fname := cmd.Args["function"].(string)
		fres := cmd.Args["resultset"]
		fargs := cmd.Args["args"].(map[string]interface{})
		r, e := eng.fm.Pipe(fname,fres,fargs)
		return r, e
	},
}

// ------------ exec ------------- //

var f31 = &RuntimeFunction {
	Name: "exec",
	Modes: []ReqMode{RootMode,UserMode},
	Handler: func(cmd *Command, u *SystemUser, m *ReqMode, eng *CommandEngine) (interface{}, error) {
		fname := cmd.Args["function"].(string)
		fargs := cmd.Args["args"].(map[string]interface{})
		r, e := eng.fm.Exec(fname,fargs)
		return r, e
	},
}

// ############## Engine ############## //

type  CommandEngine struct {
	coreFn map[string] *RuntimeFunction // core functions
	//pipeFn map[string] *RuntimeFunction // extended functions
	Auth *AuthManager // authentication module
	Bfs *BFS // file system module
	SysInit *System // initialization module
	fm *FunctionManager // remote function module
}

func NewCommandEngine(config string) *CommandEngine {
	ci := &CommandEngine{
		coreFn: map[string] *RuntimeFunction {},
	}
	// add  core functions
	ci.coreFn[f1.Name] = f1
	ci.coreFn[f2.Name] = f2
	ci.coreFn[f3.Name] = f3
	ci.coreFn[f4.Name] = f4
	ci.coreFn[f5.Name] = f5
	ci.coreFn[f6.Name] = f6
	ci.coreFn[f7.Name] = f7
	ci.coreFn[f8.Name] = f8
	ci.coreFn[f9.Name] = f9
	ci.coreFn[f10.Name] = f10
	ci.coreFn[f11.Name] = f11
	ci.coreFn[f12.Name] = f12
	ci.coreFn[f13.Name] = f13
	ci.coreFn[f14.Name] = f14
	ci.coreFn[f15.Name] = f15
	ci.coreFn[f16.Name] = f16
	ci.coreFn[f17.Name] = f17
	ci.coreFn[f18.Name] = f18
	ci.coreFn[f19.Name] = f19
	ci.coreFn[f20.Name] = f20
	ci.coreFn[f21.Name] = f21
	ci.coreFn[f22.Name] = f22
	ci.coreFn[f23.Name] = f23
	ci.coreFn[f24.Name] = f24
	ci.coreFn[f25.Name] = f25
	ci.coreFn[f26.Name] = f26
	ci.coreFn[f27.Name] = f27
	ci.coreFn[f28.Name] = f28
	ci.coreFn[f29.Name] = f29
	ci.coreFn[f30.Name] = f30
	ci.coreFn[f31.Name] = f31
	
	// initialize system with config file
	s := &System{}
	err := s.Load(config)
	if err != nil {
		err = errors.New("Couldn't read config file: " + err.Error())
		panic(err)
	}
	ci.SysInit = s

	// mongodb connection
	ms, err := s.MongoConnect()
	if err != nil {
		err = errors.New("Couldn't connect to mongodb: " + err.Error())
		panic(err)
	}

	// redis connection
	rm := s.RedisConnect()

	// authentication module
	auth := NewAuthManager(s.Config, ms, rm)
	ci.Auth = auth

	// file system module
	bfs := NewBFS(s.Config, ms, rm)
	ci.Bfs = bfs
	
	// external functions module
	ci.fm = NewFunctionManager(s.Config.Remote)

	// return
	return ci
}

func (ci *CommandEngine) Do(cmd *Command, user *SystemUser, mode *ReqMode) (interface{}, error) {
	// check core functions
	fn, exists := ci.coreFn[cmd.Name]
	if !exists {
		return nil, errors.New(fmt.Sprintf("Command '%s' not found",cmd.Name))
	}
	
	// check if user authorized to execute command
	grant := false
	for _, item := range fn.Modes {
		if item == *mode {
			grant = true
		}
	}
	if !grant {
		return nil, errors.New(fmt.Sprintf("User not allowed to execute command '%s'",cmd.Name))
	}

	// check if user has access to database
	if cmd.Database != "" && *mode != RootMode {
		dbaccess := false
		for _, item := range user.Databases {
			if cmd.Database == item {
				dbaccess = true
				break
			}
		}
		if !dbaccess {
			return nil, errors.New(fmt.Sprintf("User doesn't have access rights to database '%s'",cmd.Database))
		}
	}

	return fn.Handler(cmd, user, mode, ci)
}

func requestHandler(ch <- chan Job) {
	job := <- ch
	switch job.Req.(type) {
	case InfoRequest:
		req := job.Req.(InfoRequest)
		if req.Name == "version" {
			v := job.Engine.SysInit.Config.General.Version
			job.Req.OnSuccess(v,"text/plain")
			break
		}
		job.Req.OnFailure(fmt.Sprintf("info for '%s' not found",req.Name), ItemNotFoundError)
		break
	case LoginRequest:
		// authenticate		
		u := job.Req.(LoginRequest).Username
		p := job.Req.(LoginRequest).Password
		mode, err := job.Engine.Auth.Authenticate(u,p)
		if err != nil {
			job.Req.OnFailure(err.Error(), AuthError)
			break
		}
		// authorize with new session id
		sid, err := job.Engine.Auth.NewSession(mode, u)
		job.Req.OnSuccess(sid,"application/json")
		break
	case UploadTicketRequest:
		req := job.Req.(UploadTicketRequest)
		// get session info
		sid := req.SessionId
		username, mode, err := job.Engine.Auth.GetSession(sid)
		if err != nil {
			job.Req.OnFailure(err.Error(), ExpiredSessionError)
			break
		}

		switch mode {
		case RootMode:
			_ticket, err := job.Engine.Bfs.NewUploadRequestTicket(req.Path, req.Database)
			if err != nil {
				job.Req.OnFailure(err.Error(), EngineError)
				break
			}
			job.Req.OnSuccess(_ticket,"application/json")
		case GuestMode:
			job.Req.OnFailure("please login.", AccessDeniedError)
			break
		default:
			usr, err := job.Engine.Auth.UserInfo(username)
			if err != nil {
				job.Req.OnFailure(err.Error(), EngineError)
				break
			}

			access := false
			for _, db := range usr.Databases {
				if db == req.Database {
					access = true
					break
				}
			}
			if !access {
				job.Req.OnFailure("user doesn't have access to the database.", AccessDeniedError)
				break
			}
			_ticket, err := job.Engine.Bfs.NewUploadRequestTicket(req.Path, req.Database)
			if err != nil {
				job.Req.OnFailure(err.Error(), EngineError)
				break
			}
			job.Req.OnSuccess(_ticket,"application/json")
		}
	case DownloadRequest:
		req := job.Req.(DownloadRequest)
		// get session info
		sid := req.SessionId
		username, mode, err := job.Engine.Auth.GetSession(sid)
		if err != nil {
			job.Req.OnFailure(err.Error(), ExpiredSessionError)
			break
		}

		switch mode {
		case RootMode:
			_ticket, err := job.Engine.Bfs.GetAttachment(req.Path, req.Database)
			if err != nil {
				job.Req.OnFailure(err.Error(), EngineError)
				break
			}
			job.Req.OnSuccess(_ticket,"application/json")
		case GuestMode:
			job.Req.OnFailure("please login.", AccessDeniedError)
			break
		default:
			usr, err := job.Engine.Auth.UserInfo(username)
			if err != nil {
				job.Req.OnFailure(err.Error(), EngineError)
				break
			}

			access := false
			for _, db := range usr.Databases {
				if db == req.Database {
					access = true
					break
				}
			}
			if !access {
				job.Req.OnFailure("user doesn't have access to the database.", AccessDeniedError)
				break
			}
			_ticket, err := job.Engine.Bfs.GetAttachment(req.Path, req.Database)
			if err != nil {
				job.Req.OnFailure(err.Error(), EngineError)
				break
			}
			job.Req.OnSuccess(_ticket,"application/json")
		}
	case CommandRequest:
		// get session info
		sid := job.Req.(CommandRequest).SessionId
		username, mode, err := job.Engine.Auth.GetSession(sid)
		if err != nil {
			job.Req.OnFailure(err.Error(), ExpiredSessionError)
			break
		}
		// get system user
		var usr SystemUser
		err = nil
		
		switch mode {
		case RootMode:
			usr = SystemUser{ Username:"root", Active:true }
			break
		case GuestMode:
			usr = SystemUser{ Username:"guest", Active:true }
			break
		default:
			usr, err = job.Engine.Auth.UserInfo(username)
			break			
		}
		if err != nil {
			job.Req.OnFailure(err.Error(), EngineError)
			break
		}
		
		// parse script
		prs := NewParser()
		cmdlist, err := prs.Parse(job.Req.(CommandRequest).Script)
		if err != nil {
			job.Req.OnFailure(err.Error(), SyntaxError)
			break
		}
		// process command list
		// only the results of the last command are returned
		var lastresult *interface{}
		for cmdlist.Size() > 0 {
			cmd := cmdlist.LPop().(Command)
			if cmd.Name == "pipe" {
				cmd.Args["resultset"] = *lastresult
			}
			reply, err := job.Engine.Do(&cmd, &usr, &mode)
			if err != nil {
				job.Req.OnFailure(err.Error(), CommandError)
				lastresult = nil
				break
			}
			lastresult = &reply
		}

		if lastresult == nil {
			job.Req.OnFailure("No executable commands found in script.", ScriptError)
			break
		}
		job.Req.OnSuccess(*lastresult,"application/json") // get the value at the address
		break
	case UploadTCheckRequest:
		req := job.Req.(UploadTCheckRequest)
		info, err := job.Engine.Bfs.UploadRequestTicketInfo(req.Ticket)
		if err != nil {
			req.OnFailure(err.Error(), ItemNotFoundError)
			break
		}
		req.OnSuccess(info,"application/json")
		break
	case UploadCompleteRequest:
		req := job.Req.(UploadCompleteRequest)
		err := job.Engine.Bfs.AddAttachment(req.ContentPath, req.UploadFile, req.Database)
		if err != nil {
			req.OnFailure(err.Error(), ItemNotFoundError)
			break
		}
		req.OnSuccess(req.Size,"application/json")
		break
	case DirectAccessRequest:
		req := job.Req.(DirectAccessRequest)
		data, err := job.Engine.Bfs.DirectAccess(req.ContentPath, req.AccessType, req.Database)
		if err != nil{
			req.OnFailure(err.Error(), ItemNotFoundError)
			break
		}
		req.OnSuccess(data,"application/json")
		break
	default:
		// send error
		job.Req.OnFailure("Could not handle job.", EngineError)
	}
}

func (ci *CommandEngine) Run(ch chan Request) {
	// start polling for request on channel(s)
	for {
		select {
		case req := <- ch:
			// create job
			j := Job{
				Req: req,
				Engine: ci,
				Status: JPending,
			}
			// create a buffered channel so no blocking
			// drop job in channel and move to next req
			lchan := make(chan Job, 1)
			go requestHandler(lchan)
			lchan <- j
		}
	}
}
