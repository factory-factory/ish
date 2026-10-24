package filter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
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
	return ResolveEnv(node, Env{Lookup: lookup})
}

// Env is what Resolve resolves expressions with: Lookup reads a single
// snapshot of the application's state, and Users is the user directory,
// which the :user function looks names up in.
type Env struct {
	Lookup LookupFunc
	Users  []User

	// Now is the moment :fromnow counts from, in the time zone it counts
	// calendar days in. If it's zero, it's the current time, in the local
	// time zone.
	Now time.Time
}

// now returns the moment :fromnow counts from.
func (env Env) now() time.Time {
	if env.Now.IsZero() {
		return time.Now()
	}

	return env.Now
}

// User is a user in the directory: their display name and email.
type User struct {
	Name  string
	Email string
}

// ResolveEnv is Resolve with the user directory and the clock too. See
// Resolve. The clock is read once, so every :fromnow in a query counts from
// the same moment.
func ResolveEnv(node Node, env Env) (Node, error) {
	if env.Now.IsZero() {
		env.Now = time.Now()
	}

	return resolveEnv(node, env)
}

func resolveEnv(node Node, env Env) (Node, error) {
	switch n := node.(type) {
	case *AttributeExpression:
		if expr, ok := n.Value.Data.(*Expression); ok {
			data, err := resolveExpression(n.Operator, expr, env)
			if err != nil {
				return nil, err
			}

			resolved := *n
			resolved.Value = Value{Data: data}

			return &resolved, nil
		}

		path, ok := n.Value.Data.(Lookup)
		if !ok {
			return n, nil
		}

		data, err := resolveValue(n.Operator, path, env)
		if err != nil {
			return nil, err
		}

		resolved := *n
		resolved.Value = Value{Raw: n.Value.Raw, Data: data}

		return &resolved, nil

	case *ScopeExpression:
		condition, err := resolveCondition(n.Condition, env)
		if err != nil {
			return nil, err
		}

		resolved := *n
		resolved.Condition = condition

		return &resolved, nil

	case *AndExpression:
		left, right, err := resolveBoth(n.LeftExpression, n.RightExpression, env)
		if err != nil {
			return nil, err
		}

		return &AndExpression{LeftExpression: left, RightExpression: right, Parentheses: n.Parentheses}, nil

	case *OrExpression:
		left, right, err := resolveBoth(n.LeftExpression, n.RightExpression, env)
		if err != nil {
			return nil, err
		}

		return &OrExpression{LeftExpression: left, RightExpression: right, Parentheses: n.Parentheses}, nil

	case *NotExpression:
		expr, err := resolveEnv(n.Expression, env)
		if err != nil {
			return nil, err
		}

		return &NotExpression{Expression: expr, Parentheses: n.Parentheses}, nil
	}

	return node, nil
}

func resolveBoth(left, right Node, env Env) (Node, Node, error) {
	l, err := resolveEnv(left, env)
	if err != nil {
		return nil, nil, err
	}

	r, err := resolveEnv(right, env)
	if err != nil {
		return nil, nil, err
	}

	return l, r, nil
}

// resolveExpression returns the data for an expression compared with op: the
// expression with its lookups resolved, or unknown if its value is null, as
// it is when it uses a lookup that found nothing. The expression is evaluated
// here, to report errors such as a function applied to a value of the wrong
// type, and its value is checked as a literal is.
func resolveExpression(op Operator, expr *Expression, env Env) (any, error) {
	root, err := expr.Root.resolve(env)
	if err != nil {
		return nil, err
	}

	resolved := &Expression{Root: root}

	v, err := resolved.Root.eval(env)
	if err != nil {
		return nil, &ExprError{Expression: expr.format(plain), Err: err}
	}

	if v == nil {
		return unknown{}, nil
	}

	if op == NotEqual {
		op = Equal
	}

	if _, err := compareWith(op, "", v); err != nil {
		return nil, &ExprError{Expression: expr.format(plain), Err: err}
	}

	return resolved, nil
}

// resolveValue returns the data for the value at path, compared with op.
func resolveValue(op Operator, path Lookup, env Env) (any, error) {
	if env.Lookup == nil {
		return unknown{}, nil
	}

	v, ok := env.Lookup(string(path))
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

// resolveCondition resolves the lookups and expressions in a scope's
// condition, as resolveEnv does an attribute expression's.
func resolveCondition(c Condition, env Env) (Condition, error) {
	switch c := c.(type) {
	case *Comparison:
		resolved := *c

		switch data := c.Value.Data.(type) {
		case *Expression:
			v, err := resolveExpression(c.Operator, data, env)
			if err != nil {
				return nil, err
			}

			resolved.Value = Value{Data: v}

		case Lookup:
			v, err := resolveValue(c.Operator, data, env)
			if err != nil {
				return nil, err
			}

			resolved.Value = Value{Raw: c.Value.Raw, Data: v}
		}

		return &resolved, nil

	case *ConditionAnd:
		left, err := resolveCondition(c.Left, env)
		if err != nil {
			return nil, err
		}

		right, err := resolveCondition(c.Right, env)
		if err != nil {
			return nil, err
		}

		return &ConditionAnd{Left: left, Right: right}, nil

	case *ConditionNot:
		x, err := resolveCondition(c.X, env)
		if err != nil {
			return nil, err
		}

		return &ConditionNot{X: x}, nil

	case *ConditionGroup:
		x, err := resolveCondition(c.X, env)
		if err != nil {
			return nil, err
		}

		return &ConditionGroup{X: x}, nil
	}

	return c, nil
}
