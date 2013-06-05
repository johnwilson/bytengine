package modules

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"path"
	"github.com/vmihailenco/redis"
	"labix.org/v2/mgo"
)

// ------------ Configuration ------------ //
type Configuration struct {
	Bfs ConfigBFS
	Mongodb ConfigMongodb
	Redis ConfigRedis
	General ConfigGeneral
	Web ConfigWeb
	Paginate ConfigPaginate
	Remote ConfigRemote
	Ext map[string]interface{} `json:"ext"`
}
type ConfigGeneral struct {
	Admin string `json:"sa"`
	Password string `json:"password"`
	Version string `json:"version"`
	LogDirectory string `json:"log"`
}
type ConfigPaginate struct {
	Directory string `json:"dir"`
	Timeout int `json:"timeout"`
	PageMemSize int64 `json:"max_page_memory"`
}
type ConfigMongodb struct {
	Host string `json:"host"`
	Port int `json:"port"`
}
type ConfigRedis struct {
	Host string `json:"host"`
	Port int64 `json:"port"`
	Password string `json:"password"`
	UploadDb int64 `json:"uploaddb"`
	AuthDb int64 `json:"authdb"`
	ExtDb int64 `json:"ext"`
	BFSDb int64 `json:"bfsdb"`
	PingInterval int64 `json:"pinginterval"`
}
type ConfigBFS struct {
	ReservedDbs []string `json:"reserved_dbs"`
	ContentCol string `json:"content_collection"`
	CounterCol string `json:"counter_collection"`
	AttachmentsRoot string `json:"attachments_dir"`
	SystemDb string `json:"sysdatabase"`
	SecurityCol string `json:"security_collection"`
	SessionTimeout int `json:"session_timeout"`
	CacheTimeout int `json:"cache_timeout"`
	UploadTimeout int `json:"upload_timeout"`
}
type ConfigWeb struct {
	Host string `json:"host"`
	Port int `json:"port"`
	MaxUploadSize int `json:"maxupload_size"`
	UploadDirectory string `json:"upload_tmp"`
	StaticDirectory string `json:"static_dir"`
}
type ConfigRemote struct {
	Exec ConfigRemoteExec
	Pipe ConfigRemotePipe
}
type ConfigRemoteExec struct {
	Url string `json:"url"`
	Timeout int64 `json:"timeout"`
}
type ConfigRemotePipe struct {
	Url string `json:"url"`
	Timeout int64 `json:"timeout"`
}

// ------------ System initializer ------------ //

type System struct {
	Config *Configuration // parsed from json config
	ROOT_DIR string // system root directory

}

// Web server address
func (si *System) WebUrl() string {
	return fmt.Sprintf("%s:%d", si.Config.Web.Host, si.Config.Web.Port)
}

// Create new Mongodb connection
func (si *System) MongoConnect() (*mgo.Session, error) {
	cn := fmt.Sprintf(
		"mongodb://%s:%d",
		si.Config.Mongodb.Host,
		si.Config.Mongodb.Port,
	)

	sn, e := mgo.Dial(cn)
	if e != nil {
		return nil, e
	}
	return sn, nil
}

type RedisManager struct {
	DbConnect map[string] *redis.Client
}

// Creat new Redis connection manager
func (si *System) RedisConnect() *RedisManager {
	addr := fmt.Sprintf(
		"%s:%d",
		si.Config.Redis.Host,
		si.Config.Redis.Port,

	)	
	pw := si.Config.Redis.Password

	rm := RedisManager{ map[string] *redis.Client {} }
	rm.DbConnect["bfs"] = redis.NewTCPClient(addr, pw, si.Config.Redis.BFSDb)
	rm.DbConnect["auth"] = redis.NewTCPClient(addr, pw, si.Config.Redis.AuthDb)
	rm.DbConnect["ext"] = redis.NewTCPClient(addr, pw, si.Config.Redis.ExtDb)
	rm.DbConnect["upload"] = redis.NewTCPClient(addr, pw, si.Config.Redis.UploadDb)

	return &rm
}

// Read configuration file
func (si *System) Load(f string) error {
	_bs, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}

	var c Configuration
	err = json.Unmarshal(_bs, &c)
	if err != nil {
		return err
	}
	si.Config = &c

	si.ROOT_DIR = path.Join(path.Dir(f),"..")
	return nil
}