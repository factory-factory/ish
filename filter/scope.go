package filter

import (
	"strings"
)

// ScopeExpression tests conditions against the same row of its subject, as
// in {estimate >= 3 and <= 7} or
// {@comments author = 'alice@example.com' and text ~ /LGTM/}. It matches if
// any one row satisfies every condition, so {a and b} differs from
// {a} and {b}, which two rows can satisfy.
//
// The subject is an attribute or field, whose rows each hold one value, so
// its comparisons start with an operator, or a collection such as
// @comments, whose rows have fields, so its comparisons start with a field
// name. A plain {subject operator value} is an AttributeExpression instead.
type ScopeExpression struct {
	Subject   string
	Condition Condition

	Parentheses int
}

// Condition is a condition on a row of a scope's subject.
type Condition interface {
	formatCondition(p printer) string
}

// Comparison compares a row's value, or, in a collection, its field, with a
// value.
type Comparison struct {
	// Field is the field of a collection's row, such as author, or "" for
	// a subject whose rows each hold one value.
	Field    string
	Operator Operator
	Value    Value
}

// ConditionAnd is true of a row that both of its conditions are true of.
type ConditionAnd struct {
	Left, Right Condition
}

// ConditionNot is true of a row that its condition isn't true of.
type ConditionNot struct {
	X Condition
}

// ConditionGroup is a condition in parentheses.
type ConditionGroup struct {
	X Condition
}

// collections are the subjects whose rows have several fields, mapped to
// those fields, by their names in a scope, and the fields they are
// elsewhere.
var collections = map[string]map[string]string{
	"@comments": {
		"author":      "@comments.author",
		"text":        "@comments.text",
		"dateCreated": "@comments.dateCreated",
	},
}

// collectionTables are the tables collections' rows are in.
var collectionTables = map[string]string{
	"@comments": "comment",
}

// collectionFieldList lists a collection's fields, for error messages.
var collectionFieldList = map[string]string{
	"@comments": "author, text, and dateCreated",
}

func (e *ScopeExpression) parentheses() int { return e.Parentheses }
func (e *ScopeExpression) String() string   { return e.format(plain) }
func (e *ScopeExpression) Render() string   { return e.format(highlight) }

func (e *ScopeExpression) format(p printer) string {
	return parenthesize(p,
		p.brace("{")+p.attribute(e.Subject)+" "+e.Condition.formatCondition(p)+p.brace("}"),
		e.Parentheses,
	)
}

func (c *Comparison) formatCondition(p printer) string {
	value := p.value(c.Value)
	if expr, ok := c.Value.Data.(*Expression); ok {
		value = expr.format(p)
	}

	s := p.operator(string(c.Operator)) + " " + value

	if c.Field != "" {
		s = p.attribute(c.Field) + " " + s
	}

	return s
}

func (c *ConditionAnd) formatCondition(p printer) string {
	return c.Left.formatCondition(p) + " " + p.keyword("and") + " " + c.Right.formatCondition(p)
}

func (c *ConditionNot) formatCondition(p printer) string {
	return p.keyword("not") + " " + c.X.formatCondition(p)
}

func (c *ConditionGroup) formatCondition(p printer) string {
	return p.paren("(") + c.X.formatCondition(p) + p.paren(")")
}

// single returns the scope's only comparison, if its condition is just one
// comparison, without and, not, or parentheses. Such a scope means what the
// same comparison does as an AttributeExpression.
func (e *ScopeExpression) single() (*Comparison, bool) {
	c, ok := e.Condition.(*Comparison)
	return c, ok
}

// attributeFor returns the attribute expression comparing a row's value or
// field as c does, as it would be written outside a scope.
func (e *ScopeExpression) attributeFor(c *Comparison) *AttributeExpression {
	name := e.Subject
	if c.Field != "" {
		name = collections[e.Subject][c.Field]
	}

	return &AttributeExpression{Attribute: name, Operator: c.Operator, Value: c.Value}
}

// comparisons returns the scope's comparisons, in order.
func comparisons(c Condition) []*Comparison {
	switch c := c.(type) {
	case *Comparison:
		return []*Comparison{c}
	case *ConditionAnd:
		return append(comparisons(c.Left), comparisons(c.Right)...)
	case *ConditionNot:
		return comparisons(c.X)
	case *ConditionGroup:
		return comparisons(c.X)
	}

	return nil
}

// Where returns an SQL condition that's true if any row of the subject
// satisfies the scope's condition: an EXISTS over the attribute's values,
// or the rows of a table such as comment, with every comparison inside it.
// A scope with a single comparison means what the comparison does as an
// attribute expression. A lookup in the scope that found nothing makes it
// UNKNOWN, written NULL, as it does an attribute expression.
func (e *ScopeExpression) Where() (string, []any) {
	if c, ok := e.single(); ok {
		return e.attributeFor(c).Where()
	}

	for _, c := range comparisons(e.Condition) {
		switch c.Value.Data.(type) {
		case Lookup, unknown:
			return "NULL", nil
		}
	}

	condition, args := e.conditionSQL(e.Condition)

	switch table, isRows := rowFields[e.Subject]; {
	case collectionTables[e.Subject] != "":
		return anyRow(collectionTables[e.Subject]) + condition + ")", args

	case isRows:
		return anyRow(table) + condition + ")", args

	case fields[e.Subject] != "":
		// An issue field, such as @title, has one row: the issue's.
		return "(" + condition + ")", args
	}

	return anyValue + condition + ")", append([]any{e.Subject}, args...)
}

// column returns the SQL column a comparison in the scope compares.
func (e *ScopeExpression) column(c *Comparison) string {
	if c.Field != "" {
		return fields[collections[e.Subject][c.Field]]
	}

	if column, ok := fields[e.Subject]; ok {
		return column
	}

	return "attributeValue.value"
}

// conditionSQL returns SQL for a condition on one row. Each comparison is
// either true or false, so not is always its opposite.
func (e *ScopeExpression) conditionSQL(c Condition) (string, []any) {
	switch c := c.(type) {
	case *ConditionAnd:
		left, leftArgs := e.conditionSQL(c.Left)
		right, rightArgs := e.conditionSQL(c.Right)

		return "(" + left + ") AND (" + right + ")", append(leftArgs, rightArgs...)

	case *ConditionNot:
		x, args := e.conditionSQL(c.X)
		return "NOT (" + x + ")", args

	case *ConditionGroup:
		x, args := e.conditionSQL(c.X)
		return "(" + x + ")", args

	case *Comparison:
		return e.comparisonSQL(c)
	}

	return "0", nil
}

// comparisonSQL returns SQL comparing one row's value or field with a value:
// a literal, any value of a list, or an expression's value.
func (e *ScopeExpression) comparisonSQL(c *Comparison) (string, []any) {
	column := e.column(c)
	one := e.attributeFor(c)

	switch data := c.Value.Data.(type) {
	case valueList:
		var clauses []string
		var args []any

		for _, v := range data {
			el := *one
			el.Value = Value{Data: v}

			clause, arg := el.compare(column)
			clauses = append(clauses, "coalesce(("+clause+"), 0)")
			args = append(args, arg)
		}

		if len(clauses) == 0 {
			return "0", nil
		}

		return "(" + strings.Join(clauses, " OR ") + ")", args

	case *Expression:
		value, valueArgs := data.SQL()

		return "coalesce(" + compareFunction + "(?, " + column + ", " + value + "), 0)",
			append([]any{string(c.Operator)}, valueArgs...)
	}

	clause, arg := one.compare(column)

	return "coalesce((" + clause + "), 0)", []any{arg}
}
