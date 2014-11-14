package engine

import (
	_ "github.com/astaxie/beego/cache/redis"
	_ "github.com/johnwilson/bytengine/auth/mongo"
	_ "github.com/johnwilson/bytengine/bfs/mongo"
	_ "github.com/johnwilson/bytengine/bst/diskv"
	_ "github.com/johnwilson/bytengine/fltcore"
)

const (
	DATA_FILTER_PLUGIN = "core" // data filter function plugin
	AUTH_PLUGIN        = "mongodb"
	BST_PLUGIN         = "diskv"   // byte store plugin
	BFS_PLUGIN         = "mongodb" // bytengine file system plugin
	CACHE_PLUGIN       = "redis"
)
