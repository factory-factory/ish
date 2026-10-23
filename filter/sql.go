package filter

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

	// These evaluate expressions. Their values are JSON text, so they keep
	// their types: listFunction makes a list of its arguments,
	// unaryFunction and binaryFunction apply the operator in their first
	// argument, and each function in an expression is functionPrefix
	// followed by its name.
	listFunction = "ish_list"

	// interpolateFunction joins the parts of an interpolated string.
	interpolateFunction = "ish_interpolate"

	unaryFunction  = "ish_unary"
	binaryFunction = "ish_binary"
	functionPrefix = "ish_"

	// compareFunction returns 1 if an attribute's value, its second
	// argument, compares with an expression's value, its third, by the
	// operator in its first, 0 if it doesn't, and NULL if the expression's
	// value is null. See compareWith.
	compareFunction = "ish_compare"
)

func init() {
	if err := sqlite.RegisterDeterministicScalarFunction(numberFunction, 1, sqlNumber); err != nil {
		panic(err)
	}

	if err := sqlite.RegisterDeterministicScalarFunction(regexpFunction, 2, sqlRegexp); err != nil {
		panic(err)
	}

	register := func(name string, n int32, f func(args []any) (any, error)) {
		err := sqlite.RegisterDeterministicScalarFunction(name, n, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			values := make([]any, len(args))

			for i, a := range args {
				v, err := sqlValue(a)
				if err != nil {
					return nil, err
				}

				values[i] = v
			}

			v, err := f(values)
			if err != nil {
				return nil, err
			}

			return encodeValue(v), nil
		})

		if err != nil {
			panic(err)
		}
	}

	register(listFunction, -1, func(args []any) (any, error) { return args, nil })
	register(interpolateFunction, -1, interpolate)
	register(unaryFunction, 2, func(args []any) (any, error) { return evalUnary(args[0].(string), args[1]) })
	register(binaryFunction, 3, func(args []any) (any, error) { return evalBinary(args[0].(string), args[1], args[2]) })

	for name := range functions {
		register(sqlFunctionName(name), -1, func(args []any) (any, error) { return applyFunction(name, args) })
	}

	if err := sqlite.RegisterDeterministicScalarFunction(compareFunction, 3, sqlCompare); err != nil {
		panic(err)
	}
}

// sqlValue decodes an argument of an expression function: JSON text, or an
// operator's name, which is passed as a string.
func sqlValue(arg driver.Value) (any, error) {
	text, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("expected an expression's value, not %T", arg)
	}

	if v, err := decodeValue(text); err == nil {
		return v, nil
	}

	return text, nil
}

func sqlCompare(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	op, _ := args[0].(string)

	text, ok := args[1].(string)
	if !ok {
		return int64(0), nil
	}

	value, _ := args[2].(string)

	v, err := decodeValue(value)
	if err != nil {
		return nil, err
	}

	ok2, err := compareWith(Operator(op), text, v)
	switch {
	case err != nil:
		return nil, err
	case ok2 == nil:
		return nil, nil
	case ok2 == true:
		return int64(1), nil
	}

	return int64(0), nil
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
//     Every issue has every field, so = null never matches a field.
//   - Strings compare as case-sensitive text. Numbers compare numerically
//     with values that are numbers, such as "5", and don't match other
//     values. true and false match the text "true" and "false".
func (e *AttributeExpression) Where() (string, []any) {
	switch data := e.Value.Data.(type) {
	// A lookup that wasn't resolved, or found nothing, can't be compared, so
	// the expression is UNKNOWN, as SQL's NULL is: it matches nothing on its
	// own, but doesn't stop an or from matching. See Resolve.
	case Lookup, unknown:
		return "NULL", nil

	case valueList:
		return e.listWhere(data)

	case *Expression:
		return e.expressionWhere(data)
	}

	if column, ok := fields[e.Attribute]; ok {
		return e.fieldWhere(column)
	}

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

	condition, arg := e.compare("attributeValue.value")

	return anyValue + condition + ")", []any{e.Attribute, arg}
}

// expressionWhere returns an SQL condition comparing with an expression's
// value, which SQL functions evaluate: true if the comparison holds for any
// of the attribute's values, as for a literal, or for !=, if = holds for
// none of them. Resolve has already checked the expression, and found its
// value isn't null.
func (e *AttributeExpression) expressionWhere(expr *Expression) (string, []any) {
	op := e.Operator
	if op == NotEqual {
		op = Equal
	}

	value, valueArgs := expr.SQL()
	compare := func(column string) string {
		return compareFunction + "(?, " + column + ", " + value + ")"
	}

	var clause string
	var args []any

	if column, ok := fields[e.Attribute]; ok {
		clause = "coalesce(" + compare(column) + ", 0)"
		args = append([]any{string(op)}, valueArgs...)
	} else {
		clause = anyValue + compare("attributeValue.value") + ")"
		args = append([]any{e.Attribute, string(op)}, valueArgs...)
	}

	if e.Operator == NotEqual {
		return "NOT " + clause, args
	}

	return clause, args
}

// listWhere returns an SQL condition comparing with each of values, a
// looked-up array: true if the comparison holds for any of them, or for !=,
// if = holds for none of them. Each comparison is either true or false, so
// != is always the opposite of =.
func (e *AttributeExpression) listWhere(values valueList) (string, []any) {
	op := e.Operator
	if op == NotEqual {
		op = Equal
	}

	clauses := make([]string, len(values))
	var args []any

	for i, v := range values {
		one := *e
		one.Operator = op
		one.Value = Value{Data: v}

		clause, a := one.Where()
		clauses[i] = "coalesce((" + clause + "), 0)"
		args = append(args, a...)
	}

	matches := "0"
	if len(clauses) > 0 {
		matches = "(" + strings.Join(clauses, " OR ") + ")"
	}

	if e.Operator == NotEqual {
		return "NOT " + matches, args
	}

	return matches, args
}

// fieldWhere returns an SQL condition for an expression that compares the
// issue's field in column, which has a single value that's never NULL.
func (e *AttributeExpression) fieldWhere(column string) (string, []any) {
	if e.Value.Data == nil {
		if e.Operator == NotEqual {
			return column + " IS NOT NULL", nil
		}

		return column + " IS NULL", nil
	}

	if e.Operator == NotEqual {
		equal := *e
		equal.Operator = Equal
		condition, arg := equal.compare(column)

		// A number compared with text that isn't a number is NULL, which
		// NOT would leave NULL, so != wouldn't be the opposite of =.
		return "NOT coalesce(" + condition + ", 0)", []any{arg}
	}

	condition, arg := e.compare(column)

	return condition, []any{arg}
}

// compare returns an SQL condition that compares the text value with the
// expression's value, other than null, using the expression's operator other
// than !=, and the parameter for its placeholder.
func (e *AttributeExpression) compare(value string) (string, any) {
	switch data := e.Value.Data.(type) {
	case string:
		if e.Operator == Match {
			return regexpFunction + "(?, " + value + ")", data
		}

		return value + " " + string(e.Operator) + " ?", data

	case Regexp:
		return regexpFunction + "(?, " + value + ")", string(data)

	case json.Number:
		n, _ := data.Float64()
		return numberFunction + "(" + value + ") " + string(e.Operator) + " ?", n

	case bool:
		return value + " " + string(e.Operator) + " ?", strconv.FormatBool(data)
	}

	return "0", nil
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
