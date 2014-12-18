package base

import (
	"strconv"
)

type argType string

const (
	argString argType = "string"
	argBool   argType = "bool"
	argInt    argType = "int"
	argFloat  argType = "float"
)

type argument struct {
	Name  string
	Type  argType
	Value interface{}
}

type argList struct {
	params map[string]*argument
}

func newArgList() argList {
	ac := argList{}
	ac.params = map[string]*argument{}
	return ac
}

func (ac argList) Add(name string, typ argType) {
	ap := argument{Name: name, Type: typ}
	ac.params[name] = &ap
}

func (ac argList) Get(name string) interface{} {
	arg, ok := ac.params[name]
	if !ok {
		return nil
	}
	return arg.Value
}

// argument parser
func (p *Parser) parseArgs(ctx string, ac argList) {
	for {
		next := p.peek()
		if next.typ != itemArgument {
			break
		}
		// get argument name
		p.next() // absorb argument symbol: --
		t := p.expect(itemIdentifier, ctx)
		arg := t.val
		// check if argument is valid and get type
		ap, ok := ac.params[arg]
		if !ok {
			p.errorf("Unknown argument %s in %s", arg, ctx)
		}
		switch ap.Type {
		case argString:
			ap.Value = p.parseStringArg(ctx, ap.Name)
		case argBool:
			ap.Value = p.parseBooleanArg(ctx, ap.Name)
		case argInt:
			ap.Value = p.parseIntArg(ctx, ap.Name)
		case argFloat:
			ap.Value = p.parseFloatArg(ctx, ap.Name)
		default:
			// shouldn't reach here
			p.errorf("Unknown argument type in %s", arg, ctx)
		}
	}
}

// string argument parser
func (p *Parser) parseStringArg(ctx, name string) string {
	// parse equal
	p.expect(itemEqual, ctx)

	// get string value
	optval := p.expect(itemString, ctx)
	val, err := formatString(optval.val)
	if err != nil {
		p.errorf("Improperly quoted string value for argument %s in %s", name, ctx)
	}
	return val
}

// boolean argument parser
func (p *Parser) parseBooleanArg(ctx, name string) bool {
	return true
}

// general number argument parser
func (p *Parser) parseNumberArg(ctx, name string) string {
	// parse equal
	p.expect(itemEqual, ctx)

	// get string value of number
	optval := p.expect(itemNumber, ctx)
	return optval.val
}

// int argument parser
func (p *Parser) parseIntArg(ctx, name string) int64 {
	s := p.parseNumberArg(ctx, name)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		p.errorf("Invalid integer value for argument %s in %s", name, ctx)
	}
	return n
}

// float argument parser
func (p *Parser) parseFloatArg(ctx, name string) float64 {
	s := p.parseNumberArg(ctx, name)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		p.errorf("Invalid float value for argument %s in %s", name, ctx)
	}
	return n
}
