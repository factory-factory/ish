package filter

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// Expression is a parenthesized expression used as a value, as in
// {count = (3 + 7)} or {email = (.user.email :lower)}. Its operands are
// literals, lists, and object lookups. It's evaluated by SQL functions
// written in Go, which are also used to check it when it's resolved.
type Expression struct {
	Root Expr
}

// Expr is a part of an expression.
type Expr interface {
	// format returns the expression as text, with its tokens styled by p.
	format(p printer) string

	// eval returns the expression's value, with lookups found by lookup.
	eval(lookup LookupFunc) (any, error)

	// sql returns SQL that evaluates the expression, as JSON text, with ?
	// placeholders for the values in args. Lookups must be resolved.
	sql() (string, []any)

	// resolve returns a copy of the expression with its lookups' values
	// from lookup.
	resolve(lookup LookupFunc) (Expr, error)
}

// LiteralExpr is a literal string, number, boolean, or null.
type LiteralExpr struct {
	Value Value
}

// LookupExpr is an object lookup. Once it's resolved, Found reports whether
// a value was found, and Data holds it as an expression value.
type LookupExpr struct {
	Path     Lookup
	Resolved bool
	Found    bool
	Data     any
}

// ListExpr is a list of values, as in [1, 2, 3].
type ListExpr struct {
	Elements []Expr
}

// UnaryExpr applies - or ! to an expression.
type UnaryExpr struct {
	Operator string
	X        Expr
}

// BinaryExpr applies an arithmetic or comparison operator to two
// expressions.
type BinaryExpr struct {
	Operator    string
	Left, Right Expr
}

// CallExpr applies a function to a value and any arguments, as in
// (.email :lower) or ([1, 2] :join(',')). Args holds the value first.
type CallExpr struct {
	Name string
	Args []Expr
}

// GroupExpr is an expression in parentheses within an expression.
type GroupExpr struct {
	X Expr
}

// Formatting.

func (e *Expression) format(p printer) string {
	return p.paren("(") + e.Root.format(p) + p.paren(")")
}

func (e *LiteralExpr) format(p printer) string { return p.value(e.Value) }

func (e *LookupExpr) format(p printer) string {
	return p.value(Value{Raw: string(e.Path), Data: e.Path})
}

func (e *ListExpr) format(p printer) string {
	parts := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		parts[i] = el.format(p)
	}

	return p.paren("[") + strings.Join(parts, ", ") + p.paren("]")
}

func (e *UnaryExpr) format(p printer) string { return p.operator(e.Operator) + e.X.format(p) }

func (e *BinaryExpr) format(p printer) string {
	return e.Left.format(p) + " " + p.operator(e.Operator) + " " + e.Right.format(p)
}

func (e *CallExpr) format(p printer) string {
	s := e.Args[0].format(p) + " " + p.function(":"+e.Name)

	if len(e.Args) > 1 {
		args := make([]string, len(e.Args)-1)
		for i, a := range e.Args[1:] {
			args[i] = a.format(p)
		}

		s += p.paren("(") + strings.Join(args, ", ") + p.paren(")")
	}

	return s
}

func (e *GroupExpr) format(p printer) string { return p.paren("(") + e.X.format(p) + p.paren(")") }

// Evaluation in Go.

// Eval returns the expression's value, with lookups found by lookup, which
// should read one snapshot of the application's state.
func (e *Expression) Eval(lookup LookupFunc) (any, error) {
	return e.Root.eval(lookup)
}

func (e *LiteralExpr) eval(LookupFunc) (any, error) {
	if n, ok := e.Value.Data.(json.Number); ok {
		f, err := n.Float64()
		return f, err
	}

	return e.Value.Data, nil
}

func (e *LookupExpr) eval(lookup LookupFunc) (any, error) {
	if e.Resolved {
		return e.Data, nil
	}

	resolved, err := e.resolve(lookup)
	if err != nil {
		return nil, err
	}

	return resolved.(*LookupExpr).Data, nil
}

func (e *ListExpr) eval(lookup LookupFunc) (any, error) {
	list := make([]any, len(e.Elements))

	for i, el := range e.Elements {
		v, err := el.eval(lookup)
		if err != nil {
			return nil, err
		}

		list[i] = v
	}

	return list, nil
}

func (e *UnaryExpr) eval(lookup LookupFunc) (any, error) {
	x, err := e.X.eval(lookup)
	if err != nil {
		return nil, err
	}

	return evalUnary(e.Operator, x)
}

func (e *BinaryExpr) eval(lookup LookupFunc) (any, error) {
	a, err := e.Left.eval(lookup)
	if err != nil {
		return nil, err
	}

	b, err := e.Right.eval(lookup)
	if err != nil {
		return nil, err
	}

	return evalBinary(e.Operator, a, b)
}

func (e *CallExpr) eval(lookup LookupFunc) (any, error) {
	args := make([]any, len(e.Args))

	for i, a := range e.Args {
		v, err := a.eval(lookup)
		if err != nil {
			return nil, err
		}

		args[i] = v
	}

	return applyFunction(e.Name, args)
}

func (e *GroupExpr) eval(lookup LookupFunc) (any, error) { return e.X.eval(lookup) }

// Compiling to SQL. Each value is passed as a parameter of JSON text, and
// each operator and function is an SQL function written in Go.

// SQL returns SQL that evaluates the expression as JSON text, with ?
// placeholders for the values in args.
func (e *Expression) SQL() (string, []any) {
	return e.Root.sql()
}

func (e *LiteralExpr) sql() (string, []any) {
	v, _ := e.eval(nil)
	return "?", []any{encodeValue(v)}
}

// sql passes the looked-up value; a lookup that wasn't resolved, or found
// nothing, is null.
func (e *LookupExpr) sql() (string, []any) {
	return "?", []any{encodeValue(e.Data)}
}

func (e *ListExpr) sql() (string, []any) {
	return call(listFunction, nil, e.Elements)
}

func (e *UnaryExpr) sql() (string, []any) {
	return call(unaryFunction, []any{e.Operator}, []Expr{e.X})
}

func (e *BinaryExpr) sql() (string, []any) {
	return call(binaryFunction, []any{e.Operator}, []Expr{e.Left, e.Right})
}

func (e *CallExpr) sql() (string, []any) {
	return call(functionPrefix+e.Name, nil, e.Args)
}

func (e *GroupExpr) sql() (string, []any) { return e.X.sql() }

// call returns SQL calling the named SQL function with the parameters
// first, then the expressions.
func call(name string, params []any, exprs []Expr) (string, []any) {
	parts := make([]string, 0, len(params)+len(exprs))
	args := append([]any(nil), params...)

	for range params {
		parts = append(parts, "?")
	}

	for _, x := range exprs {
		s, a := x.sql()
		parts = append(parts, s)
		args = append(args, a...)
	}

	return name + "(" + strings.Join(parts, ", ") + ")", args
}

// Resolving lookups.

func (e *LiteralExpr) resolve(LookupFunc) (Expr, error) { return e, nil }

func (e *LookupExpr) resolve(lookup LookupFunc) (Expr, error) {
	resolved := &LookupExpr{Path: e.Path, Resolved: true}

	if lookup == nil {
		return resolved, nil
	}

	v, ok := lookup(string(e.Path))
	if !ok {
		return resolved, nil
	}

	data, err := exprValue(e.Path, v)
	if err != nil {
		return nil, err
	}

	resolved.Found = true
	resolved.Data = data

	return resolved, nil
}

func (e *ListExpr) resolve(lookup LookupFunc) (Expr, error) {
	els, err := resolveAll(e.Elements, lookup)
	if err != nil {
		return nil, err
	}

	return &ListExpr{Elements: els}, nil
}

func (e *UnaryExpr) resolve(lookup LookupFunc) (Expr, error) {
	x, err := e.X.resolve(lookup)
	if err != nil {
		return nil, err
	}

	return &UnaryExpr{Operator: e.Operator, X: x}, nil
}

func (e *BinaryExpr) resolve(lookup LookupFunc) (Expr, error) {
	both, err := resolveAll([]Expr{e.Left, e.Right}, lookup)
	if err != nil {
		return nil, err
	}

	return &BinaryExpr{Operator: e.Operator, Left: both[0], Right: both[1]}, nil
}

func (e *CallExpr) resolve(lookup LookupFunc) (Expr, error) {
	args, err := resolveAll(e.Args, lookup)
	if err != nil {
		return nil, err
	}

	return &CallExpr{Name: e.Name, Args: args}, nil
}

func (e *GroupExpr) resolve(lookup LookupFunc) (Expr, error) {
	x, err := e.X.resolve(lookup)
	if err != nil {
		return nil, err
	}

	return &GroupExpr{X: x}, nil
}

func resolveAll(exprs []Expr, lookup LookupFunc) ([]Expr, error) {
	out := make([]Expr, len(exprs))

	for i, x := range exprs {
		r, err := x.resolve(lookup)
		if err != nil {
			return nil, err
		}

		out[i] = r
	}

	return out, nil
}

// exprValue returns a looked-up value as an expression value.
func exprValue(path Lookup, v any) (any, error) {
	switch v := v.(type) {
	case string, bool:
		return v, nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	case []any:
		list := make([]any, len(v))
		for i, e := range v {
			el, err := exprValue(path, e)
			if err != nil {
				return nil, err
			}
			list[i] = el
		}
		return list, nil
	case map[string]any:
		return nil, &TypeError{path, "is a table, which can't be used in an expression"}
	}

	return nil, &TypeError{path, fmt.Sprintf("is a %T, which can't be used in an expression", v)}
}

// ExprError reports an expression that can't be evaluated, such as one
// applying a function to a value of the wrong type.
type ExprError struct {
	Expression string
	Err        error
}

func (e *ExprError) Error() string {
	return fmt.Sprintf("%s: %v", e.Expression, e.Err)
}

func (e *ExprError) Unwrap() error { return e.Err }

// Parsing.

// parseExpression parses a parenthesized expression used as a value.
func (p *parser) parseExpression() (Value, error) {
	open := p.pos
	p.pos++ // (

	root, err := p.parseComparison()
	if err != nil {
		return Value{}, err
	}

	if err := p.closeParen(open); err != nil {
		return Value{}, err
	}

	return Value{Data: &Expression{Root: root}}, nil
}

// closeParen consumes the parenthesis that closes the one at open.
func (p *parser) closeParen(open int) error {
	p.skipSpace()

	if p.done() {
		return p.errorAt(open, open+1, "this parenthesis is never closed")
	}

	if p.peek() != ')' {
		return p.errorHere("expected an operator, a :function, or a closing parenthesis")
	}

	p.pos++

	return nil
}

// comparisonOperators are the comparison operators in expressions, longest
// first so <= isn't read as <.
var comparisonOperators = []string{"!=", "<=", ">=", "=", "<", ">"}

func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parseSum()
	if err != nil {
		return nil, err
	}

	op := p.operatorAhead(comparisonOperators)
	if op == "" {
		return left, nil
	}

	p.pos += len(op)

	right, err := p.parseSum()
	if err != nil {
		return nil, err
	}

	if next := p.operatorAhead(comparisonOperators); next != "" {
		return nil, p.errorAt(p.pos, p.pos+len(next), "comparisons can't be chained; use parentheses")
	}

	return &BinaryExpr{Operator: op, Left: left, Right: right}, nil
}

func (p *parser) parseSum() (Expr, error) {
	left, err := p.parseProduct()
	if err != nil {
		return nil, err
	}

	for {
		op := p.operatorAhead([]string{"+", "-"})
		if op == "" {
			return left, nil
		}

		p.pos++

		right, err := p.parseProduct()
		if err != nil {
			return nil, err
		}

		left = &BinaryExpr{Operator: op, Left: left, Right: right}
	}
}

func (p *parser) parseProduct() (Expr, error) {
	left, err := p.parsePower()
	if err != nil {
		return nil, err
	}

	for {
		op := p.operatorAhead([]string{"**", "*", "/", "%"})
		if op == "" || op == "**" {
			return left, nil
		}

		p.pos++

		right, err := p.parsePower()
		if err != nil {
			return nil, err
		}

		left = &BinaryExpr{Operator: op, Left: left, Right: right}
	}
}

// parsePower parses exponentiation, which groups from the right.
func (p *parser) parsePower() (Expr, error) {
	base, err := p.parseChain()
	if err != nil {
		return nil, err
	}

	if p.operatorAhead([]string{"**"}) == "" {
		return base, nil
	}

	p.pos += 2

	exponent, err := p.parsePower()
	if err != nil {
		return nil, err
	}

	return &BinaryExpr{Operator: "**", Left: base, Right: exponent}, nil
}

// parseChain parses a value followed by any functions applied to it.
func (p *parser) parseChain() (Expr, error) {
	x, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipSpace()

		if p.done() || p.peek() != ':' {
			return x, nil
		}

		start := p.pos
		p.pos++

		nameStart := p.pos
		for !p.done() && (unicode.IsLetter(p.peek()) || unicode.IsDigit(p.peek())) {
			p.pos++
		}

		name := string(p.input[nameStart:p.pos])
		if name == "" {
			return nil, p.errorAt(start, start+1, "expected a function name after the colon")
		}

		f, ok := functions[name]
		if !ok {
			return nil, p.errorAt(start, p.pos, "unknown function :%s", name)
		}

		args := []Expr{x}

		if !p.done() && p.peek() == '(' {
			open := p.pos
			p.pos++
			p.skipSpace()

			if !p.done() && p.peek() == ')' {
				p.pos++
			} else {
				for {
					arg, err := p.parseComparison()
					if err != nil {
						return nil, err
					}

					args = append(args, arg)
					p.skipSpace()

					if !p.done() && p.peek() == ',' {
						p.pos++
						continue
					}

					if err := p.closeParen(open); err != nil {
						return nil, err
					}

					break
				}
			}
		}

		if n := len(args) - 1; n < f.minArgs || n > f.maxArgs {
			return nil, p.errorAt(start, p.pos, ":%s takes %s", name, arity(f))
		}

		x = &CallExpr{Name: name, Args: args}
	}
}

func (p *parser) parseUnary() (Expr, error) {
	p.skipSpace()

	if p.done() {
		return nil, p.errorHere("expected a value")
	}

	switch r := p.peek(); {
	case r == '-', r == '!' && !p.ahead("!="):
		p.pos++

		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return &UnaryExpr{Operator: string(r), X: x}, nil
	}

	return p.parseOperand()
}

func (p *parser) parseOperand() (Expr, error) {
	switch r := p.peek(); {
	case r == '(':
		open := p.pos
		p.pos++

		x, err := p.parseComparison()
		if err != nil {
			return nil, err
		}

		if err := p.closeParen(open); err != nil {
			return nil, err
		}

		return &GroupExpr{X: x}, nil

	case r == '[':
		return p.parseList()

	case r == '.':
		v, err := p.parseLookup()
		if err != nil {
			return nil, err
		}

		return &LookupExpr{Path: v.Data.(Lookup)}, nil

	case r == '/':
		return nil, p.errorHere("a regular expression literal can only follow ~ directly")

	case r == '\'' || r == '"':
		v, err := p.parseString()
		if err != nil {
			return nil, err
		}

		return &LiteralExpr{Value: v}, nil

	case r >= '0' && r <= '9':
		return p.parseExprNumber()
	}

	for _, literal := range []struct {
		word string
		data any
	}{{"null", nil}, {"true", true}, {"false", false}} {
		if p.keyword(literal.word) {
			return &LiteralExpr{Value: Value{Raw: literal.word, Data: literal.data}}, nil
		}
	}

	return nil, p.errorHere("expected a value: a quoted string, a number, a [list], a .lookup, true, false, null, or an expression in parentheses")
}

// parseExprNumber parses a number in an expression, which, unlike one
// compared with an attribute, can be followed directly by an operator, as in
// (3/2). A minus sign is the unary operator.
func (p *parser) parseExprNumber() (Expr, error) {
	start := p.pos
	match := numberPattern.FindString(string(p.input[p.pos:]))

	p.pos += len(match)

	if !p.done() && (unicode.IsLetter(p.peek()) || unicode.IsDigit(p.peek()) || p.peek() == '.') {
		return nil, p.errorAt(start, p.wordEnd(start), "expected a number")
	}

	return &LiteralExpr{Value: Value{Raw: match, Data: json.Number(match)}}, nil
}

func (p *parser) parseList() (Expr, error) {
	open := p.pos
	p.pos++ // [

	list := &ListExpr{}

	p.skipSpace()
	if !p.done() && p.peek() == ']' {
		p.pos++
		return list, nil
	}

	for {
		el, err := p.parseComparison()
		if err != nil {
			return nil, err
		}

		list.Elements = append(list.Elements, el)
		p.skipSpace()

		switch {
		case p.done():
			return nil, p.errorAt(open, open+1, "this list is never closed")
		case p.peek() == ',':
			p.pos++
		case p.peek() == ']':
			p.pos++
			return list, nil
		default:
			return nil, p.errorHere("expected a comma or a closing bracket")
		}
	}
}

// operatorAhead returns the first of ops that comes next, after any space,
// moving to it, or "" if none does.
func (p *parser) operatorAhead(ops []string) string {
	save := p.pos
	p.skipSpace()

	for _, op := range ops {
		if p.ahead(op) {
			return op
		}
	}

	p.pos = save

	return ""
}

// ahead reports whether s comes next.
func (p *parser) ahead(s string) bool {
	return strings.HasPrefix(string(p.input[p.pos:]), s)
}
