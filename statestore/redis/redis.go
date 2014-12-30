package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fzzy/radix/redis"
	"github.com/johnwilson/bytengine"
)

type Config struct {
	Address  string        `json:"address"`
	Timeout  time.Duration `json:"timeout"`
	Password string        `json:"password"`
	Database int64         `json:"database"`
}

const (
	TokenPrefix = "token"
	CachePrefix = "cache"
)

type StateStore struct {
	client *redis.Client
	config *Config
}

func (s *StateStore) TokenSet(token, user string, timeout int64) error {
	// check connection
	if err := s.checkConnection(); err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", TokenPrefix, token)
	r := s.client.Cmd("SETEX", key, timeout, user)
	if r.Err != nil {
		return r.Err
	}
	return nil
}

func (s *StateStore) TokenGet(token string) (string, error) {
	// check connection
	if err := s.checkConnection(); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s:%s", TokenPrefix, token)
	r := s.client.Cmd("GET", key)
	if r.Err != nil {
		return "", r.Err
	}
	return r.Str()
}

func (s *StateStore) CacheSet(id, value string, timeout int64) error {
	// check connection
	if err := s.checkConnection(); err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s", CachePrefix, id)
	r := s.client.Cmd("SETEX", key, timeout, value)
	if r.Err != nil {
		return r.Err
	}
	return nil
}

func (s *StateStore) CacheGet(id string) (string, error) {
	// check connection
	if err := s.checkConnection(); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s:%s", CachePrefix, id)
	r := s.client.Cmd("GET", key)
	if r.Err != nil {
		return "", r.Err
	}
	return r.Str()
}

func (s *StateStore) ClearAll() error {
	// check connection
	if err := s.checkConnection(); err != nil {
		return err
	}

	r := s.client.Cmd("FLUSHDB")
	if r.Err != nil {
		return r.Err
	}
	return nil
}

func (s *StateStore) connect() error {
	client, err := redis.DialTimeout(
		"tcp",
		s.config.Address,
		s.config.Timeout*time.Second,
	)
	if err != nil {
		return err
	}

	// check if passowrd given and authenticate accordingly
	if len(s.config.Password) > 0 {
		r := client.Cmd("AUTH", s.config.Password)
		if r.Err != nil {
			return r.Err
		}
	}

	// select the database
	r := client.Cmd("SELECT", s.config.Database)
	if r.Err != nil {
		return r.Err
	}

	// set client
	s.client = client
	return nil
}

func (s *StateStore) checkConnection() error {
	// ping server
	r := s.client.Cmd("PING")
	if r.Err != nil {
		// check if closed connection error
		if r.Err.Error() == "use of closed network connection" {
			return s.connect()
		}
		return r.Err
	}
	return nil
}

func (s *StateStore) Start(config string) error {
	var c Config
	err := json.Unmarshal([]byte(config), &c)
	if err != nil {
		return err
	}
	s.config = &c

	// connect
	return s.connect()
}

func NewStateStore() *StateStore {
	return &StateStore{}
}

func init() {
	bytengine.RegisterStateStore("redis", NewStateStore())
}
