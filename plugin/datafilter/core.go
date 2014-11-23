package datafilter

import (
	"encoding/json"
	"fmt"

	bfs "github.com/johnwilson/bytengine/filesystem"
	"github.com/johnwilson/bytengine/plugin"
)

type FilterFunction func(r *bfs.Response) bfs.Response

type RegistryItem struct {
	fn          FilterFunction
	description string
}

type CoreFilters struct {
	registry map[string]RegistryItem
}

func NewCoreFilters() *CoreFilters {
	return &CoreFilters{}
}

func (cf *CoreFilters) Start(config string) error {
	cf.registry = map[string]RegistryItem{}
	// add filters to registry
	fn := func(r *bfs.Response) bfs.Response {
		if r.Status != bfs.OK {
			return *r
		}
		b, err := json.MarshalIndent(r.Map(), "", "  ")
		if err != nil {
			return bfs.ErrorResponse(fmt.Errorf("Pretty print error"))
		}
		return bfs.OKResponse(string(b))
	}
	regItem := RegistryItem{
		fn,
		"Function to pretty print bytengine query responses.",
	}
	cf.registry["pretty"] = regItem
	return nil
}

func (cf CoreFilters) Apply(filter string, r *bfs.Response) bfs.Response {
	regitem, ok := cf.registry[filter]
	if !ok {
		return bfs.ErrorResponse(fmt.Errorf("Filter '%s' not found", filter))
	}
	return regitem.fn(r)
}

func (cf CoreFilters) Info(filter string) string {
	return "not yet implemented"
}

func (cf CoreFilters) All() []string {
	return []string{"pretty"}
}

func (cf CoreFilters) Check(filter string) bool {
	_, ok := cf.registry[filter]
	return ok
}

func init() {
	plugin.Register("core", NewCoreFilters())
}
