package datafilter

import (
	"encoding/json"
	"fmt"

	"github.com/johnwilson/bytengine"
)

func pretty(r *bytengine.Response, eng *bytengine.Engine) bytengine.Response {
	if r.Status != bytengine.OK {
		return *r
	}
	b, err := json.MarshalIndent(r.Map(), "", "  ")
	if err != nil {
		return bytengine.ErrorResponse(fmt.Errorf("Pretty print error"))
	}
	return bytengine.OKResponse(string(b))
}

func init() {
	bytengine.RegisterDataFilter("pretty", pretty)
}
