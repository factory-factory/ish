// Package filter parses filter expressions, which select issues by their
// metadata and fields, into an abstract syntax tree that can be printed back
// as text or converted into an SQL WHERE clause for the metadata database.
//
// An expression is made of attribute expressions such as
// {state != 'closed'} or {@author = 'alice'}, joined by the logical operators and, or, and not, and
// grouped with parentheses. See Parse.
package filter

import (
	"encoding/json"
	"strings"
)

// Node is a node in a filter expression's syntax tree.
type Node interface {
	// String returns the expression as text. Parsing the text gives back
	// the same tree, and the same tokens that were parsed, though the space
	// between tokens is normalized.
	String() string

	// Render returns the same text as String, with each kind of token
	// colored for display in the terminal.
	Render() string

	// Where returns an SQL condition that's true for the issues the
	// expression matches, with ? placeholders for the values in args. The
	// condition refers to the issue being tested as issue.id, so it's meant
	// for a query on the issue table.
	Where() (clause string, args []any)

	// parentheses returns the number of pairs of parentheses the expression
	// was enclosed in.
	parentheses() int

	// format returns the expression as text, with its tokens styled by p.
	format(p printer) string
}

// Operator is a comparison operator in an attribute expression.
type Operator string

// The comparison operators.
const (
	Equal          Operator = "="
	NotEqual       Operator = "!="
	Less           Operator = "<"
	Greater        Operator = ">"
	LessOrEqual    Operator = "<="
	GreaterOrEqual Operator = ">="
	Match          Operator = "~"
)

// Value is the value in an attribute expression: a JSON scalar value, or a
// regular expression literal.
type Value struct {
	// Raw is the value as it was written, such as 'closed', "closed", 1.50,
	// or /^in /, so it can be printed back exactly.
	Raw string

	// Data is the value: nil for null, a bool, a json.Number, a string, or a
	// Regexp.
	Data any
}

// Regexp is the pattern of a regular expression literal, such as /^in\s/,
// which can only be used with the ~ operator. Unlike a string, its text is
// taken literally, so backslashes reach the regular expression unchanged.
type Regexp string

// AttributeExpression compares the values of a metadata attribute with a
// value, as in {state != 'closed'}, or the value of one of the issue's
// fields, as in {@author = 'alice'}.
type AttributeExpression struct {
	// Attribute is the name of a metadata attribute, or of a field
	// including its @, such as @title.
	Attribute string
	Operator  Operator
	Value     Value

	// Parentheses is the number of pairs of parentheses the expression was
	// enclosed in. A count, rather than a flag, keeps ((x)) and (x) distinct.
	Parentheses int
}

// AndExpression matches issues that both of its expressions match.
type AndExpression struct {
	LeftExpression  Node
	RightExpression Node
	Parentheses     int
}

// OrExpression matches issues that either of its expressions match.
type OrExpression struct {
	LeftExpression  Node
	RightExpression Node
	Parentheses     int
}

// NotExpression matches issues that its expression doesn't match.
type NotExpression struct {
	Expression  Node
	Parentheses int
}

// fields maps the names of the issue fields that filters can compare, which
// start with @, to their columns in the issue table.
var fields = map[string]string{
	"@title":       "issue.title",
	"@author":      "issue.author",
	"@dateCreated": "issue.dateCreated",
}

// fieldList lists the fields, for error messages.
const fieldList = "@title, @author, and @dateCreated"

func (e *AttributeExpression) parentheses() int { return e.Parentheses }
func (e *AndExpression) parentheses() int       { return e.Parentheses }
func (e *OrExpression) parentheses() int        { return e.Parentheses }
func (e *NotExpression) parentheses() int       { return e.Parentheses }

func (e *AttributeExpression) String() string { return e.format(plain) }
func (e *AndExpression) String() string       { return e.format(plain) }
func (e *OrExpression) String() string        { return e.format(plain) }
func (e *NotExpression) String() string       { return e.format(plain) }

func (e *AttributeExpression) Render() string { return e.format(highlight) }
func (e *AndExpression) Render() string       { return e.format(highlight) }
func (e *OrExpression) Render() string        { return e.format(highlight) }
func (e *NotExpression) Render() string       { return e.format(highlight) }

func (e *AttributeExpression) format(p printer) string {
	return parenthesize(p,
		p.brace("{")+p.attribute(e.Attribute)+" "+p.operator(string(e.Operator))+" "+p.value(e.Value)+p.brace("}"),
		e.Parentheses,
	)
}

func (e *AndExpression) format(p printer) string {
	// An or inside an and needs parentheses to keep its meaning. Parsed
	// trees already record them; this covers trees built in code.
	return parenthesize(p,
		operand(p, e.LeftExpression, isOr)+" "+p.keyword("and")+" "+operand(p, e.RightExpression, isOr),
		e.Parentheses,
	)
}

func (e *OrExpression) format(p printer) string {
	return parenthesize(p,
		e.LeftExpression.format(p)+" "+p.keyword("or")+" "+e.RightExpression.format(p),
		e.Parentheses,
	)
}

func (e *NotExpression) format(p printer) string {
	return parenthesize(p, p.keyword("not")+" "+operand(p, e.Expression, isBinary), e.Parentheses)
}

// String returns the value as it was written, or as JSON or a regular
// expression literal if it was created without its original text.
func (v Value) String() string {
	if v.Raw != "" {
		return v.Raw
	}

	if pattern, ok := v.Data.(Regexp); ok {
		return "/" + strings.ReplaceAll(string(pattern), "/", `\/`) + "/"
	}

	data, err := json.Marshal(v.Data)
	if err != nil {
		return "null"
	}

	return string(data)
}

func isOr(n Node) bool {
	_, ok := n.(*OrExpression)
	return ok
}

func isBinary(n Node) bool {
	switch n.(type) {
	case *AndExpression, *OrExpression:
		return true
	}
	return false
}

// operand formats n as an operand of a tighter-binding operator, adding a
// pair of parentheses if needs reports that n would otherwise change meaning
// and it has none of its own.
func operand(p printer, n Node, needs func(Node) bool) string {
	if needs(n) && n.parentheses() == 0 {
		return p.paren("(") + n.format(p) + p.paren(")")
	}

	return n.format(p)
}

func parenthesize(p printer, s string, n int) string {
	return strings.Repeat(p.paren("("), n) + s + strings.Repeat(p.paren(")"), n)
}
