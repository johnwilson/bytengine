package base

import (
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"math"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/johnwilson/bytengine"
)

// Parser scans job.Request.Script for commands and adds them to job.CommandQueue.
type Parser struct {
	commands  []bytengine.Command
	cmdlookup map[string]map[string]interface{}
	// Parsing only; cleared after parse.
	lex       *lexer
	token     [2]item // two-token lookahead for parser.
	peekCount int
}

// next returns the next token.
func (p *Parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.nextItem()
	}
	return p.token[p.peekCount]
}

// backup backs the input stream up one token.
func (p *Parser) backup() {
	p.peekCount++
}

// backup2 backs the input stream up two tokens
func (p *Parser) backup2(t1 item) {
	p.token[1] = t1
	p.peekCount = 2
}

// peek returns but does not consume the next token.
func (p *Parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lex.nextItem()
	return p.token[0]
}

// Parsing.

// Create a new parser pointer.
func NewParser() *Parser {
	p := &Parser{}
	p.initRegistry()
	return p
}

// errorf formats the error and terminates processing.
func (p *Parser) errorf(format string, args ...interface{}) {
	p.commands = nil
	format = fmt.Sprintf("line[%d]: %s", p.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (p *Parser) error(err error) {
	p.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (p *Parser) expect(expected itemType, context string) item {
	token := p.next()
	if token.typ != expected {
		p.errorf("expected %s in %s; got %s", expected, context, token)
	}
	return token
}

// expectEither consumes the next token and guarantees it has one of the required types.
func (p *Parser) expectOneOf(expected1, expected2 itemType, context string) item {
	token := p.next()
	if token.typ != expected1 && token.typ != expected2 {
		p.errorf("expected %s or %s in %s; got %s", expected1, expected2, context, token)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (p *Parser) unexpected(token item, context string) {
	p.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (p *Parser) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if p != nil {
			p.stopParse()
		}
		*errp = e.(error)
	}
	return
}

// startParse initializes the parser, using the lexer.
func (p *Parser) startParse(lex *lexer) {
	p.commands = make([]bytengine.Command, 0)
	p.lex = lex
}

// stopParse terminates parsing.
func (p *Parser) stopParse() {
	p.lex = nil
	p.token = [2]item{}
	p.peekCount = 0
}

// atEOF returns true if, possibly after spaces, we're at EOF.
func (p *Parser) atEOF() bool {
	for {
		token := p.peek()
		switch token.typ {
		case itemEOF:
			return true
		}
		break
	}
	return false
}

func (p *Parser) Parse(s string) (c []bytengine.Command, err error) {
	defer p.recover(&err)
	p.startParse(lex(s))
	p.parse()
	p.stopParse()
	return p.commands, nil
}

// It runs to EOF.
// all commands and sub-commands are made case insensitive with strings.ToLower()
func (p *Parser) parse() map[string]interface{} {
	for p.peek().typ != itemEOF {
		switch _next := p.peek(); {
		case _next.typ == itemError:
			p.errorf("Parsing error: %s", p.peek().val)
		case _next.typ == itemIdentifier:
			switch cmdprefix := strings.ToLower(_next.val); {
			case cmdprefix == "server", cmdprefix == "user":
				context := cmdprefix
				// absorb keyword
				p.next()
				p.expect(itemDot, context)
				item := p.expect(itemIdentifier, context)
				key := strings.ToLower(item.val)
				// update context
				context = cmdprefix + "." + key
				// lookup key
				if val, ok := p.cmdlookup[cmdprefix][key]; ok {
					if fn, ok := val.(func(ctx string)); ok {
						fn(context)
					} else {
						p.errorf("Invalid %s function type", context)
					}
				} else {
					p.errorf("%s parse function not found", context)
				}
			default:
				p.errorf("Invalid command prefix '%s'", cmdprefix)
			}
		case _next.typ == itemDatabase:
			cmdprefix := "database"
			db := p.next().val
			p.expect(itemDot, cmdprefix)
			item := p.expect(itemIdentifier, cmdprefix)
			key := strings.ToLower(item.val)
			context := cmdprefix + "." + key
			if val, ok := p.cmdlookup[cmdprefix][key]; ok {
				if fn, ok := val.(func(db, ctx string)); ok {
					fn(db, context)
				} else {
					p.errorf("Invalid %s function type", context)
				}
			} else {
				p.errorf("Invalid command database command '%s'", key)
			}
		default:
			//absorb unknown token
			p.next()
		}
	}

	return nil
}

// copied from go source docs: strconv.Unquote
// removed restriction of single quote 1 character length
func unquote(s string) (t string, err error) {
	n := len(s)
	if n < 2 {
		return "", strconv.ErrSyntax
	}
	quote := s[0]
	if quote != s[n-1] {
		return "", strconv.ErrSyntax
	}
	s = s[1 : n-1]

	if quote == '`' {
		if contains(s, '`') {
			return "", strconv.ErrSyntax
		}
		return s, nil
	}
	if quote != '"' && quote != '\'' {
		return "", strconv.ErrSyntax
	}
	if contains(s, '\n') {
		return "", strconv.ErrSyntax
	}

	// Is it trivial?  Avoid allocation.
	if !contains(s, '\\') && !contains(s, quote) {
		switch quote {
		case '"':
			return s, nil
		case '\'':
			r, size := utf8.DecodeRuneInString(s)
			if size == len(s) && (r != utf8.RuneError || size != 1) {
				return s, nil
			}
		}
	}

	var runeTmp [utf8.UTFMax]byte
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.
	for len(s) > 0 {
		c, multibyte, ss, err := strconv.UnquoteChar(s, quote)
		if err != nil {
			return "", err
		}
		s = ss
		if c < utf8.RuneSelf || !multibyte {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
	}
	return string(buf), nil
}

// contains reports whether the string contains the byte c.
func contains(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// remove quotes from string tokens
func formatString(s string) (string, error) {
	_tmp, err := unquote(s)
	if err != nil {
		return "", err
	}
	return _tmp, nil
}

// field prefix: this is necessary because a BFS file format is as follows
// {"__header__":{...}, "__bytes__":{...}, "content": {...}}
// hence mongodb queries would require the 'content.' prefix
const FieldPrefix string = "content."
const HeaderPrefix string = "__header__."
const BytesPrefix string = "__bytes__."

// send to filter function parser
func (p *Parser) parseFilterResult() string {
	if p.peek().typ == itemSendTo {
		// absorb symbol
		p.next()
		_nxt := p.expect(itemIdentifier, "Result assignment")
		return _nxt.val
	}
	return ""
}

// command option parser
func (p *Parser) parseCommandOption(ctx string) (name, val string) {
	// absorb option symbol
	p.next()
	_token := p.expect(itemIdentifier, ctx)
	name = _token.val
	if p.peek().typ == itemEqual {
		// absorb equal
		p.next()
		optval := p.expectOneOf(itemString, itemNumber, ctx)
		if optval.typ == itemNumber {
			val = optval.val
		} else {
			var err error
			val, err = formatString(optval.val)
			if err != nil {
				p.errorf("Improperly quoted string value in %s", ctx)
			}
		}
	}
	return
}

// end of command statement parser
func (p *Parser) parseEndofCommand(ctx string) string {
	filter := p.parseFilterResult()
	_nxt := p.expectOneOf(itemSemiColon, itemEOF, ctx)
	if _nxt.typ == itemEOF {
		p.backup()
	}
	return filter
}

// login parser
func (p *Parser) parseLoginCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_usr, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_token = p.expect(itemString, ctx)
	_pw, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted password in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _usr
	cmd.Args["password"] = _pw
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// list databases parser
func (p *Parser) parseListDatabasesCmd(ctx string) {
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,

		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{})} // check if regex option has been added
	if p.peek().typ == itemArgument {
		name, val := p.parseCommandOption(ctx)
		if name != "regex" || len(val) == 0 {
			p.errorf("Invalid option %s in %s", name, ctx)
		}
		cmd.Options[name] = val
	}
	_filter := p.parseEndofCommand(ctx)
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// create new database parser
func (p *Parser) parseNewDatabaseCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_db, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted database name in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["database"] = _db
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// delete database parser
func (p *Parser) parseDropDatabaseCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_db, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted database name in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["database"] = _db
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// current user info parser
func (p *Parser) parseWhoamiCmd(ctx string) {
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// create new user parser
func (p *Parser) parseNewUserCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_user, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_token = p.expect(itemString, ctx)
	_pw, err2 := formatString(_token.val)
	if err2 != nil {
		p.errorf("Improperly quoted password in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _user
	cmd.Args["password"] = _pw
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// list users parser
func (p *Parser) parseListUsersCmd(ctx string) {
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,

		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{})} // check if regex option has been added
	if p.peek().typ == itemArgument {
		name, val := p.parseCommandOption(ctx)
		if name != "regex" || len(val) == 0 {
			p.errorf("Invalid option %s in %s", name, ctx)
		}
		cmd.Options[name] = val
	}
	_filter := p.parseEndofCommand(ctx)
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// user info parser
func (p *Parser) parseUserInfoCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_user, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _user
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// delete user parser
func (p *Parser) parseDropUserCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_user, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _user
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// new user password parser
func (p *Parser) parseNewPasswordCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_user, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_token = p.expect(itemString, ctx)
	_pw, err2 := formatString(_token.val)
	if err2 != nil {
		p.errorf("Improperly quoted password in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _user
	cmd.Args["password"] = _pw
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// activate/deactivate user account parser
func (p *Parser) parseUserSystemAccessCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_user, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_token = p.expect(itemIdentifier, ctx)
	_grant := false
	switch _token.val {
	case "grant":
		_grant = true
	case "deny":
		// do nothing _grant already false
		break
	default:
		p.errorf("Invalid indentifier "+_token.val+" in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _user
	cmd.Args["grant"] = _grant
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// grant/deny user database access parser
func (p *Parser) parseUserDatabaseAccessCmd(ctx string) {
	_token := p.expect(itemString, ctx)
	_user, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted username in %s", ctx)
	}
	_token = p.expect(itemString, ctx)
	_db, err2 := formatString(_token.val)
	if err2 != nil {
		p.errorf("Improperly quoted database in %s", ctx)
	}
	_token = p.expect(itemIdentifier, ctx)
	_grant := false
	switch _token.val {
	case "grant":
		_grant = true
	case "deny":
		// do nothing _grant already false
		break
	default:
		p.errorf("Invalid indentifier "+_token.val+" in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Args["username"] = _user
	cmd.Args["database"] = _db
	cmd.Args["grant"] = _grant
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// initialize bytengine parser
func (p *Parser) parseServerInitCmd(ctx string) {
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: true,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// create new directory parser
func (p *Parser) parseNewDirectoryCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// create new file parser
func (p *Parser) parseNewFileCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val

	// check if next item is a json object
	var _json interface{}
	if p.peek().typ == itemLeftBrace {
		_json = p.parseJSON(ctx)
	} else {
		p.errorf("Expecting a JSON object in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Args["data"] = _json
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// list directory contents parser
func (p *Parser) parseListDirectoryCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path

	// check if regex option has been added
	if p.peek().typ == itemArgument {
		name, val := p.parseCommandOption(ctx)
		if name != "regex" || len(val) == 0 {
			p.errorf("Invalid option %s in %s", name, ctx)
		}
		cmd.Options[name] = val
	}
	_filter := p.parseEndofCommand(ctx)
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// rename file/directory parser
func (p *Parser) parseRenameContentCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_token = p.expect(itemString, ctx)
	_name, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted new name in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Args["name"] = _name
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// move file/directory parser
func (p *Parser) parseMoveContentCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_token = p.expect(itemPath, ctx)
	_path2 := _token.val
	_rename := ""
	if p.peek().typ == itemString {
		_nxt := p.next()
		_rename = _nxt.val
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Args["to"] = _path2
	cmd.Args["rename"] = _rename
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// copy file/directory parser
func (p *Parser) parseCopyContentCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_token = p.expect(itemPath, ctx)
	_path2 := _token.val
	_rename := ""
	if p.peek().typ == itemString {
		_nxt := p.next()
		_rename = _nxt.val
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Args["to"] = _path2
	cmd.Args["rename"] = _rename
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// delete file/directory parser
func (p *Parser) parseDeleteContentCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// file/directory info parser
func (p *Parser) parseContentInfoCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// make file public parser
func (p *Parser) parseMakeContentPublicCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// make file private parser
func (p *Parser) parseMakeContentPrivateCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// read file JSON content parser
func (p *Parser) parseReadFileCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	// check if we have an array of fields to return
	_list := []string{}
	if p.peek().typ == itemLeftBracket {
		// parse array and make sure all elements are strings
		// absorb left bracket
		p.next()
		// check type of next token
	Loop:
		for {
			switch p.peek().typ {
			case itemString:
				_next := p.next()
				_val, err := formatString(_next.val)
				if err != nil {
					p.errorf("Improperly quoted string value in %s Array definition.", ctx)
				}
				_list = append(_list, _val)
				continue
			case itemComma:
				// absorb comma
				p.next()
				if p.peek().typ == itemRightBracket {
					p.errorf("Trailing comma ',' in %s Array definition.", ctx)
				}
				continue
			case itemRightBracket:
				// absorb
				p.next()
				break Loop
			default:
				p.errorf("Invalid value: "+p.peek().val+" in %s Array definition.", ctx)
			}
		}
	}

	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Args["fields"] = _list
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// overwrite file JSON parser
func (p *Parser) parseModifyFileCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val

	// check if next item is a json object
	var _json interface{}
	if p.peek().typ == itemLeftBrace {
		_json = p.parseJSON(ctx)
	} else {
		p.errorf("Expecting a JSON object in %s", ctx)
	}
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Args["data"] = _json
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// delete file bytes parser
func (p *Parser) parseDeleteAttachmentCmd(db, ctx string) {
	_token := p.expect(itemPath, ctx)
	_path := _token.val
	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["path"] = _path
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// increment/decrement/list counter parser
func (p *Parser) parseCounterCmd(db, ctx string) {
	_token := p.expectOneOf(itemString, itemIdentifier, ctx)
	if _token.typ == itemIdentifier {
		if _token.val == "list" {
			cmd := bytengine.Command{
				Name:    ctx,
				IsAdmin: false,
				Args:    make(map[string]interface{}),
				Options: make(map[string]interface{}),
			}
			cmd.Database = db
			cmd.Args["action"] = "list"

			// check if regex option has been added
			if p.peek().typ == itemArgument {
				name, val := p.parseCommandOption(ctx)
				if name != "regex" || len(val) == 0 {
					p.errorf("Invalid option %s in %s", name, ctx)
				}
				cmd.Options[name] = val
			}
			_filter := p.parseEndofCommand(ctx)
			cmd.Filter = _filter
			p.commands = append(p.commands, cmd)
			return

		} else {
			p.errorf("Invalid identifier %s in %s", _token.val, ctx)
		}
	}

	_counter, err := formatString(_token.val)
	if err != nil {
		p.errorf("Improperly quoted counter name in %s", ctx)
	}
	_token = p.expect(itemIdentifier, ctx)
	_action := ""
	switch _token.val {
	case "incr":
		fallthrough
	case "decr":
		fallthrough
	case "reset":
		_action = _token.val
	default:
		p.errorf("Invalid indentifier "+_token.val+" in %s", ctx)
	}

	_token = p.expect(itemNumber, ctx)
	_val, err := strconv.ParseInt(_token.val, 10, 64) // base 10 64bit integer
	if err != nil {
		p.errorf("Invalid numerical value in %s", ctx)
	}

	_filter := p.parseEndofCommand(ctx)
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["name"] = _counter
	cmd.Args["action"] = _action
	cmd.Args["value"] = _val
	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// select query statement parser
func (p *Parser) parseSelectCmd(db, ctx string) {
	_fields := []string{}
	// get fields
	for p.peek().typ == itemString {
		_token := p.next()
		_field, err := formatString(_token.val)
		if err != nil {
			p.errorf("Improperly quoted field name in %s", ctx)
		}
		_fields = append(_fields, FieldPrefix+_field)
		continue
	}
	// get directories
	_in := p.expect(itemIdentifier, ctx)
	if strings.ToLower(_in.val) != "in" {
		p.errorf("Invalid %s, expecting 'In statement'.", ctx)
	}
	_paths := []string{}
	for p.peek().typ == itemPath {
		_path := p.next().val
		_paths = append(_paths, _path)
		continue
	}
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["fields"] = _fields
	cmd.Args["dirs"] = _paths
	var _filter string

	// get optional identifiers
Loop:
	for {
		switch _token := p.next(); {
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "where":
			_where := p.parseWhereCmd()
			cmd.Args["where"] = _where
			continue
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "sort":
			cmd.Args["sort"] = p.parseSortCmd()
			continue
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "limit":
			// add to select statement
			cmd.Args["limit"] = p.parseLimitCmd()
			continue
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "distinct":
			// add to select statement
			cmd.Args["distinct"] = p.parseDistinctCmd()
			continue
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "count":
			cmd.Args["count"] = true
			continue
		case _token.typ == itemSendTo:
			p.backup()
			_filter = p.parseEndofCommand(ctx)
			break Loop
		case _token.typ == itemSemiColon:
			break Loop
		case _token.typ == itemEOF:
			p.backup()
			break Loop
		default:
			p.errorf("Invalid identifier "+_token.val+" in %s", ctx)
		}
	}

	// validate select statement
	_, hascount := cmd.Args["count"]
	_, haslimit := cmd.Args["limit"]
	_, hassort := cmd.Args["sort"]
	_, hasdistinct := cmd.Args["distinct"]

	if haslimit || hassort {
		if hascount {
			p.errorf("'Count' cannot be used with 'Limit' or 'Sort' in %s", ctx)
		}
		if hasdistinct {
			p.errorf("'Distinct' cannot be used with 'Limit' or 'Sort' in %s", ctx)
		}
	} else if hasdistinct {
		if haslimit {
			p.errorf("'Limit' cannot be used with 'Distinct' in %s", ctx)
		}
		if hassort {
			p.errorf("'Sort' cannot be used with 'Distinct' in %s", ctx)
		}
		if hascount {
			p.errorf("'Count' cannot be used with 'Distinct' in %s", ctx)
		}
	} else if hascount {
		if haslimit {
			p.errorf("'Limit' cannot be used with 'Count' in %s", ctx)
		}
		if hassort {
			p.errorf("'Sort' cannot be used with 'Count' in %s", ctx)
		}
		if hasdistinct {
			p.errorf("'Distinct' cannot be used with 'Count' in %s", ctx)
		}
	}

	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// set query statement parser
func (p *Parser) parseSetCmd(db, ctx string) {
	_fields := map[string]interface{}{}
	_incr := map[string]interface{}{}

	// get field assignment list
Loop:
	for {
		switch i := p.next(); i.typ {
		case itemString:
			switch p.peek().typ {
			case itemEqual:
				p.backup2(i)
				f, v := p.parseValueAssignment()
				_fields[f] = v
				continue
			case itemPlusEqual:
				fallthrough
			case itemMinusEqual:
				p.backup2(i)
				f, v := p.parseIncrDecrValue()
				_incr[f] = v
				continue
			default:
				p.errorf("Invalid assingment operator in %s", ctx)
			}

		default:
			p.backup()
			break Loop
		}
	}
	if len(_fields) < 1 && len(_incr) < 1 {
		p.errorf("Invalid %s: no field assignments found", ctx)
	}

	// get directories
	_in := p.expect(itemIdentifier, ctx)
	if strings.ToLower(_in.val) != "in" {
		p.errorf("Invalid %s, expecting 'In statement'.", ctx)
	}
	_paths := []string{}
	for p.peek().typ == itemPath {
		_path := p.next().val
		_paths = append(_paths, _path)
		continue
	}
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["fields"] = _fields
	if len(_incr) > 0 {
		cmd.Args["incr"] = _incr
	}
	cmd.Args["dirs"] = _paths
	var _filter string

	// get optional identifiers
Loop2:
	for {
		switch _token := p.next(); {
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "where":
			_where := p.parseWhereCmd()
			cmd.Args["where"] = _where
			continue
		case _token.typ == itemSendTo:
			p.backup()
			_filter = p.parseEndofCommand(ctx)
			break Loop2
		case _token.typ == itemSemiColon:
			break Loop2
		case _token.typ == itemEOF:
			// do not consume eof
			p.backup()
			break Loop2
		default:
			p.errorf("Invalid identifier "+_token.val+" in %s", ctx)
		}
	}

	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// unset query statement parser
func (p *Parser) parseUnsetCmd(db, ctx string) {
	_fields := map[string]interface{}{}
	// get fields
	for p.peek().typ == itemString {
		_token := p.next()
		_field, err := formatString(_token.val)
		if err != nil {
			p.errorf("Improperly quoted field name in %s", ctx)
		}
		_field = FieldPrefix + _field
		_fields[_field] = 1
		continue
	}
	if len(_fields) < 1 {
		p.errorf("Invalid %s: no fields found", ctx)
	}

	// get directories
	_in := p.expect(itemIdentifier, ctx)
	if strings.ToLower(_in.val) != "in" {
		p.errorf("Invalid %s, expecting 'In statement'.", ctx)
	}
	_paths := []string{}
	for p.peek().typ == itemPath {
		_path := p.next().val
		_paths = append(_paths, _path)
		continue
	}
	cmd := bytengine.Command{
		Name:    ctx,
		IsAdmin: false,
		Args:    make(map[string]interface{}),
		Options: make(map[string]interface{}),
	}
	cmd.Database = db
	cmd.Args["fields"] = _fields
	cmd.Args["dirs"] = _paths
	var _filter string

	// get optional identifiers
Loop2:
	for {
		switch _token := p.next(); {
		case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "where":
			_where := p.parseWhereCmd()
			cmd.Args["where"] = _where
			continue
		case _token.typ == itemSendTo:
			p.backup()
			_filter = p.parseEndofCommand(ctx)
			break Loop2
		case _token.typ == itemSemiColon:
			break Loop2
		case _token.typ == itemEOF:
			// do not consume eof
			p.backup()
			break Loop2
		default:
			p.errorf("Invalid identifier "+_token.val+" in %s", ctx)
		}
	}

	cmd.Filter = _filter
	p.commands = append(p.commands, cmd)
}

// sort query statement parser
func (p *Parser) parseSortCmd() []string {
	context := "Select Sort Statement"
	_order := p.expect(itemIdentifier, context)
	var prefix string
	switch strings.ToLower(_order.val) {
	case "asc":
		prefix = ""
	case "desc":
		prefix = "-"
	default:
		p.errorf("%s error: Expected 'Asc' or 'Desc'.", context)
	}
	_sort := []string{}
	for p.peek().typ == itemString {
		field, err := formatString(p.next().val)
		// add sort prefix
		field = prefix + FieldPrefix + field
		if err != nil {
			p.errorf("Improperly quoted field name in %s", context)
		}
		_sort = append(_sort, field)
		continue
	}
	return _sort
}

// limit query statement parser
func (p *Parser) parseLimitCmd() int64 {
	context := "Select Limit Statement"
	_next := p.expect(itemNumber, context)
	// check if value is Int
	_number, err := strconv.ParseInt(_next.val, 10, 64)
	if err != nil {
		p.errorf("%s error: Limit requires an integer value.", context)
	}
	return _number
}

// distinct query statement parser
func (p *Parser) parseDistinctCmd() string {
	context := "Select Distinct Statement"
	_next := p.expect(itemString, context)
	field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field name in %s", context)
	}
	field = FieldPrefix + field
	return field
}

// array value parser
func (p *Parser) parseArray() []interface{} {
	context := "Select Statement Array Definition"
	// absorb left bracket
	p.next()
	_list := []interface{}{}
	// check type of next token
Loop:
	for {
		switch p.peek().typ {
		case itemString:
			_next := p.next()
			_val, err := formatString(_next.val)
			if err != nil {
				p.errorf("Improperly quoted string value in %s", context)
			}
			_list = append(_list, _val)
			continue
		case itemBool:
			_next := p.next()
			if _next.val == "false" {
				_list = append(_list, false)
			} else {
				_list = append(_list, true)
			}
			continue
		case itemNull:
			p.next()
			_list = append(_list, nil)
		case itemNumber:
			_next := p.next()
			// go uses float64 for json numerical values
			_val, err := strconv.ParseFloat(_next.val, 64)
			if err != nil {
				p.errorf("Invalid numerical value in %s", context)
			}
			_list = append(_list, _val)
			continue
		case itemLeftBracket:
			// recursion
			_val := p.parseArray()
			_list = append(_list, _val)
			continue
		case itemComma:
			// absorb comma
			p.next()
			if p.peek().typ == itemRightBracket {
				p.errorf("Trailing comma ',' in %s", context)
			}
			continue
		case itemLeftBrace:
			// json object
			_val := p.parseJSON(context)
			_list = append(_list, _val)
			continue
		case itemRightBracket:
			// absorb
			p.next()
			break Loop
		default:
			p.errorf("Invalid value: "+p.peek().val+" in %s", context)
		}
	}
	return _list
}

// string value parser
func (p *Parser) parseString() string {
	context := "String Definition"
	_next := p.next()
	_val, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted string value in %s", context)
	}
	return _val
}

// numeric value parser (only float)
func (p *Parser) parseNumber() float64 {
	context := "Number Definition"
	_next := p.next()
	_val, err := strconv.ParseFloat(_next.val, 64)
	if err != nil {
		p.errorf("Invalid numerical value in %s", context)
	}
	return _val
}

// boolean value parser
func (p *Parser) parseBoolean() bool {
	_next := p.next()
	if _next.val == "false" {
		return false
	}
	return true
}

// json value parser
func (p *Parser) parseJSON(context string) map[string]interface{} {
	// check if next item is a json object
	_objlevel := 0
	_json := ""
Loop:
	for {
		switch _next := p.next(); {
		case _next.typ == itemError:
			p.errorf("Parsing error: %s", _next.val)
			break Loop
		case _next.typ == itemLeftBrace:
			_objlevel += 1
			_json += _next.val
		case _next.typ == itemRightBrace:
			_objlevel -= 1
			_json += _next.val
			// check if end of JSON object
			if _objlevel == 0 {
				break Loop
			}
		case _next.typ == itemEOF:
			if _objlevel != 0 {
				p.errorf("Invalid json object in %s", context)
			}
			p.backup()
			break Loop
		case _next.typ == itemSemiColon:
			if _objlevel != 0 {
				p.errorf("Invalid json object in %s", context)
			}
			p.backup()
			break Loop
		default:
			_json += _next.val
		}
	}

	// validate json object
	var _i map[string]interface{}
	_b := []byte(_json)
	err := json.Unmarshal(_b, &_i)
	if err != nil {
		p.errorf("Invalid json object in %s", context)
	}

	return _i
}

// value assignment parser
func (p *Parser) parseValueAssignment() (string, interface{}) {
	context := "Assignment Statement"
	// get field
	_next := p.next()
	_field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field name in %s", context)
	}
	_field = FieldPrefix + _field
	//absorb equals
	p.expect(itemEqual, context)
	// get value string, array, json, number
	var _val interface{}
	switch p.peek().typ {
	case itemString:
		_val = p.parseString()
	case itemBool:
		_val = p.parseBoolean()
	case itemNumber:
		_val = p.parseNumber()
	case itemNull:
		_val = nil
	case itemLeftBracket:
		_val = p.parseArray()
	case itemLeftBrace:
		_val = p.parseJSON(context)
	default:
		p.errorf("Invalid field value for %s", context)
	}
	return _field, _val
}

// increment/decrement value parser
func (p *Parser) parseIncrDecrValue() (string, interface{}) {
	context := "Increment/Decrement Statement"
	// get field
	_next := p.expect(itemString, context)
	_field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field name in %s", context)
	}
	_field = FieldPrefix + _field
	// get operator
	_next = p.expectOneOf(itemPlusEqual, itemMinusEqual, context)
	var _incr bool
	if _next.typ == itemPlusEqual {
		_incr = true
	}
	// get value only number
	if p.peek().typ != itemNumber {
		p.errorf("Invalid field value for %s", context)
	}
	_val := math.Abs(p.parseNumber())
	if !_incr {
		_val *= -1
	}
	return _field, _val
}

// convert file metadata tag to field name to enable queries on meta
// e.g. 'file_name' will be converted to '__header__.name'
func fileMetaToField(meta string) string {
	switch meta {
	case "file_name":
		return HeaderPrefix + "name"
	case "file_mime":
		return BytesPrefix + "mime"
	case "file_size":
		return BytesPrefix + "size"
	case "file_ispublic":
		return HeaderPrefix + "ispublic"
	default:
		return ""
	}
}

// simple where statement parser
func (p *Parser) parseSimpleWhereCondition() map[string]interface{} {
	context := "Where Condition Statement"
	_next := p.next()
	var _field string
	if _next.typ == itemIdentifier {
		_field = fileMetaToField(_next.val)
	} else {
		v, err := formatString(_next.val)
		if err != nil {
			p.errorf("Improperly quoted field value in %s", context)
		}
		_field = FieldPrefix + v
	}

	// parse operator
	switch _next2 := p.next(); _next2.typ {
	case itemEqualTo, itemNotEqualTo, itemGreaterThan, itemGreaterThanEquals, itemLesserThan, itemLesserThanEquals:
		// get left value
		var _val interface{}
		switch p.peek().typ {
		case itemString:
			_val = p.parseString()
		case itemBool:
			_val = p.parseBoolean()
		case itemNumber:
			_val = p.parseNumber()
		case itemNull:
			_val = nil
		case itemLeftBracket:
			_val = p.parseArray()
		case itemLeftBrace:
			_val = p.parseJSON(context)
		default:
			p.errorf("Invalid field value for %s", context)
		}

		// get operetor
		switch _next2.val {
		case "==":
			return map[string]interface{}{_field: _val}
		case "!=":
			return map[string]interface{}{_field: map[string]interface{}{"$ne": _val}}
		case "<":
			return map[string]interface{}{_field: map[string]interface{}{"$lt": _val}}
		case "<=":
			return map[string]interface{}{_field: map[string]interface{}{"$lte": _val}}
		case ">":
			return map[string]interface{}{_field: map[string]interface{}{"$gt": _val}}
		case ">=":
			return map[string]interface{}{_field: map[string]interface{}{"$gte": _val}}
		default:
			p.errorf("Invalid operator for %s", context)
		}
	case itemIdentifier:
		switch strings.ToLower(_next2.val) {
		case "in":
			// expect array
			_val := p.parseArray()
			return map[string]interface{}{_field: map[string]interface{}{"$in": _val}}
		case "nin":
			// expect array
			_val := p.parseArray()
			return map[string]interface{}{_field: map[string]interface{}{"$nin": _val}}
		default:
			p.errorf("Invalid operator for %s", context)
		}
	default:
		p.errorf("Invalid operator for %s", context)
	}

	// shouldnt get here
	return nil
}

// value type where statement parser
func (p *Parser) parseTypeofWhereCondition() map[string]interface{} {
	context := "Where Typeof Condition Statement"
	// absorb typeof
	p.next()
	// left parenthesis
	p.expect(itemLeftParenthesis, context)
	// get field
	_next := p.expect(itemString, context)
	_field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field value in %s", context)
	}
	_field = FieldPrefix + _field
	// right parenthesis
	p.expect(itemRightParenthesis, context)
	// operator == or !=
	_next = p.expectOneOf(itemEqualTo, itemNotEqualTo, context)
	_isequal := false
	if _next.typ == itemEqualTo {
		_isequal = true
	}
	// get type name
	_next = p.expect(itemString, context)
	_typetext, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted Type value in %s", context)
	}
	_typenum := -1
	switch _typetext {
	case "string":
		_typenum = 2 // mongodb specific
	case "int":
		_typenum = 16 // mongodb specific
	}

	if _isequal {
		return map[string]interface{}{_field: map[string]interface{}{"$type": _typenum}}
	}

	return map[string]interface{}{_field: map[string]interface{}{"$not": map[string]interface{}{"$type": _typenum}}}
}

// exists where statement parser
func (p *Parser) parseExistsWhereCondition() map[string]interface{} {
	context := "Where Exists Condition Statement"
	// absorb exists
	p.next()
	// left parenthesis
	p.expect(itemLeftParenthesis, context)
	// get field
	_next := p.expect(itemString, context)
	_field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field value in %s", context)
	}
	_field = FieldPrefix + _field
	// right parenthesis
	p.expect(itemRightParenthesis, context)
	// operator == or !=
	_next = p.expectOneOf(itemEqualTo, itemNotEqualTo, context)
	_isequal := false
	if _next.typ == itemEqualTo {
		_isequal = true
	}
	// get type name
	_next = p.expect(itemBool, context)
	_val := true
	if _next.val == "false" {
		_val = false
	}

	if _isequal {
		if _val == true {
			return map[string]interface{}{_field: map[string]interface{}{"$exists": true}}
		}
		return map[string]interface{}{_field: map[string]interface{}{"$exists": false}}
	} else {
		if _val == true {
			return map[string]interface{}{_field: map[string]interface{}{"$exists": false}}
		}

		return map[string]interface{}{_field: map[string]interface{}{"$exists": true}}
	}

	// shouldn't reach here
	return nil
}

func (p *Parser) parseRegexWhereCondition() map[string]interface{} {
	context := "Where Regex Condition Statement"
	// absorb regex
	p.next()
	// left parenthesis
	p.expect(itemLeftParenthesis, context)
	// get field
	_next := p.next()
	var _field string
	if _next.typ == itemIdentifier {
		_field = fileMetaToField(_next.val)
	} else {
		v, err := formatString(_next.val)
		if err != nil {
			p.errorf("Improperly quoted field value in %s", context)
		}
		_field = FieldPrefix + v
	}
	// absorb comma
	p.expect(itemComma, context)
	// get regex option i.e. 'i' for case insensitive search etc...
	_next = p.expect(itemString, context)
	_opt, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted regex option value in %s", context)
	}
	// right parenthesis
	p.expect(itemRightParenthesis, context)
	// absorb operator ==
	p.expect(itemEqualTo, context)
	// get pattern
	_next = p.expect(itemString, context)
	_pattern_text, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted regex pattern value in %s", context)
	}

	_repattern := bson.RegEx{Pattern: _pattern_text, Options: _opt}
	_rgx := map[string]interface{}{"$regex": _repattern}
	return map[string]interface{}{_field: _rgx}
}

// Where parser mode
const (
	ConditionalOr int = iota
	ConditionalAnd
)

// where statement parser
func (p *Parser) parseWhereCmd() map[string]interface{} {
	context := "Where Statement"
	_where := map[string]interface{}{}
	_and := []map[string]interface{}{}
	_or := []map[string]interface{}{}
	_mode := ConditionalAnd
Loop:
	for {
		// condition statement
		var _condition map[string]interface{}

		switch p.peek().typ {
		case itemString:
			_condition = p.parseSimpleWhereCondition()
		case itemIdentifier:
			// check for function based conditions
			switch strings.ToLower(p.peek().val) {
			case "file_mime": // attachment mime
				fallthrough
			case "file_size": // attachment size
				fallthrough
			case "file_ispublic": // file accessibility
				fallthrough
			case "file_name": // file name
				_condition = p.parseSimpleWhereCondition()
			case "typeof":
				_condition = p.parseTypeofWhereCondition()
				break
			case "exists":
				_condition = p.parseExistsWhereCondition()
				break
			case "regex":
				_condition = p.parseRegexWhereCondition()
				break
			default:
				break Loop
			}
		default:
			break Loop
		}

		if _condition == nil {
			p.errorf("Missing or invalid condition statement in %s", context)
		}

		// do condition mode logic here to decide if it should be added to Or/And list
		_next := p.peek()
		if _next.typ == itemIdentifier && strings.ToLower(_next.val) == "or" {
			// absorb 'or'
			p.next()
			// update mode
			if _mode == ConditionalAnd {
				_mode = ConditionalOr
			}
			// add condition to 'or list'
			_or = append(_or, _condition)
		} else {
			if _mode == ConditionalOr {
				_or = append(_or, _condition)
			} else {
				_and = append(_and, _condition)
			}
			// update mode
			_mode = ConditionalAnd
		}
	}
	// check for empty where statement
	if len(_or) == 0 && len(_and) == 0 {
		p.errorf("Invalid syntax for %s", context)
	}

	// add conditions to where
	_where["and"] = _and
	_where["or"] = _or
	return _where
}
