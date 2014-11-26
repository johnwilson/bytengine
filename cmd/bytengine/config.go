package main

import (
	_ "github.com/johnwilson/bytengine/auth/mongo"
	_ "github.com/johnwilson/bytengine/bytestore/diskv"
	_ "github.com/johnwilson/bytengine/datafilter"
	_ "github.com/johnwilson/bytengine/filesystem/mongo"
	_ "github.com/johnwilson/bytengine/statestore/redis"
)

const (
	STATE_PLUGIN       = "redis"   // state store plugin
	AUTH_PLUGIN        = "mongodb" // authentication plugin
	BFS_PLUGIN         = "mongodb" // bytengine file system plugin
	BST_PLUGIN         = "diskv"   // byte store plugin
	DATA_FILTER_PLUGIN = "core"    // data filter function plugin
)
