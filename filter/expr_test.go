package filter

import (
	"regexp"
	"strings"
	"testing"

	"github.com/factory-factory/ish/styles"
)

// evalText parses the expression in text, the value of {a = text}, and
// returns its value as JSON, with lookups from lookupState.
func evalText(t *testing.T, text string) (string, error) {
	t.Helper()

	node, err := Parse("{a = " + text + "}")
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", text, err)
	}

	expr, ok := node.(*AttributeExpression).Value.Data.(*Expression)
	if !ok {
		t.Fatalf("Parse(%q) value = %#v, want an expression", text, node.(*AttributeExpression).Value.Data)
	}

	v, err := expr.Eval(lookupState)
	if err != nil {
		return "", err
	}

	return encodeValue(v), nil
}

func TestExpressionsPrintAsWritten(t *testing.T) {
	for _, text := range []string{
		"(3 + 7)",
		"(.user.email :lower)",
		"([1, 2, 3] :join(','))",
		"(-.count :abs)",
		"(!(.count > 3))",
		"(2 ** 3 ** 2)",
		"((1 + 2) * 3)",
		"('a b' :split :count)",
		"(.team :includes('bob'))",
		"(1 - -2)",
		"(1.5 :round(1))",
		"([] :count)",
		"(null)",
	} {
		filter := "{a = " + text + "}"

		if got := mustParse(t, filter).String(); got != filter {
			t.Errorf("String() = %q, want %q", got, filter)
		}
	}

	// Spacing is normalized.
	if got := mustParse(t, "{a=(1+2*3 :abs)}").String(); got != "{a = (1 + 2 * 3 :abs)}" {
		t.Errorf("String() = %q, want the spacing normalized", got)
	}
}

func TestExpressionPrecedence(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		{"(1 + 2 * 3)", "7"},
		{"((1 + 2) * 3)", "9"},
		{"(10 - 4 - 3)", "3"},
		{"(2 ** 3 ** 2)", "512"},
		// Unary operators bind tightest, then functions.
		{"(-2 ** 2)", "4"},
		{"(-.count :abs)", "5"},
		{"(1 - 2 :abs)", "-1"},
		{"(1 + 2 * 3 > 6)", "true"},
		{"(!(1 < 2))", "false"},
		{"(!true = false)", "true"},
	} {
		if got, err := evalText(t, tt.text); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v, want %s", tt.text, got, err, tt.want)
		}
	}
}

func TestExpressionOperators(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		{"(7 / 2)", "3.5"},
		{"(7 % 3)", "1"},
		{"(2 ** 0.5 ** 2)", "1.189207115002721"},
		{"(.ratio * 2)", "5"},
		// Dividing by zero is null, as in SQL, and so is anything with
		// null.
		{"(1 / 0)", "null"},
		{"(1 % 0)", "null"},
		{"(null + 1)", "null"},
		{"(-null)", "null"},
		{"(.missing + 1)", "null"},
		// Comparisons.
		{"('a' = 'a')", "true"},
		{"([1, 2] = [1, 2])", "true"},
		{"(1 != 2)", "true"},
		{"('a' < 'b')", "true"},
		{"(2 >= 2)", "true"},
		{"(null = 1)", "null"},
	} {
		if got, err := evalText(t, tt.text); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v, want %s", tt.text, got, err, tt.want)
		}
	}
}

func TestExpressionFunctions(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		{"('YELLING' :lower)", `"yelling"`},
		{"(.user.email :upper)", `"ALICE@EXAMPLE.COM"`},
		{"([1, 2, 3] :join(','))", `"1,2,3"`},
		{"(['a', true, 2.5] :join(' '))", `"a true 2.5"`},
		{`(' a  b\tc ' :split)`, `["a","b","c"]`},
		{"('a,b,,c' :split(','))", `["a","b","","c"]`},
		{"('a-b-c' :replace('-', '+'))", `"a+b+c"`},
		{"('  padded  ' :trim)", `"padded"`},
		{"('--dashes--' :trim('-'))", `"dashes"`},
		{"(.team :count)", "2"},
		{"([1, 2, 1, 3, 2] :distinct)", "[1,2,3]"},
		{"(.team :includes('carol'))", "true"},
		{"(.team :includes('dave'))", "false"},
		{"(.team :indexof('carol'))", "1"},
		{"(.team :indexof('dave'))", "null"},
		{"(-3.5 :abs)", "3.5"},
		{"(3.2 :ceil)", "4"},
		{"(3.8 :floor)", "3"},
		{"(2.5 :round)", "3"},
		{"(3.14159 :round(2))", "3.14"},
		{"([3, 9, 1] :max)", "9"},
		{"([3, 9, 1] :min)", "1"},
		{"([] :max)", "null"},
		{"(null :lower)", "null"},
		// Functions chain.
		{"(' A B ' :trim :lower :split :join('-'))", `"a-b"`},
		// Arguments are expressions.
		{"(['a', 'b'] :join(('+' :upper)))", `"a+b"`},
	} {
		if got, err := evalText(t, tt.text); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v, want %s", tt.text, got, err, tt.want)
		}
	}
}

func TestExpressionTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		text, message string
	}{
		{"('a' + 1)", "+ needs two numbers, not a string and a number"},
		{"(-'a')", "- can't be applied to a string"},
		{"(!1)", "! can't be applied to a number"},
		{"(1 = 'a')", "= can't compare a number with a string"},
		{"('a' < 1)", "< needs two numbers or two strings"},
		{"(5 :lower)", ":lower needs a value that's a string, not a number"},
		{"('a' :count)", ":count needs a list, not a string"},
		{"([1, 'a'] :max)", ":max needs a list of numbers"},
		{"([[1]] :join(','))", ":join can't join a list"},
		{"(1.5 :round(0.5))", ":round needs a whole number of places"},
		{"(.table :lower)", ".table is a table"},
	} {
		_, err := evalText(t, tt.text)

		if err == nil || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("%s error = %v, want one containing %q", tt.text, err, tt.message)
		}
	}
}

func TestExpressionSyntaxErrors(t *testing.T) {
	for _, tt := range []struct {
		text       string
		start, end int
		message    string
	}{
		{"{a = (1 +)}", 9, 11, "expected a value"},
		{"{a = (1 + 2", 5, 6, "this parenthesis is never closed"},
		{"{a = (1 + 2}", 11, 12, "expected an operator, a :function, or a closing parenthesis"},
		{"{a = (1 2)}", 8, 11, "expected an operator, a :function, or a closing parenthesis"},
		{"{a = (1 < 2 < 3)}", 12, 13, "comparisons can't be chained"},
		{"{a = ('a' :shout)}", 10, 16, "unknown function :shout"},
		{"{a = ('a' :lower('b'))}", 10, 21, ":lower takes no arguments"},
		{"{a = (['a'] :join)}", 12, 17, ":join takes 1 argument"},
		{"{a = ('a' :replace('b'))}", 10, 23, ":replace takes 2 arguments"},
		{"{a = ('a' :)}", 10, 11, "expected a function name"},
		{"{a = ([1, 2)}", 11, 13, "expected a comma or a closing bracket"},
		{"{a = ([1, 2", 6, 7, "this list is never closed"},
		{"{a = (/x/)}", 6, 11, "a regular expression literal can only follow ~ directly"},
	} {
		_, err := Parse(tt.text)
		syntaxErr, ok := err.(*SyntaxError)

		if !ok || syntaxErr.Start != tt.start || syntaxErr.End != tt.end || !strings.Contains(syntaxErr.Message, tt.message) {
			t.Errorf("Parse(%q) error = %#v, want [%d, %d) containing %q", tt.text, err, tt.start, tt.end, tt.message)
		}
	}
}

func TestResolveExpressionErrors(t *testing.T) {
	for _, tt := range []struct {
		text, message string
	}{
		// The value is checked against the comparison, as a literal is.
		{"{done < (1 = 1)}", "< can't compare with a boolean"},
		{"{a ~ (2 + 3)}", "~ needs a regular expression, not a number"},
		{"{a ~ ('(' :lower)}", "invalid regular expression"},
		// And the expression is evaluated.
		{"{a = ('x' + 1)}", "('x' + 1): + needs two numbers"},
	} {
		_, err := Resolve(mustParse(t, tt.text), lookupState)

		if _, ok := err.(*ExprError); !ok || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("Resolve(%q) error = %v, want an expression error containing %q", tt.text, err, tt.message)
		}
	}

	// A table in an expression is a type error.
	if _, err := Resolve(mustParse(t, "{a = (.table :count)}"), lookupState); err == nil || !strings.Contains(err.Error(), ".table is a table") {
		t.Errorf("Resolve() with a table error = %v, want a type error", err)
	}
}

func TestResolveNullExpressionIsUnknown(t *testing.T) {
	resolved, err := Resolve(mustParse(t, "{a = (.missing :lower)}"), lookupState)
	if err != nil {
		t.Fatal(err)
	}

	if clause, _ := resolved.Where(); clause != "NULL" {
		t.Errorf("Where() = %q, want NULL", clause)
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestRenderExpression(t *testing.T) {
	node := mustParse(t, "{a = ((.user.email :lower) + [1, -2] :join(','))}")
	rendered := node.Render()

	if got := ansi.ReplaceAllString(rendered, ""); got != node.String() {
		t.Errorf("Render() without colors = %q, want %q", got, node.String())
	}

	for _, tt := range []struct {
		name  string
		style string
	}{
		{"function", styles.FilterLookup.Render(":lower")},
		{"lookup", styles.FilterLookup.Render(".user.email")},
		{"parenthesis", styles.FilterParenthesis.Render("(")},
		{"bracket", styles.FilterParenthesis.Render("[")},
		{"operator", styles.FilterOperator.Render("+")},
		{"unary operator", styles.FilterOperator.Render("-")},
		{"string", styles.FilterValue.Render("','")},
	} {
		if !strings.Contains(rendered, tt.style) {
			t.Errorf("%s isn't styled in %q", tt.name, rendered)
		}
	}
}
