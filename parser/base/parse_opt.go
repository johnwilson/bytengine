package base

import (
	"strconv"
)

type optType string

const (
	optString optType = "string"
	optBool   optType = "bool"
	optInt    optType = "int"
	optFloat  optType = "float"
)

type option struct {
	Name  string
	Type  optType
	Value interface{}
}

type optList struct {
	items map[string]*option
}

func newOptList() optList {
	l := optList{}
	l.items = map[string]*option{}
	return l
}

func (l optList) Add(name string, typ optType) {
	opt := option{Name: name, Type: typ}
	l.items[name] = &opt
}

func (l optList) Get(name string) interface{} {
	opt, ok := l.items[name]
	if !ok {
		return nil
	}
	return opt.Value
}

// option parser
func (p *Parser) parseOptions(ctx string, ac optList) {
	for {
		next := p.peek()
		if next.typ != itemOption {
			break
		}
		// get option name
		p.next() // absorb option symbol: --
		t := p.expect(itemIdentifier, ctx)
		name := t.val
		// check if option is valid and get type
		opt, ok := ac.items[name]
		if !ok {
			p.errorf("Unknown option %s in %s", name, ctx)
		}
		switch opt.Type {
		case optString:
			opt.Value = p.parseStringOpt(ctx, opt.Name)
		case optBool:
			opt.Value = p.parseBooleanOpt(ctx, opt.Name)
		case optInt:
			opt.Value = p.parseIntOpt(ctx, opt.Name)
		case optFloat:
			opt.Value = p.parseFloatOpt(ctx, opt.Name)
		default:
			// shouldn't reach here
			p.errorf("Unknown option type in %s", name, ctx)
		}
	}
}

// string option parser
func (p *Parser) parseStringOpt(ctx, name string) string {
	// parse equal
	p.expect(itemEqual, ctx)

	// get string value
	optval := p.expect(itemString, ctx)
	val, err := formatString(optval.val)
	if err != nil {
		p.errorf("Improperly quoted string value for option %s in %s", name, ctx)
	}
	return val
}

// boolean option parser
func (p *Parser) parseBooleanOpt(ctx, name string) bool {
	return true
}

// general number option parser
func (p *Parser) parseNumberOpt(ctx, name string) string {
	// parse equal
	p.expect(itemEqual, ctx)

	// get string value of number
	optval := p.expect(itemNumber, ctx)
	return optval.val
}

// int option parser
func (p *Parser) parseIntOpt(ctx, name string) int64 {
	s := p.parseNumberOpt(ctx, name)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		p.errorf("Invalid integer value for option %s in %s", name, ctx)
	}
	return n
}

// float option parser
func (p *Parser) parseFloatOpt(ctx, name string) float64 {
	s := p.parseNumberOpt(ctx, name)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		p.errorf("Invalid float value for option %s in %s", name, ctx)
	}
	return n
}
