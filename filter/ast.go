// Package filter parses filter expressions, which select issues by their
// metadata, into an abstract syntax tree that can be printed back as text or
// converted into an SQL WHERE clause for the metadata database.
//
// An expression is made of attribute expressions such as
// {state != 'closed'}, joined by the logical operators and, or, and not, and
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

	// Where returns an SQL condition that's true for the issues the
	// expression matches, with ? placeholders for the values in args. The
	// condition refers to the issue being tested as issue.id, so it's meant
	// for a query on the issue table.
	Where() (clause string, args []any)

	// parentheses returns the number of pairs of parentheses the expression
	// was enclosed in.
	parentheses() int
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
// value, as in {state != 'closed'}.
type AttributeExpression struct {
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

func (e *AttributeExpression) parentheses() int { return e.Parentheses }
func (e *AndExpression) parentheses() int       { return e.Parentheses }
func (e *OrExpression) parentheses() int        { return e.Parentheses }
func (e *NotExpression) parentheses() int       { return e.Parentheses }

func (e *AttributeExpression) String() string {
	return parenthesize("{"+e.Attribute+" "+string(e.Operator)+" "+e.Value.String()+"}", e.Parentheses)
}

func (e *AndExpression) String() string {
	// An or inside an and needs parentheses to keep its meaning. Parsed
	// trees already record them; this covers trees built in code.
	return parenthesize(
		operand(e.LeftExpression, isOr)+" and "+operand(e.RightExpression, isOr),
		e.Parentheses,
	)
}

func (e *OrExpression) String() string {
	return parenthesize(e.LeftExpression.String()+" or "+e.RightExpression.String(), e.Parentheses)
}

func (e *NotExpression) String() string {
	return parenthesize("not "+operand(e.Expression, isBinary), e.Parentheses)
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

// operand prints n as an operand of a tighter-binding operator, adding a pair
// of parentheses if needs reports that n would otherwise change meaning and
// it has none of its own.
func operand(n Node, needs func(Node) bool) string {
	if needs(n) && n.parentheses() == 0 {
		return "(" + n.String() + ")"
	}

	return n.String()
}

func parenthesize(s string, n int) string {
	return strings.Repeat("(", n) + s + strings.Repeat(")", n)
}
