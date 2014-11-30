package builtin

import (
	"encoding/json"

	"github.com/johnwilson/bytengine"
)

func pretty(r interface{}, eng *bytengine.Engine) (interface{}, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func init() {
	bytengine.RegisterDataFilter("pretty", pretty)
}
