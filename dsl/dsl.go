package dsl

import (
	"encoding/json"
)

type Command struct {
	Name     string
	Database string
	Args     map[string]interface{}
	Filter   string // name of filter to apply to result
	isAdmin  bool   // admin command
}

func (c *Command) IsAdmin() bool {
	return c.isAdmin
}

func (c Command) String() string {
	val := map[string]interface{}{
		"command":  c.Name,
		"database": c.Database,
		"filter":   c.Filter,
		"args":     c.Args,
		"isadmin":  c.isAdmin,
	}
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func NewCommand(name string, isadmin bool) Command {
	cmd := Command{
		Name:    name,
		isAdmin: isadmin,
		Args:    map[string]interface{}{},
	}
	return cmd
}
