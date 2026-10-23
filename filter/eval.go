package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Expression values are nil for null, a string, a float64, a bool, or a
// []any of values for a list. Between the SQL functions that evaluate
// expressions, they're passed as JSON text, which keeps their types, so a
// mismatch is an error rather than a conversion.

// typeName names the type of an expression value, for error messages.
func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "a string"
	case float64:
		return "a number"
	case bool:
		return "a boolean"
	case []any:
		return "a list"
	}

	return fmt.Sprintf("a %T", v)
}

// encodeValue returns v as JSON text.
func encodeValue(v any) string {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(numbersAsJSON(v)); err != nil {
		return "null"
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// numbersAsJSON returns v with its numbers as json.Numbers, written without
// exponents or trailing zeros where possible, so 3 is written as 3, not 3e+00.
func numbersAsJSON(v any) any {
	switch v := v.(type) {
	case float64:
		return json.Number(formatNumber(v))
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = numbersAsJSON(e)
		}
		return out
	}

	return v
}

// formatNumber returns n as it's written in JSON and in text.
func formatNumber(n float64) string {
	if n == math.Trunc(n) && math.Abs(n) < 1e15 {
		return strconv.FormatInt(int64(n), 10)
	}

	return strconv.FormatFloat(n, 'g', -1, 64)
}

// decodeValue returns the expression value in JSON text.
func decodeValue(text string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}

	return fromJSON(v), nil
}

// fromJSON returns a decoded JSON value as an expression value, with its
// numbers as float64s.
func fromJSON(v any) any {
	switch v := v.(type) {
	case json.Number:
		n, _ := v.Float64()
		return n
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = fromJSON(e)
		}
		return out
	}

	return v
}

// equalValues reports whether a and b are the same value, as JSON.
func equalValues(a, b any) bool {
	return encodeValue(a) == encodeValue(b)
}

// evalUnary applies a unary operator: - negates a number, and ! negates a
// boolean. Applied to null, it gives null, as in SQL.
func evalUnary(op string, x any) (any, error) {
	if x == nil {
		return nil, nil
	}

	switch op {
	case "-":
		if n, ok := x.(float64); ok {
			return -n, nil
		}
	case "!":
		if b, ok := x.(bool); ok {
			return !b, nil
		}
	}

	return nil, fmt.Errorf("%s can't be applied to %s", op, typeName(x))
}

// evalBinary applies a binary operator. Arithmetic needs two numbers, and
// division is exact; dividing by zero gives null, as in SQL. = and != compare
// values of the same type, and <, <=, >, and >= compare two numbers or two
// strings. Either side being null gives null, as in SQL.
func evalBinary(op string, a, b any) (any, error) {
	if a == nil || b == nil {
		return nil, nil
	}

	switch op {
	case "+", "-", "*", "/", "%", "**":
		x, xok := a.(float64)
		y, yok := b.(float64)

		if !xok || !yok {
			return nil, fmt.Errorf("%s needs two numbers, not %s and %s", op, typeName(a), typeName(b))
		}

		var n float64

		switch op {
		case "+":
			n = x + y
		case "-":
			n = x - y
		case "*":
			n = x * y
		case "/":
			if y == 0 {
				return nil, nil
			}
			n = x / y
		case "%":
			if y == 0 {
				return nil, nil
			}
			n = math.Mod(x, y)
		case "**":
			n = math.Pow(x, y)
		}

		if math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, nil
		}

		return n, nil

	case "=", "!=":
		if typeName(a) != typeName(b) {
			return nil, fmt.Errorf("%s can't compare %s with %s", op, typeName(a), typeName(b))
		}

		return equalValues(a, b) == (op == "="), nil

	case "<", "<=", ">", ">=":
		c, err := order(op, a, b)
		if err != nil {
			return nil, err
		}

		switch op {
		case "<":
			return c < 0, nil
		case "<=":
			return c <= 0, nil
		case ">":
			return c > 0, nil
		default:
			return c >= 0, nil
		}
	}

	return nil, fmt.Errorf("unknown operator %s", op)
}

// order compares two numbers or two strings, returning a negative number,
// zero, or a positive number as a is less than, equal to, or greater than b.
func order(op string, a, b any) (int, error) {
	switch x := a.(type) {
	case float64:
		if y, ok := b.(float64); ok {
			return compareFloat(x, y), nil
		}
	case string:
		if y, ok := b.(string); ok {
			return strings.Compare(x, y), nil
		}
	}

	return 0, fmt.Errorf("%s needs two numbers or two strings, not %s and %s", op, typeName(a), typeName(b))
}

func compareFloat(x, y float64) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}

	return 0
}

// interpolate joins the parts of an interpolated string: text, and the values
// of inserted expressions, converted to strings. Strings are inserted as they
// are, numbers as they're written in JSON, and booleans as true or false.
// null is inserted as nothing, and a list is an error.
func interpolate(values []any) (any, error) {
	var b strings.Builder

	for _, v := range values {
		switch v := v.(type) {
		case nil:
		case string:
			b.WriteString(v)
		case float64:
			b.WriteString(formatNumber(v))
		case bool:
			b.WriteString(strconv.FormatBool(v))
		default:
			return nil, fmt.Errorf("can't insert %s in a string; use :join to make a list a string", typeName(v))
		}
	}

	return b.String(), nil
}

// function is a function in a filter expression: the number of arguments it
// takes after the value it's applied to, and its implementation, which is
// given the value first, then the arguments. Every function gives null for
// a null value.
type function struct {
	minArgs, maxArgs int
	apply            func(args []any) (any, error)
}

// functions are the functions filter expressions can apply, by name.
var functions = map[string]function{
	"lower":    {0, 0, strFunc(strings.ToLower)},
	"upper":    {0, 0, strFunc(strings.ToUpper)},
	"join":     {1, 1, join},
	"split":    {0, 1, split},
	"replace":  {2, 2, replace},
	"trim":     {0, 1, trim},
	"count":    {0, 0, count},
	"distinct": {0, 0, distinct},
	"includes": {1, 1, includes},
	"indexof":  {1, 1, indexOf},
	"abs":      {0, 0, numFunc(math.Abs)},
	"ceil":     {0, 0, numFunc(math.Ceil)},
	"floor":    {0, 0, numFunc(math.Floor)},
	"round":    {0, 1, round},
	"max":      {0, 0, extreme(1)},
	"min":      {0, 0, extreme(-1)},

	// :user is resolved before a query is made, from the user directory;
	// see CallExpr.resolve. Unresolved, it finds no one.
	userFunction: {0, 0, func([]any) (any, error) { return nil, nil }},
}

// userFunction looks up users' emails by their display names.
const userFunction = "user"

// lookUpUsers returns the email of the user in users with the display name v,
// or a list of emails for a list of names. A leading @, as names are shown
// with, is ignored. A name matches exactly, or else regardless of case, if
// only one user's does. A name that matches no one gives null, as does one
// matching several users regardless of case.
func lookUpUsers(v any, users []User) (any, error) {
	switch v := v.(type) {
	case nil:
		return nil, nil

	case string:
		return lookUpUser(v, users), nil

	case []any:
		emails := make([]any, len(v))

		for i, e := range v {
			name, ok := e.(string)
			if !ok && e != nil {
				return nil, fmt.Errorf("needs names, not a list with %s", typeName(e))
			}

			if ok {
				emails[i] = lookUpUser(name, users)
			}
		}

		return emails, nil
	}

	return nil, fmt.Errorf("needs a name or a list of names, not %s", typeName(v))
}

// lookUpUser returns the email of the user with the display name name, or
// nil if there isn't exactly one.
func lookUpUser(name string, users []User) any {
	name = strings.TrimPrefix(name, "@")

	var match any
	matches := 0

	for _, u := range users {
		if u.Name == name {
			return u.Email
		}

		if strings.EqualFold(u.Name, name) {
			match = u.Email
			matches++
		}
	}

	if matches != 1 {
		return nil
	}

	return match
}

// applyFunction applies the named function to args: the value, then the
// function's arguments.
func applyFunction(name string, args []any) (any, error) {
	f, ok := functions[name]
	if !ok {
		return nil, fmt.Errorf("unknown function :%s", name)
	}

	if n := len(args) - 1; n < f.minArgs || n > f.maxArgs {
		return nil, fmt.Errorf(":%s takes %s", name, arity(f))
	}

	if args[0] == nil {
		return nil, nil
	}

	v, err := f.apply(args)
	if err != nil {
		return nil, fmt.Errorf(":%s %w", name, err)
	}

	return v, nil
}

// arity describes how many arguments a function takes, for error messages.
func arity(f function) string {
	switch {
	case f.maxArgs == 0:
		return "no arguments"
	case f.minArgs == f.maxArgs && f.minArgs == 1:
		return "1 argument"
	case f.minArgs == f.maxArgs:
		return fmt.Sprintf("%d arguments", f.minArgs)
	case f.minArgs == 0 && f.maxArgs == 1:
		return "an optional argument"
	}

	return fmt.Sprintf("%d to %d arguments", f.minArgs, f.maxArgs)
}

// argString returns args[i] if it's a string, describing it as what for an
// error.
func argString(args []any, i int, what string) (string, error) {
	s, ok := args[i].(string)
	if !ok {
		return "", fmt.Errorf("needs %s that's a string, not %s", what, typeName(args[i]))
	}

	return s, nil
}

func argList(args []any) ([]any, error) {
	list, ok := args[0].([]any)
	if !ok {
		return nil, fmt.Errorf("needs a list, not %s", typeName(args[0]))
	}

	return list, nil
}

func argNumber(args []any, i int, what string) (float64, error) {
	n, ok := args[i].(float64)
	if !ok {
		return 0, fmt.Errorf("needs %s that's a number, not %s", what, typeName(args[i]))
	}

	return n, nil
}

func strFunc(f func(string) string) func([]any) (any, error) {
	return func(args []any) (any, error) {
		s, err := argString(args, 0, "a value")
		if err != nil {
			return nil, err
		}

		return f(s), nil
	}
}

func numFunc(f func(float64) float64) func([]any) (any, error) {
	return func(args []any) (any, error) {
		n, err := argNumber(args, 0, "a value")
		if err != nil {
			return nil, err
		}

		return f(n), nil
	}
}

// join joins a list's strings, numbers, and booleans with a separator.
func join(args []any) (any, error) {
	list, err := argList(args)
	if err != nil {
		return nil, err
	}

	sep, err := argString(args, 1, "a separator")
	if err != nil {
		return nil, err
	}

	parts := make([]string, len(list))

	for i, e := range list {
		switch e := e.(type) {
		case string:
			parts[i] = e
		case float64:
			parts[i] = formatNumber(e)
		case bool:
			parts[i] = strconv.FormatBool(e)
		default:
			return nil, fmt.Errorf("can't join %s", typeName(e))
		}
	}

	return strings.Join(parts, sep), nil
}

// split splits a string at a separator, or at runs of whitespace if there's
// none, giving a list of strings.
func split(args []any) (any, error) {
	s, err := argString(args, 0, "a value")
	if err != nil {
		return nil, err
	}

	var parts []string

	if len(args) > 1 {
		sep, err := argString(args, 1, "a separator")
		if err != nil {
			return nil, err
		}

		parts = strings.Split(s, sep)
	} else {
		parts = strings.Fields(s)
	}

	list := make([]any, len(parts))
	for i, p := range parts {
		list[i] = p
	}

	return list, nil
}

// replace replaces every occurrence of some text with other text.
func replace(args []any) (any, error) {
	s, err := argString(args, 0, "a value")
	if err != nil {
		return nil, err
	}

	old, err := argString(args, 1, "text to replace")
	if err != nil {
		return nil, err
	}

	replacement, err := argString(args, 2, "a replacement")
	if err != nil {
		return nil, err
	}

	return strings.ReplaceAll(s, old, replacement), nil
}

// trim removes whitespace from both ends of a string, or the given
// characters if there are any.
func trim(args []any) (any, error) {
	s, err := argString(args, 0, "a value")
	if err != nil {
		return nil, err
	}

	if len(args) > 1 {
		chars, err := argString(args, 1, "characters to remove")
		if err != nil {
			return nil, err
		}

		return strings.Trim(s, chars), nil
	}

	return strings.TrimSpace(s), nil
}

func count(args []any) (any, error) {
	list, err := argList(args)
	if err != nil {
		return nil, err
	}

	return float64(len(list)), nil
}

// distinct removes repeated values from a list, keeping the first of each.
func distinct(args []any) (any, error) {
	list, err := argList(args)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	out := []any{}

	for _, e := range list {
		if key := encodeValue(e); !seen[key] {
			seen[key] = true
			out = append(out, e)
		}
	}

	return out, nil
}

func includes(args []any) (any, error) {
	i, err := indexOf(args)
	if err != nil {
		return nil, err
	}

	return i != nil, nil
}

// indexOf returns the position of a value in a list, from 0, or null if it
// isn't in it.
func indexOf(args []any) (any, error) {
	list, err := argList(args)
	if err != nil {
		return nil, err
	}

	for i, e := range list {
		if equalValues(e, args[1]) {
			return float64(i), nil
		}
	}

	return nil, nil
}

// round rounds a number to a whole number, or to a number of decimal places.
func round(args []any) (any, error) {
	n, err := argNumber(args, 0, "a value")
	if err != nil {
		return nil, err
	}

	places := 0.0

	if len(args) > 1 {
		if places, err = argNumber(args, 1, "a number of places"); err != nil {
			return nil, err
		}

		if places != math.Trunc(places) {
			return nil, fmt.Errorf("needs a whole number of places, not %s", formatNumber(places))
		}
	}

	scale := math.Pow(10, places)

	return math.Round(n*scale) / scale, nil
}

// extreme returns a function giving a list's largest number if sign is 1, or
// its smallest if sign is -1, or null for an empty list.
func extreme(sign int) func([]any) (any, error) {
	return func(args []any) (any, error) {
		list, err := argList(args)
		if err != nil {
			return nil, err
		}

		var best any

		for _, e := range list {
			n, ok := e.(float64)
			if !ok {
				return nil, fmt.Errorf("needs a list of numbers, not one with %s", typeName(e))
			}

			if b, ok := best.(float64); !ok || compareFloat(n, b) == sign {
				best = n
			}
		}

		return best, nil
	}
}

// compareWith reports whether an attribute's value, text as it's stored,
// compares with an expression's value v by op, which isn't !=: as a literal
// does, a string compares as text, or with ~ matches as a regular
// expression; a number compares numerically with values that are numbers;
// a boolean matches "true" or "false"; and a list matches if any of its
// values does. It's nil for a null value, which can't be compared.
func compareWith(op Operator, text string, v any) (any, error) {
	switch v := v.(type) {
	case nil:
		return nil, nil

	case []any:
		for _, e := range v {
			ok, err := compareWith(op, text, e)
			if err != nil {
				return nil, err
			}

			if ok == true {
				return true, nil
			}
		}

		return false, nil

	case string:
		if op == Match {
			re, err := compiledPattern(v)
			if err != nil {
				return nil, err
			}

			return re.MatchString(text), nil
		}

		return compareOrdered(op, strings.Compare(text, v)), nil

	case float64:
		if op == Match {
			return nil, fmt.Errorf("~ needs a regular expression, not a number")
		}

		if numberPattern.FindString(text) != text {
			return false, nil
		}

		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return false, nil
		}

		return compareOrdered(op, compareFloat(n, v)), nil

	case bool:
		if op != Equal {
			return nil, fmt.Errorf("%s can't compare with a boolean", op)
		}

		return text == strconv.FormatBool(v), nil
	}

	return nil, fmt.Errorf("can't compare with %s", typeName(v))
}

func compareOrdered(op Operator, c int) bool {
	switch op {
	case Equal:
		return c == 0
	case Less:
		return c < 0
	case LessOrEqual:
		return c <= 0
	case Greater:
		return c > 0
	case GreaterOrEqual:
		return c >= 0
	}

	return false
}

// compiledPattern returns the compiled regular expression, from the cache
// sqlRegexp uses.
func compiledPattern(pattern string) (*regexp.Regexp, error) {
	if cached, ok := patterns.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}

	cached, _ := patterns.LoadOrStore(pattern, re)

	return cached.(*regexp.Regexp), nil
}
