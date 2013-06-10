package modules

import (
	"testing"
	//"os"
)

func TestParser(t *testing.T) {
	s := `
	@db.select "name" in /tmp/users
	where "name" == "john" file_name=="index.html"`

	p := NewParser()
	l, err := p.Parse(s)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(l.ToList())
}