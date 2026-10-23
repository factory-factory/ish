package filter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SyntaxError describes a problem with a filter expression, and where in the
// expression it is, so it can be highlighted.
type SyntaxError struct {
	// Start and End are the positions of the unexpected text, as offsets in
	// runes (not bytes) from the start of the expression. End is exclusive,
	// and equals Start when the problem is unexpected end of input.
	Start, End int

	Message string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("column %d: %s", e.Start+1, e.Message)
}

// attributeChar reports whether r may appear in an attribute name: the same
// characters as metadata keys.
func attributeChar(r rune) bool {
	return r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_./", r))
}

// numberPattern matches a JSON number.
var numberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?`)

// Parse parses a filter expression:
//
//	expression = or
//	or         = and { "or" and }
//	and        = not { "and" not }
//	not        = "not" not | "(" expression ")" | attribute
//	attribute  = "{" ( name | field ) operator value "}"
//	field      = "@title" | "@author" | "@dateCreated"
//	operator   = "=" | "!=" | "<" | ">" | "<=" | ">=" | "~"
//	value      = "null" | "true" | "false" | number | string | regexp | lookup
//	           | "(" comparison ")"
//	lookup     = "." key { "." key }
//
// Expressions, in parentheses, compute a value:
//
//	comparison = sum [ ( "=" | "!=" | "<" | "<=" | ">" | ">=" ) sum ]
//	sum        = product { ( "+" | "-" ) product }
//	product    = power { ( "*" | "/" | "%" ) power }
//	power      = chain [ "**" power ]
//	chain      = unary { ":" name [ "(" comparison { "," comparison } ")" ] }
//	unary      = ( "-" | "!" ) unary | operand
//	operand    = "null" | "true" | "false" | number | string | lookup | list
//	           | "(" comparison ")"
//	list       = "[" [ comparison { "," comparison } ] "]"
//
// Names use letters, digits, and - _ . / , and refer to metadata
// attributes. Fields, which start with @, refer to the issue's title, the
// author of its first comment, and when it was created. Numbers are JSON numbers, and
// strings are JSON strings in single or double quotes; \' is accepted as an
// escape in either. A regexp is a regular expression literal between slashes,
// such as /^in\s/, which is taken literally except that \/ stands for a
// slash. Keywords are lowercase. Tokens may be separated by any amount of
// space. An expression made only of space is an error.
//
// The value of a ~ match must be a regexp or a string that's a valid Go
// regular expression, a regexp can only be used with ~, and null, true, and
// false can only be compared with = and !=.
func Parse(text string) (Node, error) {
	p := &parser{input: []rune(text)}

	p.skipSpace()

	if p.done() {
		return nil, p.errorHere("expected a filter expression")
	}

	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	p.skipSpace()

	if !p.done() {
		start := p.pos
		return nil, p.errorAt(start, p.wordEnd(start), "expected and, or, or the end of the expression")
	}

	return node, nil
}

type parser struct {
	input []rune
	pos   int
}

func (p *parser) done() bool {
	return p.pos >= len(p.input)
}

func (p *parser) peek() rune {
	if p.done() {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) skipSpace() {
	for !p.done() && unicode.IsSpace(p.peek()) {
		p.pos++
	}
}

func (p *parser) errorAt(start, end int, format string, args ...any) *SyntaxError {
	return &SyntaxError{Start: start, End: end, Message: fmt.Sprintf(format, args...)}
}

// errorHere reports a problem with the text at the current position, or
// unexpected end of input.
func (p *parser) errorHere(format string, args ...any) *SyntaxError {
	return p.errorAt(p.pos, p.wordEnd(p.pos), format, args...)
}

// wordEnd returns the end of the run of non-space text starting at start, to
// highlight as unexpected.
func (p *parser) wordEnd(start int) int {
	end := start

	for end < len(p.input) && !unicode.IsSpace(p.input[end]) {
		end++
	}

	return end
}

// keyword reports whether the keyword word comes next, as a whole word, and
// consumes it if so.
func (p *parser) keyword(word string) bool {
	p.skipSpace()

	end := p.pos + utf8.RuneCountInString(word)

	if end > len(p.input) || string(p.input[p.pos:end]) != word {
		return false
	}

	// A keyword followed by a name character is part of a longer word.
	if end < len(p.input) && attributeChar(p.input[end]) {
		return false
	}

	p.pos = end

	return true
}

func (p *parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.keyword("or") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}

		left = &OrExpression{LeftExpression: left, RightExpression: right}
	}

	return left, nil
}

func (p *parser) parseAnd() (Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.keyword("and") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}

		left = &AndExpression{LeftExpression: left, RightExpression: right}
	}

	return left, nil
}

func (p *parser) parseNot() (Node, error) {
	if p.keyword("not") {
		expression, err := p.parseNot()
		if err != nil {
			return nil, err
		}

		return &NotExpression{Expression: expression}, nil
	}

	p.skipSpace()

	switch p.peek() {
	case '(':
		open := p.pos
		p.pos++

		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		p.skipSpace()

		if p.peek() != ')' {
			if p.done() {
				return nil, p.errorAt(open, open+1, "this parenthesis is never closed")
			}

			return nil, p.errorHere("expected and, or, or a closing parenthesis")
		}

		p.pos++
		addParentheses(node)

		return node, nil

	case '{':
		return p.parseAttribute()
	}

	if p.done() {
		return nil, p.errorHere("expected an attribute expression such as {state = 'open'}, not, or (")
	}

	return nil, p.errorHere("expected an attribute expression such as {state = 'open'}, not, or (")
}

func addParentheses(n Node) {
	switch n := n.(type) {
	case *AttributeExpression:
		n.Parentheses++
	case *AndExpression:
		n.Parentheses++
	case *OrExpression:
		n.Parentheses++
	case *NotExpression:
		n.Parentheses++
	}
}

func (p *parser) parseAttribute() (Node, error) {
	open := p.pos
	p.pos++ // {

	p.skipSpace()

	start := p.pos
	field := !p.done() && p.peek() == '@'

	if field {
		p.pos++
	}

	for !p.done() && attributeChar(p.peek()) {
		p.pos++
	}

	if field && p.pos == start+1 {
		return nil, p.errorAt(start, p.pos, "expected a field name: %s", fieldList)
	}

	if p.pos == start {
		if p.peek() == ':' {
			return nil, p.errorAt(p.pos, p.pos+1, "attribute names can't contain colons")
		}

		return nil, p.errorHere("expected an attribute name")
	}

	// A colon or other character directly after the name is part of an
	// invalid name, rather than the start of the operator.
	if !p.done() && !unicode.IsSpace(p.peek()) && !strings.ContainsRune("=!<>~}", p.peek()) {
		if p.peek() == ':' {
			return nil, p.errorAt(p.pos, p.pos+1, "attribute names can't contain colons")
		}

		return nil, p.errorAt(p.pos, p.pos+1, "attribute names may only contain letters, digits, and - _ . /")
	}

	attribute := string(p.input[start:p.pos])

	if _, ok := fields[attribute]; field && !ok {
		return nil, p.errorAt(start, p.pos, "unknown field %s; the fields are %s", attribute, fieldList)
	}

	operator, err := p.parseOperator()
	if err != nil {
		return nil, err
	}

	valueStart := p.skipSpaceAndMark()
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	if err := checkOperand(operator, value, valueStart, p.pos); err != nil {
		return nil, err
	}

	p.skipSpace()

	if p.peek() != '}' {
		if p.done() {
			return nil, p.errorAt(open, open+1, "this brace is never closed")
		}

		return nil, p.errorHere("expected a closing brace")
	}

	p.pos++

	return &AttributeExpression{Attribute: attribute, Operator: operator, Value: value}, nil
}

func (p *parser) skipSpaceAndMark() int {
	p.skipSpace()
	return p.pos
}

func (p *parser) parseOperator() (Operator, error) {
	p.skipSpace()

	// Two-character operators are tried first, so <= isn't read as <.
	for _, op := range []Operator{NotEqual, LessOrEqual, GreaterOrEqual, Equal, Less, Greater, Match} {
		end := p.pos + len(op)

		if end <= len(p.input) && string(p.input[p.pos:end]) == string(op) {
			p.pos = end
			return op, nil
		}
	}

	return "", p.errorHere("expected an operator: =, !=, <, >, <=, >=, or ~")
}

func (p *parser) parseValue() (Value, error) {
	start := p.pos

	switch r := p.peek(); {
	case r == '\'' || r == '"':
		return p.parseString()

	case r == '/':
		return p.parseRegexp()

	case r == '.':
		return p.parseLookup()

	case r == '(':
		return p.parseExpression()

	case r == '-' || (r >= '0' && r <= '9'):
		match := numberPattern.FindString(string(p.input[p.pos:]))

		if match == "" {
			return Value{}, p.errorHere("expected a number")
		}

		p.pos += utf8.RuneCountInString(match)

		// A number must end at a word boundary, so 12ab isn't read as 12.
		if !p.done() && (attributeChar(p.peek()) || p.peek() == '.') {
			return Value{}, p.errorAt(start, p.wordEnd(start), "expected a number")
		}

		return Value{Raw: match, Data: json.Number(match)}, nil
	}

	for _, literal := range []struct {
		word string
		data any
	}{{"null", nil}, {"true", true}, {"false", false}} {
		if p.keyword(literal.word) {
			return Value{Raw: literal.word, Data: literal.data}, nil
		}
	}

	return Value{}, p.errorHere("expected a value: a quoted string, a number, a /regular expression/, a .lookup, true, false, or null")
}

// lookupChar reports whether r may appear in a key of an object lookup:
// letters, digits, _, and -.
func lookupChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

// parseLookup parses an object lookup: keys after dots, as in .user.email.
func (p *parser) parseLookup() (Value, error) {
	start := p.pos

	for !p.done() && p.peek() == '.' {
		p.pos++
		keyStart := p.pos

		for !p.done() && lookupChar(p.peek()) {
			p.pos++
		}

		if p.pos == keyStart {
			return Value{}, p.errorAt(keyStart-1, keyStart, "expected a key after the dot: letters, digits, _, and -")
		}
	}

	// A lookup ends where its last key does, before space, the end of the
	// attribute expression, or punctuation in an expression.
	if !p.done() && !unicode.IsSpace(p.peek()) && !strings.ContainsRune("}),]:+*/%<>=!", p.peek()) {
		return Value{}, p.errorAt(p.pos, p.pos+1, "keys may only contain letters, digits, _, and -")
	}

	raw := string(p.input[start:p.pos])

	return Value{Raw: raw, Data: Lookup(raw)}, nil
}

// escapes maps the characters after a backslash in a string to what they
// stand for. \u is handled separately.
var escapes = map[rune]rune{
	'"': '"', '\'': '\'', '\\': '\\', '/': '/',
	'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t',
}

// parseString parses a quoted string. A string with expressions inserted
// in it, each written \( expression ), is an interpolated string: its value
// is an *Expression whose root is an *InterpExpr.
func (p *parser) parseString() (Value, error) {
	start := p.pos
	quote := p.peek()
	p.pos++

	var b strings.Builder

	interp := &InterpExpr{Quote: quote}
	segment := p.pos

	// endText adds the text since the last expression to the parts.
	endText := func(end int) {
		if end > segment {
			interp.Parts = append(interp.Parts, InterpPart{Raw: string(p.input[segment:end]), Text: b.String()})
		}

		b.Reset()
	}

	for {
		if p.done() {
			return Value{}, p.errorAt(start, start+1, "this string is never closed")
		}

		r := p.peek()

		switch {
		case r == quote:
			end := p.pos
			p.pos++

			if len(interp.Parts) == 0 {
				return Value{Raw: string(p.input[start:p.pos]), Data: b.String()}, nil
			}

			endText(end)

			return Value{Data: &Expression{Root: interp}}, nil

		case r == '\\':
			escapeStart := p.pos
			p.pos++

			if p.done() {
				return Value{}, p.errorAt(start, start+1, "this string is never closed")
			}

			e := p.peek()
			p.pos++

			// An inserted expression, parsed as a parenthesized one is.
			if e == '(' {
				endText(escapeStart)

				x, err := p.parseComparison()
				if err != nil {
					return Value{}, err
				}

				if err := p.closeParen(escapeStart + 1); err != nil {
					return Value{}, err
				}

				interp.Parts = append(interp.Parts, InterpPart{X: x})
				segment = p.pos

				continue
			}

			if e == 'u' {
				if p.pos+4 > len(p.input) {
					return Value{}, p.errorAt(escapeStart, len(p.input), `expected four hex digits after \u`)
				}

				code, err := strconv.ParseUint(string(p.input[p.pos:p.pos+4]), 16, 32)
				if err != nil {
					return Value{}, p.errorAt(escapeStart, p.pos+4, `expected four hex digits after \u`)
				}

				b.WriteRune(rune(code))
				p.pos += 4

				continue
			}

			replacement, ok := escapes[e]
			if !ok {
				return Value{}, p.errorAt(escapeStart, p.pos, `unknown escape \%c`, e)
			}

			b.WriteRune(replacement)

		case r < ' ':
			return Value{}, p.errorAt(p.pos, p.pos+1, "strings can't contain control characters; use an escape such as \\n")

		default:
			b.WriteRune(r)
			p.pos++
		}
	}
}

// parseRegexp parses a regular expression literal. Its text is taken
// literally, except that \/ stands for a slash, as in JavaScript.
func (p *parser) parseRegexp() (Value, error) {
	start := p.pos
	p.pos++

	var b strings.Builder

	for {
		if p.done() {
			return Value{}, p.errorAt(start, start+1, "this regular expression is never closed")
		}

		r := p.peek()

		switch {
		case r == '/':
			p.pos++
			return Value{Raw: string(p.input[start:p.pos]), Data: Regexp(b.String())}, nil

		case r == '\\' && p.pos+1 < len(p.input):
			next := p.input[p.pos+1]

			if next != '/' {
				// Other escapes, including \\, belong to the regular
				// expression.
				b.WriteRune(r)
			}

			b.WriteRune(next)
			p.pos += 2

		case r < ' ':
			return Value{}, p.errorAt(p.pos, p.pos+1, "regular expressions can't contain control characters")

		default:
			b.WriteRune(r)
			p.pos++
		}
	}
}

// checkOperand reports an operator that can't be used with value, which runs
// from start to end in the input.
func checkOperand(op Operator, value Value, start, end int) error {
	switch data := value.Data.(type) {
	// A lookup's or an expression's value is checked once it's known. See
	// Resolve.
	case Lookup, *Expression:
		return nil

	case string:
		if op == Match {
			if _, err := regexp.Compile(data); err != nil {
				return &SyntaxError{Start: start, End: end, Message: "invalid regular expression: " + err.Error()}
			}
		}

		return nil

	case Regexp:
		if op != Match {
			return &SyntaxError{Start: start, End: end, Message: "a regular expression literal can only follow ~"}
		}

		if _, err := regexp.Compile(string(data)); err != nil {
			return &SyntaxError{Start: start, End: end, Message: "invalid regular expression: " + err.Error()}
		}

		return nil

	case json.Number:
		if op == Match {
			return &SyntaxError{Start: start, End: end, Message: "~ needs a regular expression, such as /^in /"}
		}

		return nil
	}

	// null, true, and false.
	if op != Equal && op != NotEqual {
		return &SyntaxError{Start: start, End: end, Message: fmt.Sprintf("%s can only be compared with = or !=", value.Raw)}
	}

	return nil
}
