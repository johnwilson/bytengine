package datafilter

import (
	bfs "github.com/johnwilson/bytengine/filesystem"
)

type DataFilter interface {
	Start(config string) error
	Apply(filter string, r *bfs.Response) bfs.Response
	Info(filter string) string
	Check(filter string) bool
	All() []string
}
