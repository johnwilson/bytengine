package dsl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	s := `server.listdb --regex="^\w"`
	result := []item{}
	l := lex(s)
	var token item
	for token.typ != itemEOF {
		token = l.nextItem()
		result = append(result, token)
	}
	assert.Len(t, result, 8, "wrong number of tokens")
	assert.Equal(t, result[0].val, "server", "wrong token at position 0")
	assert.Equal(t, result[1].val, ".", "wrong token at position 1")
	assert.Equal(t, result[2].val, "listdb", "wrong token at position 2")
	assert.Equal(t, result[3].val, "--", "wrong token at position 3")
	assert.Equal(t, result[4].val, "regex", "wrong token at position 4")
	assert.Equal(t, result[5].val, "=", "wrong token at position 5")
	assert.Equal(t, result[6].val, `"^\w"`, "wrong token at position 6")
	assert.Equal(t, result[7].val, "", "wrong token at position 7")
}

func TestServerCommands(t *testing.T) {
	s := `server.listdb --regex="^\w"`
	p := NewParser()
	cmdlist, err := p.Parse(s)
	assert.Nil(t, err, fmt.Sprintf("parsing failed: %s", err))
	assert.Len(t, cmdlist, 1, "wrong number of commands parsed")
	cmd := cmdlist[0]
	assert.Equal(t, cmd.Name, "server.listdb", "wrong command name")
	assert.Equal(t, cmd.Options["regex"].(string), `"^\w"`, "wrong regex value")
}
