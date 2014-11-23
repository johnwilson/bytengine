package core

import (
	_ "github.com/astaxie/beego/cache/redis" // cache plugin
	_ "github.com/johnwilson/bytengine/plugin/auth/mongo"
	_ "github.com/johnwilson/bytengine/plugin/bytestore/diskv"
	_ "github.com/johnwilson/bytengine/plugin/datafilter"
	_ "github.com/johnwilson/bytengine/plugin/filesystem/mongo"
)

const (
	CACHE_PLUGIN       = "redis"   // cache plugin
	AUTH_PLUGIN        = "mongodb" // authentication plugin
	BFS_PLUGIN         = "mongodb" // bytengine file system plugin
	BST_PLUGIN         = "diskv"   // byte store plugin
	DATA_FILTER_PLUGIN = "core"    // data filter function plugin
)
