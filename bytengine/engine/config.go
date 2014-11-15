package engine

import (
	_ "github.com/astaxie/beego/cache/redis"       // cache plugin
	_ "github.com/johnwilson/bytengine/auth/mongo" // authentication plugin
	_ "github.com/johnwilson/bytengine/bfs/mongo"  // bytengine file system plugin
	_ "github.com/johnwilson/bytengine/bst/diskv"  // byte store plugin
	_ "github.com/johnwilson/bytengine/fltcore"    // data filter function plugin
)

const (
	CACHE_PLUGIN       = "redis"   // cache plugin
	AUTH_PLUGIN        = "mongodb" // authentication plugin
	BFS_PLUGIN         = "mongodb" // bytengine file system plugin
	BST_PLUGIN         = "diskv"   // byte store plugin
	DATA_FILTER_PLUGIN = "core"    // data filter function plugin
)
