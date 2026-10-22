package filter

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// LookupFunc returns the value at an object lookup's path, such as
// .user.email, from one snapshot of the application's state, or reports that
// there's none. Values are strings, int64s, float64s, bools, []any for
// arrays, and map[string]any for tables, as config.Config.Lookup returns.
type LookupFunc func(path string) (any, bool)

// unknown is a lookup's value when nothing was found at its path. An
// expression comparing with it is UNKNOWN, as in SQL: it matches nothing on
// its own, and an or with it matches what the other side matches.
type unknown struct{}

// valueList is a lookup's value when it's an array. An expression comparing
// with it matches if the comparison holds for any of the values, except !=,
// which matches if = holds for none of them.
type valueList []any

// TypeError reports a looked-up value that can't be compared as the
// expression compares it.
type TypeError struct {
	Path    Lookup
	Message string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("%s %s", e.Path, e.Message)
}

// Resolve returns a copy of node with each object lookup replaced by its
// value from lookup, which should read a single snapshot of the
// application's state, so every value comes from the same moment. node isn't
// changed, so it can be resolved again with new values. A value is checked as
// a literal is when it's parsed: for example, ~ needs a string or a regular
// expression, and true and false can only be compared with = and !=.
func Resolve(node Node, lookup LookupFunc) (Node, error) {
	switch n := node.(type) {
	case *AttributeExpression:
		path, ok := n.Value.Data.(Lookup)
		if !ok {
			return n, nil
		}

		data, err := resolveValue(n.Operator, path, lookup)
		if err != nil {
			return nil, err
		}

		resolved := *n
		resolved.Value = Value{Raw: n.Value.Raw, Data: data}

		return &resolved, nil

	case *AndExpression:
		left, right, err := resolveBoth(n.LeftExpression, n.RightExpression, lookup)
		if err != nil {
			return nil, err
		}

		return &AndExpression{LeftExpression: left, RightExpression: right, Parentheses: n.Parentheses}, nil

	case *OrExpression:
		left, right, err := resolveBoth(n.LeftExpression, n.RightExpression, lookup)
		if err != nil {
			return nil, err
		}

		return &OrExpression{LeftExpression: left, RightExpression: right, Parentheses: n.Parentheses}, nil

	case *NotExpression:
		expr, err := Resolve(n.Expression, lookup)
		if err != nil {
			return nil, err
		}

		return &NotExpression{Expression: expr, Parentheses: n.Parentheses}, nil
	}

	return node, nil
}

func resolveBoth(left, right Node, lookup LookupFunc) (Node, Node, error) {
	l, err := Resolve(left, lookup)
	if err != nil {
		return nil, nil, err
	}

	r, err := Resolve(right, lookup)
	if err != nil {
		return nil, nil, err
	}

	return l, r, nil
}

// resolveValue returns the data for the value at path, compared with op.
func resolveValue(op Operator, path Lookup, lookup LookupFunc) (any, error) {
	if lookup == nil {
		return unknown{}, nil
	}

	v, ok := lookup(string(path))
	if !ok {
		return unknown{}, nil
	}

	if array, ok := v.([]any); ok {
		list := make(valueList, len(array))

		for i, e := range array {
			data, err := scalarData(op, path, e)
			if err != nil {
				return nil, err
			}

			list[i] = data
		}

		return list, nil
	}

	return scalarData(op, path, v)
}

// scalarData returns the data for a looked-up value that isn't an array,
// checked for comparing with op.
func scalarData(op Operator, path Lookup, v any) (any, error) {
	var data any

	switch v := v.(type) {
	case string:
		data = v
	case bool:
		data = v
	case int64:
		data = json.Number(strconv.FormatInt(v, 10))
	case int:
		data = json.Number(strconv.Itoa(v))
	case float64:
		data = json.Number(strconv.FormatFloat(v, 'g', -1, 64))
	case map[string]any:
		return nil, &TypeError{path, "is a table, which can't be compared"}
	case []any:
		return nil, &TypeError{path, "has an array in an array, which can't be compared"}
	default:
		return nil, &TypeError{path, fmt.Sprintf("is a %T, which can't be compared", v)}
	}

	if err := checkOperand(op, Value{Raw: fmt.Sprint(v), Data: data}, 0, 0); err != nil {
		return nil, &TypeError{path, "is " + strconv.Quote(fmt.Sprint(v)) + ": " + err.(*SyntaxError).Message}
	}

	return data, nil
}
