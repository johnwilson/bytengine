package core

import (
	_ "github.com/johnwilson/bytengine/plugin/auth/mongo"
	_ "github.com/johnwilson/bytengine/plugin/bytestore/diskv"
	_ "github.com/johnwilson/bytengine/plugin/datafilter"
	_ "github.com/johnwilson/bytengine/plugin/filesystem/mongo"
	_ "github.com/johnwilson/bytengine/plugin/statestore/redis"
)

const (
	STATE_PLUGIN       = "redis"   // state store plugin
	AUTH_PLUGIN        = "mongodb" // authentication plugin
	BFS_PLUGIN         = "mongodb" // bytengine file system plugin
	BST_PLUGIN         = "diskv"   // byte store plugin
	DATA_FILTER_PLUGIN = "core"    // data filter function plugin
)
