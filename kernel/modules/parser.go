package modules

import (
	//"os"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"encoding/json"
	"unicode/utf8"
	"labix.org/v2/mgo/bson"
)

// Parser scans job.Request.Script for commands and adds them to job.CommandQueue.
type Parser struct {
	queue *List
	
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
	return &Parser{}
}

// errorf formats the error and terminates processing.
func (p *Parser) errorf(format string, args ...interface{}) {
	p.queue = nil
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
	p.queue = &List{}
	p.lex = lex
}

// stopParse terminates parsing.
func (p *Parser) stopParse() {
	p.lex = nil
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

func (p *Parser) Parse(s string) (q *List, err error) {
	defer p.recover(&err)
	p.startParse(lex(s))
	p.parse()
	p.stopParse()
	return p.queue, nil
}

// It runs to EOF.
// all commands and sub-commands are made case insensitive with strings.ToLower()
func (p *Parser) parse() map[string]interface{} {
	for p.peek().typ != itemEOF {
		switch _next := p.peek(); {
			case _next.typ == itemError:
				p.errorf("Parsing error: %s",p.peek().val)
			case _next.typ == itemIdentifier:
				switch strings.ToLower(_next.val) {
					case "server":
						context := "Server Command"
						// absorb keyword server
						p.next()
						p.expect(itemDot, context)
						item := p.expect(itemIdentifier, context)
						switch strings.ToLower(item.val) {
							case "listdb":
								p.parseListDatabasesCmd()
							case "newdb":
								p.parseNewDatabaseCmd()
							case "newuser":
								p.parseNewUserCmd()
							case "listuser":
								p.parseListUsersCmd()
							case "userinfo":
								p.parseUserInfoCmd()							
							case "dropuser":
								p.parseDropUserCmd()
							case "newpass":
								p.parseNewPasswordCmd()
							case "sysaccess":
								p.parseUserSystemAccessCmd()
							case "userdb":
								p.parseUserDatabaseAccessCmd()
							case "init":
								p.parseServerInitCmd()
							case "dropdb":
								p.parseDropDatabaseCmd()
							default:
								p.errorf("Invalid %s", context)
						}
					case "whoami":
                        p.parseWhoamiCmd()
                    default:
						p.next()
				}
			case _next.typ == itemDatabase:
				context := "Database command"
				db := p.next().val
				p.expect(itemDot, context)
				item := p.expect(itemIdentifier, context)
				switch strings.ToLower(item.val) {
					case "newdir":
						p.parseNewDirectoryCmd(db)
					case "newfile":
						p.parseNewFileCmd(db)
					case "listdir":
						p.parseListDirectoryCmd(db)
					case "rename":
						p.parseRenameContentCmd(db)
					case "move":
						p.parseMoveContentCmd(db)
					case "copy":
						p.parseCopyContentCmd(db)
					case "delete":
						p.parseDeleteContentCmd(db)
					case "info":
						p.parseContentInfoCmd(db)
					case "makepublic":
						p.parseMakeContentPublicCmd(db)
					case "makeprivate":
						p.parseMakeContentPrivateCmd(db)
					case "readfile":
						p.parseReadFileCmd(db)
					case "modfile":
						p.parseModifyFileCmd(db)
					case "deletebinary":
						p.parseDeleteAttachmentCmd(db)
					case "counter":
						p.parseCounterCmd(db)
					case "select":
						p.parseSelectCmd(db)
					case "set":
						p.parseSetCmd(db)
					case "unset":
						p.parseUnsetCmd(db)
					default:
						p.errorf("Invalid %s", context)
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
// {"__header__":{...}, "__attch__":{...}, "content": {...}}
// hence mongodb queries would require the 'content.' prefix
const FIELD_PREFIX string = "content."
const HEADER_PREFIX string = "__header__."
const ATTACH_PREFIX string = "__attch__."

func (p *Parser) parseSaveResult() string {
	if p.peek().typ == itemSaveTo {
		// absorb symbol
		p.next()
		_nxt := p.expect(itemIdentifier, "Result assignment")
		return _nxt.val
	}
	return ""
}

func (p *Parser) parseEndofCommand(ctx string) string {
	saveto := p.parseSaveResult()
	_nxt := p.expectOneOf(itemSemiColon, itemEOF, ctx)
    if _nxt.typ == itemEOF {
        p.backup()
    }
    return saveto
}

// example:
//    whoami ;
//
func (p *Parser) parseWhoamiCmd() {
	context := "Whoami Command"
	// absorb keyword whoami
    p.next();
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("whoami")
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//    server.listdb `^\w` ;
//    server.listdb ;
//
func (p *Parser) parseListDatabasesCmd() {
	context := "List Databases"
	// check if regex filter has been added
	_regex := "."    
	if p.peek().typ == itemRegex {
		_token := p.next()
		_r, err := formatString(_token.val)
		_regex = _r
		if err != nil {
	    	p.errorf("Improperly quoted regex in %s", context)
	    }
	}
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("listdb")
    cmd.Args["regex"] = _regex
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.newdb "test" ;
//
func (p *Parser) parseNewDatabaseCmd() {
	context := "New Database"
    _token := p.expect(itemString, context)
    _db, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted database name in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("newdb")
    cmd.Args["db"] = _db
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.dropdb "test" ;
//
func (p *Parser) parseDropDatabaseCmd() {
	context := "Delete Database"
    _token := p.expect(itemString, context)
    _db, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted database name in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("dropdb")
    cmd.Args["db"] = _db
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.newuser "user" "password" ;
//
func (p *Parser) parseNewUserCmd() {
	context := "New User"
    _token := p.expect(itemString, context)
    _user, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted username in %s", context)
    }
    _token = p.expect(itemString, context)
    _pw, err2 := formatString(_token.val)
    if err2 != nil {
    	p.errorf("Improperly quoted password in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("newuser")
    cmd.Args["username"] = _user
    cmd.Args["password"] = _pw
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.listuser ;
//     server.listuser `^\w` ;
//
func (p *Parser) parseListUsersCmd() {
	context := "List Users"
	// check if regex filter has been added
	_regex := "."    
	if p.peek().typ == itemRegex {
		_token := p.next()
		_r, err := formatString(_token.val)
		_regex = _r
		if err != nil {
	    	p.errorf("Improperly quoted regex in %s", context)
	    }
	}
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("listuser")
    cmd.Args["regex"] = _regex
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.userinfo "user1" ;
//
func (p *Parser) parseUserInfoCmd() {
	context := "User Info"
    _token := p.expect(itemString, context)
    _user, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted username in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("userinfo")
    cmd.Args["username"] = _user
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.dropuser "user1" ;
//
func (p *Parser) parseDropUserCmd() {
	context := "Drop User"
    _token := p.expect(itemString, context)
    _user, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted username in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("dropuser")
    cmd.Args["username"] = _user
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.newpass "user1" "newpassword" ;
//
func (p *Parser) parseNewPasswordCmd() {
	context := "New Password"
    _token := p.expect(itemString, context)
    _user, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted username in %s", context)
    }
    _token = p.expect(itemString, context)
    _pw, err2 := formatString(_token.val)
    if err2 != nil {
    	p.errorf("Improperly quoted password in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("newpass")
    cmd.Args["username"] = _user
    cmd.Args["password"] = _pw
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.sysaccess "user" grant ;
//     server.sysaccess "user" deny ;
//
func (p *Parser) parseUserSystemAccessCmd() {
	context := "User System Access"
    _token := p.expect(itemString, context)
    _user, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted username in %s", context)
    }
    _token = p.expect(itemIdentifier, context)
    _grant := false
    switch _token.val {
    	case "grant":
    		_grant = true
    	case "deny":
    		// do nothing _grant already false
    		break
    	default:
    		p.errorf("Invalid indentifier " + _token.val + " in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("sysaccess")
    cmd.Args["username"] = _user
    cmd.Args["grant"] = _grant
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.userdb "user" "testdb" grant ;
//     server.userdb "user" "testdb" deny ;
//
func (p *Parser) parseUserDatabaseAccessCmd() {
	context := "User Database Access"
    _token := p.expect(itemString, context)
    _user, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted username in %s", context)
    }
    _token = p.expect(itemString, context)
    _db, err2 := formatString(_token.val)
    if err2 != nil {
    	p.errorf("Improperly quoted database in %s", context)
    }
    _token = p.expect(itemIdentifier, context)
    _grant := false
    switch _token.val {
    	case "grant":
    		_grant = true
    	case "deny":
    		// do nothing _grant already false
    		break
    	default:
    		p.errorf("Invalid indentifier " + _token.val + " in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("userdb")
    cmd.Args["username"] = _user
    cmd.Args["database"] = _db
    cmd.Args["grant"] = _grant
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     server.init ;
//
func (p *Parser) parseServerInitCmd() {
    context := "Server Init"
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("initserver")
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.newdir /tmp ;
//     @testdb.newdir /var/www_1/css ;
//
func (p *Parser) parseNewDirectoryCmd(db string) {
	context := "Make Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("newdir")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.newfile /tmp/test {} ;
//     @testdb.newfile /var/www/index.html {"title":"welcome","body":"Hello World!"} ;
//
func (p *Parser) parseNewFileCmd(db string) {
    context := "Make File"
    _token := p.expect(itemPath, context)
    _path := _token.val

    // check if next item is a json object
    var _json interface{}
    if p.peek().typ == itemLeftBrace {
    	_json = p.parseJSON(context)
    } else {
    	p.errorf("Expecting a JSON object in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("newfile")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["data"] = _json
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.listdir / ;
//     @testdb.listdir /users `^\w` ;
//
func (p *Parser) parseListDirectoryCmd(db string) {
	context := "List Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    _regex := "."
	if p.peek().typ == itemRegex {
		_token := p.next()
		_r, err := formatString(_token.val)
		_regex = _r
		if err != nil {
	    	p.errorf("Improperly quoted regex in %s", context)
	    }
	}
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("listdir")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["regex"] = _regex
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.rename /tmp/test "testing" ;
//
func (p *Parser) parseRenameContentCmd(db string) {
	context := "Rename File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    _token = p.expect(itemString, context)
    _name, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted new name in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("newdir")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["name"] = _name
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.move /var/www/project1 /tmp ;
//     @testdb.move /var/www/project1 /var/www "project2" ;
//
func (p *Parser) parseMoveContentCmd(db string) {
	context := "Move File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    _token = p.expect(itemPath, context)
    _path2 := _token.val
    _rename := ""
    if p.peek().typ == itemString {
    	_nxt := p.next()
    	_rename = _nxt.val
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("move")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["to"] = _path2
    cmd.Args["rename"] = _rename
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//      @testdb.copy /var/www/site /var/www/deployed ;
//      @testdb.copy /index.html /var/www/site ;
//      @testdb.copy /var/www/site /var/www "site2" ;
//
func (p *Parser) parseCopyContentCmd(db string) {
	context := "Copy File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    _token = p.expect(itemPath, context)
    _path2 := _token.val
    _rename := ""
    if p.peek().typ == itemString {
    	_nxt := p.next()
    	_rename = _nxt.val
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("copy")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["to"] = _path2
    cmd.Args["rename"] = _rename
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.delete /tmp/testing ;
//
func (p *Parser) parseDeleteContentCmd(db string) {
	context := "Delete File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("delete")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.info /index.html ;
//     @testdb.info /var/www ;
//
func (p *Parser) parseContentInfoCmd(db string) {
	context := "Info File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("info")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.makepublic /var/www/site1 ;
//     @testdb.makepublic /var/www/index.html ;
//
func (p *Parser) parseMakeContentPublicCmd(db string) {
	context := "Make Public File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("makepublic")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.makeprivate /var/www/site1 ;
//     @testdb.makeprivate /var/www/index.html ;
//
func (p *Parser) parseMakeContentPrivateCmd(db string) {
	context := "Make Private File/Directory"
    _token := p.expect(itemPath, context)
    _path := _token.val
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("makeprivate")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.readfile /etc/nginx.conf ;
//     @testdb.readfile /var/www/index.html ["title","body"] ;
//
func (p *Parser) parseReadFileCmd(db string) {
	context := "Read File"
    _token := p.expect(itemPath, context)
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
						p.errorf("Improperly quoted string value in %s Array definition.", context)
					}
					_list = append(_list, _val)
					continue
				case itemComma:
					// absorb comma
					p.next()
					if p.peek().typ == itemRightBracket {
						p.errorf("Trailing comma ',' in %s Array definition.", context)
					}
					continue
				case itemRightBracket:
					// absorb
					p.next()
					break Loop
				default:
					p.errorf("Invalid value: " + p.peek().val + " in %s Array definition.", context)
			}
		}
    }

    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("readfile")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["fields"] = _list
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.modfile /index.html {"title":"hello world","body":""} ;
//
func (p *Parser) parseModifyFileCmd(db string) {
    context := "Modify File"
    _token := p.expect(itemPath, context)
    _path := _token.val

    // check if next item is a json object
    var _json interface{}
    if p.peek().typ == itemLeftBrace {
    	_json = p.parseJSON(context)
    } else {
    	p.errorf("Expecting a JSON object in %s", context)
    }
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("modfile")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Args["data"] = _json
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.deletebinary /var/www/site1/images/img1.png ;
//
func (p *Parser) parseDeleteAttachmentCmd(db string) {
	context := "Delete File Binary Content"
    _token := p.expect(itemPath, context)
    _path := _token.val
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("deletebinary")
    cmd.Database = db
    cmd.Args["path"] = _path
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.counter "counter1" incr 1 ;
//     @testdb.counter "course.users" reset 0 ;
//     @testdb.counter "temperature" decr 10 ;
//     @testdb.counter list ;
//
func (p *Parser) parseCounterCmd(db string) {
	context := "Counter Statement"
    _token := p.expectOneOf(itemString, itemIdentifier, context)
    if _token.typ == itemIdentifier {
        if _token.val == "list" {
        	cmd := NewCommand("counter")
            cmd.Database = db
            cmd.Args["action"] = "list"
            cmd.Args["regex"] = "." //match everything by default

        	if p.peek().typ == itemRegex {
        		// get pattern
				_next := p.next()
				_pattern_text, err := formatString(_next.val)
				if err != nil {
					p.errorf("Improperly quoted regex pattern value in %s", context)
				}
				cmd.Args["regex"] = _pattern_text
        	}
            saveto := p.parseEndofCommand(context)
            cmd.Store = saveto            
            p.queue.RPush(cmd)            
            return

        } else {
            p.errorf("Invalid identifier %s in %s",_token.val, context)
        }        
    }

    _counter, err := formatString(_token.val)
    if err != nil {
    	p.errorf("Improperly quoted counter name in %s", context)
    }
    _token = p.expect(itemIdentifier, context)
    _action := ""
    switch _token.val {
    	case "incr":
    		fallthrough
    	case "decr":
    		fallthrough
    	case "reset":
    		_action = _token.val
    	default:
    		p.errorf("Invalid indentifier " + _token.val + " in %s", context)
    }

    _token = p.expect(itemNumber, context)
    _val, err := strconv.ParseInt(_token.val, 10, 64) // base 10 64bit integer
    if err != nil {
        p.errorf("Invalid numerical value in %s", context)
    }
    
    saveto := p.parseEndofCommand(context)
    cmd := NewCommand("counter")
    cmd.Database = db
    cmd.Args["name"] = _counter
    cmd.Args["action"] = _action
    cmd.Args["value"] = _val
    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.select "name.first" "country"
//     in /tmp/users /tmp2/users
//     where "name.last" == "Wilson"
//     "age" > 52 or "children.count" <= 20
//     sort asc "name" ;
//
//     @testdb.select "name" in /tmp/sysusers
//     limit 10;
//
func (p *Parser) parseSelectCmd(db string) {
    const context = "Select Statement"
    _fields := []string{}
    // get fields
    for p.peek().typ == itemString {
        _token := p.next()
        _field, err := formatString(_token.val)
	    if err != nil {
	    	p.errorf("Improperly quoted field name in %s", context)
	    }
	    _fields = append(_fields, FIELD_PREFIX + _field)
        continue
    }
    // get directories
    _in := p.expect(itemIdentifier, context)
    if strings.ToLower(_in.val) != "in" {
    	p.errorf("Invalid %s, expecting 'In statement'.", context)
    }
    _paths := []string{}
    for p.peek().typ == itemPath {
        _path := p.next().val
        _paths = append(_paths, _path)
        continue
    }
    cmd := NewCommand("select")
    cmd.Database = db
    cmd.Args["fields"] = _fields
    cmd.Args["dirs"] = _paths
    var saveto string
    
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
				case _token.typ == itemSaveTo:
					p.backup()
					saveto = p.parseEndofCommand(context)
					break Loop
    			case _token.typ == itemSemiColon:
    				break Loop
    			case _token.typ == itemEOF:
    				p.backup()
    				break Loop
    			default:
    				p.errorf("Invalid identifier "+_token.val+" in %s", context)
    		}
    	}

    // validate select statement
    _, hascount := cmd.Args["count"]
    _, haslimit := cmd.Args["limit"]
    _, hassort := cmd.Args["sort"]    
    _, hasdistinct := cmd.Args["distinct"]
    
    if haslimit || hassort {
    	if hascount {
    		p.errorf("'Count' cannot be used with 'Limit' or 'Sort' in %s", context)
    	}
    	if hasdistinct {
    		p.errorf("'Distinct' cannot be used with 'Limit' or 'Sort' in %s", context)
    	}
    } else if hasdistinct {
    	if haslimit {
    		p.errorf("'Limit' cannot be used with 'Distinct' in %s", context)
    	}
    	if hassort {
    		p.errorf("'Sort' cannot be used with 'Distinct' in %s", context)
    	}
    	if hascount {
    		p.errorf("'Count' cannot be used with 'Distinct' in %s", context)
    	}
    } else if hascount {
    	if haslimit {
    		p.errorf("'Limit' cannot be used with 'Count' in %s", context)
    	}
    	if hassort {
    		p.errorf("'Sort' cannot be used with 'Count' in %s", context)
    	}
    	if hasdistinct {
    		p.errorf("'Distinct' cannot be used with 'Count' in %s", context)
    	}
    }

    cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.set "user.age" = 50
//     "user.colors" = ["blue","white"]
//     in /tmp/users
//     where regex("user.name","i") == `^j` ;
//
func (p *Parser) parseSetCmd(db string) {
    const context = "Set Statement"
    _fields := map[string]interface{} {}
    // get field assignment list
    Loop:
	for {
		switch p.peek().typ {
			case itemString:
				f, v := p.parseValueAssignment()
				_fields[f] = v
				continue
			default:
				break Loop
		}
	}
	if len(_fields) < 1 {
		p.errorf("Invalid %s: no field assignments found", context)
	}

    // get directories
    _in := p.expect(itemIdentifier, context)
    if strings.ToLower(_in.val) != "in" {
    	p.errorf("Invalid %s, expecting 'In statement'.", context)
    }
    _paths := []string{}
    for p.peek().typ == itemPath {
        _path := p.next().val
        _paths = append(_paths, _path)
        continue
    }
    cmd := NewCommand("set")
    cmd.Database = db
    cmd.Args["fields"] = _fields
    cmd.Args["dirs"] = _paths
    var saveto string
    
    // get optional identifiers
    Loop2:
	for {
		switch _token := p.next(); {
			case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "where":
				_where := p.parseWhereCmd()
				cmd.Args["where"] = _where
				continue
			case _token.typ == itemSaveTo:
					p.backup()
					saveto = p.parseEndofCommand(context)
					break Loop2   			
			case _token.typ == itemSemiColon:
				break Loop2
			case _token.typ == itemEOF:
				// do not consume eof
				p.backup()
				break Loop2
			default:
				p.errorf("Invalid identifier "+_token.val+" in %s", context)
		}
	}

	cmd.Store = saveto
    p.queue.RPush(cmd)
}

// example:
//     @testdb.unset "user" "courses"
//     in /tmp/users where "user.year" == "2nd";
//
func (p *Parser) parseUnsetCmd(db string) {
    const context = "Unset Statement"
    _fields := map[string]interface{} {}
    // get fields
    for p.peek().typ == itemString {
        _token := p.next()
        _field, err := formatString(_token.val)
	    if err != nil {
	    	p.errorf("Improperly quoted field name in %s", context)
	    }
	    _field = FIELD_PREFIX + _field
	    _fields[_field] = 1
        continue
    }
	if len(_fields) < 1 {
		p.errorf("Invalid %s: no fields found", context)
	}

    // get directories
    _in := p.expect(itemIdentifier, context)
    if strings.ToLower(_in.val) != "in" {
    	p.errorf("Invalid %s, expecting 'In statement'.", context)
    }
    _paths := []string{}
    for p.peek().typ == itemPath {
        _path := p.next().val
        _paths = append(_paths, _path)
        continue
    }
    cmd := NewCommand("unset")
    cmd.Database = db
    cmd.Args["fields"] = _fields
    cmd.Args["dirs"] = _paths
    var saveto string
    
    // get optional identifiers
    Loop2:
	for {
		switch _token := p.next(); {
			case _token.typ == itemIdentifier && strings.ToLower(_token.val) == "where":
				_where := p.parseWhereCmd()
				cmd.Args["where"] = _where
				continue
			case _token.typ == itemSaveTo:
					p.backup()
					saveto = p.parseEndofCommand(context)
					break Loop2    			
			case _token.typ == itemSemiColon:
				break Loop2
			case _token.typ == itemEOF:
				// do not consume eof
				p.backup()
				break Loop2
			default:
				p.errorf("Invalid identifier "+_token.val+" in %s", context)
		}
	}

	cmd.Store = saveto
    p.queue.RPush(cmd)
}

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
		field = prefix + FIELD_PREFIX + field
	    if err != nil {
	    	p.errorf("Improperly quoted field name in %s", context)
	    }				        
        _sort = append(_sort, field)
        continue
    }
    return _sort
}

// example:
//     @testdb.select ...
//     limit 10 ;
//
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

// example:
//     @testdb.select ...
//     distinct "user.country" ;
//
func (p *Parser) parseDistinctCmd() string {
	context := "Select Distinct Statement"
	_next := p.expect(itemString, context)
	field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field name in %s", context)
	}
	field = FIELD_PREFIX + field
    return field
}

// example:
//     @testdb.select ...
//     where "user.colors" in ["blue","red"]
//     or "user.subscription.id" in [1,109,50] ;
//
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
					p.errorf("Invalid value: " + p.peek().val + " in %s", context)
			}
		}
	return _list
}

// example:
//     @testdb.set "assignment" = "completed"
//     "status" = "pending" in /tmp/users
//     where "time.elapsed" > 3600 ;
//
func (p *Parser) parseString() string {
	context := "String Definition"
	_next := p.next()
	_val, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted string value in %s", context)
	}
	return _val
}

// example:
//     @testdb.select ...
//     where "cars" > 2 or "average_score" > 10.5 ;
//
func (p *Parser) parseNumber() float64 {
	context := "Number Definition"
	_next := p.next()
	_val, err := strconv.ParseFloat(_next.val, 64)
	if err != nil {
		p.errorf("Invalid numerical value in %s", context)
	}
	return _val
}

// example:
//     @testdb.select ...
//     where "isactive" == true or "isfieldagent" == false ;
//
func (p *Parser) parseBoolean() bool {
	_next := p.next()
	if _next.val == "false" {
		return false
	}
	return true
}

// example:
//     @testdb.set "user.name" = {"first":"jason","last":"bourne"}
//     in /tmp/users ... ;
//
func (p *Parser) parseJSON(context string) map[string]interface{} {
	// check if next item is a json object
    _objlevel := 0
    _json := ""
    Loop:
    	for {
    		switch _next := p.next(); {
    			case _next.typ == itemError:
					p.errorf("Parsing error: %s",_next.val)
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

// example:
//     @testdb.set "user.dob" = "13/06/1965" "user.country" = "usa"
//     in /tmp/users ... ;
//
func (p *Parser) parseValueAssignment() (string, interface{}) {
	context := "Assignment Statement"
	// get field
	_next := p.expect(itemString, context)
	_field, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted field name in %s", context)
	}
	_field = FIELD_PREFIX + _field
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

// convert file metadata tag to field name to enable queries on meta
// e.g. 'file_name' will be converted to '__header__.name'
func fileMetaToField(meta string) string {
	switch meta {
		case "file_name":
			return HEADER_PREFIX + "name"
		case "file_mime":
			return ATTACH_PREFIX + "mime"
		case "file_size":
			return ATTACH_PREFIX + "size"
		case "file_ispublic":
			return HEADER_PREFIX + "ispublic"
		default:
			return ""
	}
}

// example:
//     @testdb.select "user.name" in /tmp/users
//     where "user.name" == "luke" "user.surname" = "skywalker" ;
//
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
		_field = FIELD_PREFIX + v
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
					return map[string]interface{}{_field:_val}
				case "!=":
					return map[string]interface{}{_field:map[string]interface{}{"$neq":_val}}
				case "<":
					return map[string]interface{}{_field:map[string]interface{}{"$lt":_val}}
				case "<=":
					return map[string]interface{}{_field:map[string]interface{}{"$lte":_val}}
				case ">":
					return map[string]interface{}{_field:map[string]interface{}{"$gt":_val}}
				case ">=":
					return map[string]interface{}{_field:map[string]interface{}{"$gte":_val}}
				default:
					p.errorf("Invalid operator for %s", context)
			}
		case itemIdentifier:
			switch strings.ToLower(_next2.val) {
				case "in":
					// expect array
					_val := p.parseArray()
					return map[string]interface{}{_field:map[string]interface{}{"$in":_val}}
				case "nin":
					// expect array
					_val := p.parseArray()
					return map[string]interface{}{_field:map[string]interface{}{"$nin":_val}}
				default:
					p.errorf("Invalid operator for %s", context)
			}
		default:
			p.errorf("Invalid operator for %s", context)
	}

	// shouldnt get here
	return nil
}

// example:
//     @testdb.select "user.name" in /tmp/users
//     where typeof("user.age") == "int" ;
//
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
	_field = FIELD_PREFIX + _field
	// right parenthesis
	p.expect(itemRightParenthesis, context)
	// operator == or !=
	_next = p.expectOneOf(itemEqualTo, itemNotEqualTo, context)
	_isequal := false
	if _next.typ == itemEqualTo { _isequal = true }
	// get type name
	_next = p.expect(itemString, context)
	_typetext, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted Type value in %s", context)
	}
	_typenum := -1
	switch _typetext {
		case "string":
			_typenum = 2
		case "int":
			_typenum = 16
	}

	if _isequal {
		return map[string]interface{}{_field:map[string]interface{}{"$type":_typenum}}
	}

	return map[string]interface{}{_field:map[string]interface{}{"$not":map[string]interface{}{"$type":_typenum}}}
}

// example:
//     @testdb.select "user.name" in /tmp/users
//     where exists("user.age") == true ;
//
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
	_field = FIELD_PREFIX + _field
	// right parenthesis
	p.expect(itemRightParenthesis, context)
	// operator == or !=
	_next = p.expectOneOf(itemEqualTo, itemNotEqualTo, context)
	_isequal := false
	if _next.typ == itemEqualTo { _isequal = true }
	// get type name
	_next = p.expect(itemBool, context)
	_val := true
	if _next.val == "false" {
		_val = false
	}

	if _isequal {
		if _val == true {
			return map[string]interface{}{_field:map[string]interface{}{"$exists":true}}
		}
		return map[string]interface{}{_field:map[string]interface{}{"$exists":false}}
	} else {
		if _val == true {
			return map[string]interface{}{_field:map[string]interface{}{"$exists":false}}
		}

		return map[string]interface{}{_field:map[string]interface{}{"$exists":true}}
	}

	// shouldn't reach here
	return nil
}

// regex are delimited by 'ticks': `[regex]`
//
// example:
//     @testdb.select "user.name" in /tmp/users
//     where regex("user.name") == `^\w` ;
//
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
		_field = FIELD_PREFIX + v
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
	_next = p.expect(itemRegex, context)
	_pattern_text, err := formatString(_next.val)
	if err != nil {
		p.errorf("Improperly quoted regex pattern value in %s", context)
	}	
	
	_repattern := bson.RegEx{Pattern:_pattern_text, Options:_opt}
	_rgx := map[string]interface{}{"$regex":_repattern}
	return map[string]interface{}{_field:_rgx}	
}

// Where parser mode
const (
	ConditionalOr int = iota
	ConditionalAnd
)

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