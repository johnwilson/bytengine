package redis

import (
	"github.com/johnwilson/bytengine"
	"testing"
)

func TestStateStore(t *testing.T) {
	// create plugin
	sts, err := bytengine.NewStateStore(
		"redis",
		`{
            "address":"localhost:6379",
            "database":1,
            "password":"",
            "timeout":60
        }`,
	)
	if err != nil {
		t.Fatal(err)
	}

	// add token
	err = sts.TokenSet("token1", "user1", 10)
	if err != nil {
		t.Fatal(err)
	}

	// get token
	val, err := sts.TokenGet("token1")
	if err != nil {
		t.Fatal(err)
	}
	if val != "user1" {
		t.Fatal("Token value mismatch")
	}

	// add cache
	err = sts.CacheSet("1", "cacheitem1", 10)
	if err != nil {
		t.Fatal(err)
	}

	// get cache
	val, err = sts.CacheGet("1")
	if err != nil {
		t.Fatal(err)
	}
	if val != "cacheitem1" {
		t.Fatal("Cache value mismatch")
	}

	// clear all
	err = sts.ClearAll()
	if err != nil {
		t.Fatal(err)
	}
}
