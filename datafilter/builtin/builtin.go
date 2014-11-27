package builtin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/johnwilson/bytengine"
)

func pretty(r *bytengine.Response, eng *bytengine.Engine) (bytengine.Response, error) {
	if r.Status != bytengine.OK {
		return *r, errors.New("Filter function can only be applied to OK responses")
	}
	b, err := json.MarshalIndent(r.Map(), "", "  ")
	if err != nil {
		return bytengine.ErrorResponse(fmt.Errorf("Pretty print error")), err
	}
	return bytengine.OKResponse(string(b)), nil
}

func init() {
	bytengine.RegisterDataFilter("pretty", pretty)
}
