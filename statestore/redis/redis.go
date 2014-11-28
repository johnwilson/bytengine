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
}

func (s *StateStore) TokenSet(token, user string, timeout int64) error {
	key := fmt.Sprintf("%s:%s", TokenPrefix, token)
	r := s.client.Cmd("SETEX", key, timeout, user)
	if r.Err != nil {
		return r.Err
	}
	return nil
}

func (s *StateStore) TokenGet(token string) (string, error) {
	key := fmt.Sprintf("%s:%s", TokenPrefix, token)
	r := s.client.Cmd("GET", key)
	if r.Err != nil {
		return "", r.Err
	}
	return r.Str()
}

func (s *StateStore) CacheSet(id, value string, timeout int64) error {
	key := fmt.Sprintf("%s:%s", CachePrefix, id)
	r := s.client.Cmd("SETEX", key, timeout, value)
	if r.Err != nil {
		return r.Err
	}
	return nil
}

func (s *StateStore) CacheGet(id string) (string, error) {
	key := fmt.Sprintf("%s:%s", CachePrefix, id)
	r := s.client.Cmd("GET", key)
	if r.Err != nil {
		return "", r.Err
	}
	return r.Str()
}

func (s *StateStore) ClearAll() error {
	r := s.client.Cmd("FLUSHDB")
	if r.Err != nil {
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

	client, err := redis.DialTimeout("tcp", c.Address, c.Timeout*time.Second)
	if err != nil {
		return err
	}

	// check if passowrd given and authenticate accordingly
	if len(c.Password) > 0 {
		r := client.Cmd("AUTH", c.Password)
		if r.Err != nil {
			return r.Err
		}
	}

	// select the database
	r := client.Cmd("SELECT", c.Database)
	if r.Err != nil {
		return r.Err
	}

	s.client = client
	return nil
}

func NewStateStore() *StateStore {
	return &StateStore{}
}

func init() {
	bytengine.RegisterStateStore("redis", NewStateStore())
}
