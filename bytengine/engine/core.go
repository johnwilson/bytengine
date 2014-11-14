package engine

import (
	"github.com/astaxie/beego/cache"
	"github.com/bitly/go-simplejson"
	"github.com/johnwilson/bytengine/auth"
	"github.com/johnwilson/bytengine/bfs"
	"github.com/johnwilson/bytengine/bst"
	"github.com/johnwilson/bytengine/core"
	"github.com/johnwilson/bytengine/ext"
)

func CreateDataFilter(config *simplejson.Json) ext.DataFilter {
	df, err := ext.NewPlugin(DATA_FILTER_PLUGIN, "")
	if err != nil {
		panic(err)
	}
	return df
}

func CreateAuthManager(config *simplejson.Json) auth.Authentication {
	b, err := config.Get("auth").MarshalJSON()
	if err != nil {
		panic(err)
	}
	authM, err := auth.NewPlugin(AUTH_PLUGIN, string(b))
	if err != nil {
		panic(err)
	}
	return authM
}

func CreateBSTManager(config *simplejson.Json) bst.ByteStore {
	b, err := config.Get("bst").MarshalJSON()
	if err != nil {
		panic(err)
	}
	bstM, err := bst.NewPlugin(BST_PLUGIN, string(b))
	if err != nil {
		panic(err)
	}
	return bstM
}

func CreateBFSManager(bstore *bst.ByteStore, config *simplejson.Json) bfs.BFS {
	b, err := config.Get("bfs").MarshalJSON()
	if err != nil {
		panic(err)
	}
	bfsM, err := bfs.NewPlugin(BFS_PLUGIN, string(b), bstore)
	if err != nil {
		panic(err)
	}
	return bfsM
}

func CreateCacheManager(config *simplejson.Json) cache.Cache {
	b, err := config.Get("cache").MarshalJSON()
	if err != nil {
		panic(err)
	}
	cacheM, err := cache.NewCache(CACHE_PLUGIN, string(b))
	if err != nil {
		panic(err)
	}
	return cacheM
}

func WorkerPool(n int, config *simplejson.Json) (chan *core.ScriptRequest, chan *core.CommandRequest) {
	queries := make(chan *core.ScriptRequest)
	commands := make(chan *core.CommandRequest)

	for i := 0; i < n; i++ {
		authM := CreateAuthManager(config)
		bstM := CreateBSTManager(config)
		bfsM := CreateBFSManager(&bstM, config)
		cacheM := CreateCacheManager(config)
		df := CreateDataFilter(config)
		router := core.NewCommandRouter()
		router.AddFilters(df)
		initialize(router)
		eng := core.Engine{router, authM, bfsM, bstM, cacheM}

		go eng.Start(queries, commands)
	}

	return queries, commands
}

func CreateAdminUser(usr, pw string, config *simplejson.Json) error {
	authM := CreateAuthManager(config)
	err := authM.NewUser(usr, pw, true)
	return err
}
