package filter

import (
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"strconv"
	"sync"

	"modernc.org/sqlite"
)

// The SQL functions filters use, which are registered with SQLite when the
// package is loaded.
const (
	// numberFunction returns its argument as a number if it's text that's a
	// JSON number, and NULL otherwise, so "5" compares as 5 but "8h" doesn't
	// compare at all.
	numberFunction = "ish_number"

	// regexpFunction returns 1 if its second argument matches the Go regular
	// expression in its first, and 0 otherwise.
	regexpFunction = "ish_regexp"
)

func init() {
	if err := sqlite.RegisterDeterministicScalarFunction(numberFunction, 1, sqlNumber); err != nil {
		panic(err)
	}

	if err := sqlite.RegisterDeterministicScalarFunction(regexpFunction, 2, sqlRegexp); err != nil {
		panic(err)
	}
}

func sqlNumber(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	text, ok := args[0].(string)

	if !ok || numberPattern.FindString(text) != text {
		return nil, nil
	}

	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, nil
	}

	return n, nil
}

// patterns caches compiled regular expressions, which are matched against
// every value a filter checks.
var patterns sync.Map

func sqlRegexp(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	pattern, ok := args[0].(string)
	if !ok {
		return int64(0), nil
	}

	text, ok := args[1].(string)
	if !ok {
		return int64(0), nil
	}

	cached, ok := patterns.Load(pattern)

	if !ok {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}

		cached, _ = patterns.LoadOrStore(pattern, compiled)
	}

	if cached.(*regexp.Regexp).MatchString(text) {
		return int64(1), nil
	}

	return int64(0), nil
}

// hasAttribute is true if the issue has the attribute given as its parameter.
const hasAttribute = `EXISTS (SELECT 1 FROM attribute WHERE attribute.issueId = issue.id AND attribute.key = ?)`

// anyValue is true if any of the issue's values for the attribute given as
// its first parameter satisfy the condition that's appended to it, which
// refers to the value as attributeValue.value.
const anyValue = `EXISTS (SELECT 1 FROM attribute, json_each(attribute.value) AS attributeValue
WHERE attribute.issueId = issue.id AND attribute.key = ? AND `

// Where returns an SQL condition for the attribute expression:
//
//   - An attribute with several values matches if any of them do, and !=
//     matches if none of them equal the value, so it's always the opposite
//     of =.
//   - An issue without the attribute matches only !=, except that = null
//     matches issues without the attribute and != null those with it.
//   - Strings compare as case-sensitive text. Numbers compare numerically
//     with values that are numbers, such as "5", and don't match other
//     values. true and false match the text "true" and "false".
func (e *AttributeExpression) Where() (string, []any) {
	if e.Value.Data == nil {
		if e.Operator == NotEqual {
			return hasAttribute, []any{e.Attribute}
		}

		return "NOT " + hasAttribute, []any{e.Attribute}
	}

	if e.Operator == NotEqual {
		equal := *e
		equal.Operator = Equal
		clause, args := equal.Where()

		return "NOT " + clause, args
	}

	var condition string
	var arg any

	switch data := e.Value.Data.(type) {
	case string:
		arg = data
		condition = "attributeValue.value " + string(e.Operator) + " ?"

		if e.Operator == Match {
			condition = regexpFunction + "(?, attributeValue.value)"
		}

	case Regexp:
		arg = string(data)
		condition = regexpFunction + "(?, attributeValue.value)"

	case json.Number:
		n, _ := data.Float64()
		arg = n
		condition = numberFunction + "(attributeValue.value) " + string(e.Operator) + " ?"

	case bool:
		arg = strconv.FormatBool(data)
		condition = "attributeValue.value " + string(e.Operator) + " ?"
	}

	return anyValue + condition + ")", []any{e.Attribute, arg}
}

func (e *AndExpression) Where() (string, []any) {
	return binary("AND", e.LeftExpression, e.RightExpression)
}

func (e *OrExpression) Where() (string, []any) {
	return binary("OR", e.LeftExpression, e.RightExpression)
}

func (e *NotExpression) Where() (string, []any) {
	clause, args := e.Expression.Where()
	return "NOT (" + clause + ")", args
}

func binary(op string, left, right Node) (string, []any) {
	leftClause, leftArgs := left.Where()
	rightClause, rightArgs := right.Where()

	return "(" + leftClause + ") " + op + " (" + rightClause + ")", append(leftArgs, rightArgs...)
}
