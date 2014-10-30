package dsl

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// item represents a token returned from the scanner

type item struct {
	typ itemType
	val string
}

// item.String => string representation of item

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 30:
		return fmt.Sprintf("%.30q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

// itemType identifies the type of the lexemes
type itemType int

const (
	itemError    itemType = iota // error occurred; value is text of error
	itemDatabase                 // database constant
	itemBool                     // boolean constant
	itemEOF
	itemEqual             // equal for argument value assignment
	itemPlusEqual         // += for value increment
	itemMinusEqual        // -= for value decrement
	itemColon             // :
	itemDot               // .
	itemSemiColon         // ;
	itemComma             // ,
	itemSendTo            // >>
	itemLeftBrace         // {
	itemRightBrace        // }
	itemLeftParenthesis   // (
	itemRightParenthesis  // )
	itemLeftBracket       // [
	itemRightBracket      // ]
	itemEqualTo           // ==
	itemNotEqualTo        // !=
	itemGreaterThan       // >
	itemGreaterThanEquals // >=
	itemLesserThan        // <
	itemLesserThanEquals  // <=
	itemIdentifier        // alphanumeric identifier
	itemNumber            // simple number
	itemString            // quoted string (includes quotes)
	itemPath              // unix type file path
	itemArgument          // --argument
	// Keywords appear after all the rest.
	itemKeyword // used only to delimit the keywords
	itemNull
)

// Make the types prettyprint.
var itemName = map[itemType]string{
	itemError:             "error",
	itemBool:              "bool",
	itemDatabase:          "database",
	itemEqual:             "=",
	itemPlusEqual:         "+=",
	itemMinusEqual:        "-=",
	itemColon:             ":",
	itemDot:               ".",
	itemSemiColon:         ";",
	itemComma:             ",",
	itemSendTo:            ">>",
	itemLeftBrace:         "{",
	itemRightBrace:        "}",
	itemLeftParenthesis:   "(",
	itemRightParenthesis:  ")",
	itemLeftBracket:       "[",
	itemRightBracket:      "]",
	itemEqualTo:           "==",
	itemNotEqualTo:        "!=",
	itemGreaterThan:       ">",
	itemGreaterThanEquals: ">=",
	itemLesserThan:        "<",
	itemLesserThanEquals:  "<=",
	itemEOF:               "EOF",
	itemIdentifier:        "identifier",
	itemNumber:            "number",
	itemString:            "string",
	itemPath:              "path",
	itemArgument:          "--",
	itemNull:              "null",
}

// itemType.String => string representation of itemType
func (i itemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

const eof = -1

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	input string    // the string being scanned.
	state stateFn   // the next lexing function to enter.
	pos   int       // current position in the input.
	start int       // start position of this item.
	width int       // width of last rune read from input.
	items chan item // channel of scanned items.
}

// next returns the next rune in the input.
func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.pos], "\n")
}

// error returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, fmt.Sprintf(format, args...)}
	return nil
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
	panic("not reached")
}

// lex creates a new scanner for the input string.
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		state: lexInsideScript,
		items: make(chan item, 2), // Two items of buffering is sufficient for all state functions
	}
	return l
}

// state functions
const (
	leftComment  = "/*"
	rightComment = "*/"
)

// lexComment scans a comment. The left comment marker is known to be present.
func lexComment(l *lexer) stateFn {
	l.pos += len(leftComment)
	i := strings.Index(l.input[l.pos:], rightComment)
	if i < 0 {
		return l.errorf("unclosed comment")
	}
	l.pos += i + len(rightComment)
	l.ignore()
	return lexInsideScript
}

// lexEnvironmentSetting scans interpreter directives
func lexDatabase(l *lexer) stateFn {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	specials := "_"
	if !l.accept(letters) {
		return l.errorf("database name must start with a letter")
	}
	l.acceptRun(letters + digits + specials)
	l.emit(itemDatabase)
	return lexInsideScript
}

//
func lexInsideScript(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		break
	case isSpace(r):
		l.ignore()
		return lexInsideScript
	case r == '@':
		// ignore '@' in front and database name
		l.ignore()
		return lexDatabase
	case r == '/':
		if l.peek() == '*' {
			return lexComment
		}
		return lexPath
	case r == '<':
		if l.next() == '=' {
			l.emit(itemLesserThanEquals)
			return lexInsideScript
		}
		l.backup()
		l.emit(itemLesserThan)
		return lexInsideScript
	case r == '>':
		switch nxt := l.next(); {
		case nxt == '=':
			l.emit(itemGreaterThanEquals)
			break
		case nxt == '>':
			l.emit(itemSendTo)
			break
		default:
			l.backup()
			l.emit(itemGreaterThan)
		}
		return lexInsideScript
	case r == '!':
		if l.next() == '=' {
			l.emit(itemNotEqualTo)
			return lexInsideScript
		}
		return l.errorf("expected !=")
	case r == '=':
		if l.peek() == '=' {
			// absorb
			l.next()
			l.emit(itemEqualTo)
			return lexInsideScript
		}
		l.emit(itemEqual)
		return lexInsideScript
	case r == ';':
		l.emit(itemSemiColon)
		return lexInsideScript
	case r == '.':
		l.emit(itemDot)
		return lexInsideScript
	case r == ':':
		l.emit(itemColon)
		return lexInsideScript
	case r == ',':
		l.emit(itemComma)
		return lexInsideScript
	case r == '(':
		l.emit(itemLeftParenthesis)
		return lexInsideScript
	case r == '[':
		l.emit(itemLeftBracket)
		return lexInsideScript
	case r == '{':
		l.emit(itemLeftBrace)
		return lexInsideScript
	case r == ')':
		l.emit(itemRightParenthesis)
		return lexInsideScript
	case r == ']':
		l.emit(itemRightBracket)
		return lexInsideScript
	case r == '}':
		l.emit(itemRightBrace)
		return lexInsideScript
	case r == '"':
		return lexDoubleQuote
	case r == '\'':
		return lexSingleQuote
	case r == '+':
		if l.peek() == '=' {
			// absorb equal
			l.next()
			l.emit(itemPlusEqual)
			return lexInsideScript
		}

		l.backup()
		return lexNumber
	case r == '-':
		pk := l.peek()
		if pk == '=' {
			// absorb equal
			l.next()
			l.emit(itemMinusEqual)
			return lexInsideScript
		} else if pk == '-' {
			// absorbe minus
			l.next()
			l.emit(itemArgument)
			return lexInsideScript
		}

		l.backup()
		return lexNumber
	case ('0' <= r && r <= '9'):
		l.backup()
		return lexNumber
	case isAlphaNumeric(r):
		l.backup()
		return lexIdentifier
	default:
		return l.errorf("unrecognized character in script: %#U", r)
	}

	// Correctly reached EOF.
	l.emit(itemEOF)
	return nil
}

// lexIdentifier scans an alphanumeric.
func lexIdentifier(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
			break
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			switch id := strings.ToLower(word); {
			case id == "null":
				l.emit(itemNull)
			case id == "true", id == "false":
				l.emit(itemBool)
			default:
				l.emit(itemIdentifier)
			}
			break Loop
		}
	}
	return lexInsideScript
}

// lexDoubleQuote scans a double quoted string.
func lexDoubleQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			break Loop
		}
	}
	l.emit(itemString)
	return lexInsideScript
}

// lexSingleQuote scans a single quoted string.
func lexSingleQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '\'':
			break Loop
		}
	}
	l.emit(itemString)
	return lexInsideScript
}

// lexNumber scans a number: decimal, octal, hex or float.  This
// isn't a perfect number scanner - for instance it accepts "." and "0x0.2"
// and "089" - but when it's wrong the input is invalid and the parser (via
// strconv) will notice.
func lexNumber(l *lexer) stateFn {
	if !l.scanNumber() {
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	l.emit(itemNumber)
	return lexInsideScript
}

func (l *lexer) scanNumber() bool {
	// Optional leading sign.
	l.accept("+-")
	// Is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		l.next()
		return false
	}
	return true
}

func lexPath(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isValidPathCharacter(r), r == '/':
			break
		default:
			l.backup()
			break Loop
		}
	}
	l.emit(itemPath)
	return lexInsideScript
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isValidPathCharacter reports whether r is a valid path character.
func isValidPathCharacter(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-'
}
