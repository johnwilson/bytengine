package main

import (
	_ "github.com/johnwilson/bytengine/auth/mongo"
	_ "github.com/johnwilson/bytengine/bytestore/diskv"
	_ "github.com/johnwilson/bytengine/bytestore/mongo"
	_ "github.com/johnwilson/bytengine/cmdhandler/base"
	_ "github.com/johnwilson/bytengine/datafilter/builtin"
	_ "github.com/johnwilson/bytengine/filesystem/mongo"
	_ "github.com/johnwilson/bytengine/parser/base"
	_ "github.com/johnwilson/bytengine/statestore/redis"
)
