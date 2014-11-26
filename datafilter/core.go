package datafilter

import (
	"encoding/json"
	"fmt"

	"github.com/johnwilson/bytengine"
)

type FilterFunction func(r *bytengine.Response) bytengine.Response

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
	fn := func(r *bytengine.Response) bytengine.Response {
		if r.Status != bytengine.OK {
			return *r
		}
		b, err := json.MarshalIndent(r.Map(), "", "  ")
		if err != nil {
			return bytengine.ErrorResponse(fmt.Errorf("Pretty print error"))
		}
		return bytengine.OKResponse(string(b))
	}
	regItem := RegistryItem{
		fn,
		"Function to pretty print bytengine query responses.",
	}
	cf.registry["pretty"] = regItem
	return nil
}

func (cf CoreFilters) Apply(filter string, r *bytengine.Response) bytengine.Response {
	regitem, ok := cf.registry[filter]
	if !ok {
		return bytengine.ErrorResponse(fmt.Errorf("Filter '%s' not found", filter))
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
	bytengine.RegisterDataFilter("core", NewCoreFilters())
}
