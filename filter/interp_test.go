package filter

import (
	"strings"
	"testing"

	"github.com/factory-factory/ish/styles"
)

func TestInterpolatedStringsPrintAsWritten(t *testing.T) {
	for _, text := range []string{
		`{a = 'issue \(.count)'}`,
		`{a = "x\(1 + 2)y"}`,
		`{a = 'a\(('b' :upper))c'}`,
		`{a = '\(.user.email :lower)'}`,
		`{a = 'tab\t\(.count)\n'}`,
		`{a = (['x', 'y \(.count)'] :join(','))}`,
		`{a ~ '^\(.pattern)'}`,
	} {
		if got := mustParse(t, text).String(); got != text {
			t.Errorf("String() = %q, want %q", got, text)
		}
	}

	// A string without \( is a literal, and \\( is a backslash and a
	// parenthesis.
	for text, want := range map[string]string{
		`{a = 'plain'}`:   "plain",
		`{a = 'a\\(b)'}`:  `a\(b)`,
		`{a = "q\"\\(x"}`: `q"\(x`,
	} {
		if got := mustParse(t, text).(*AttributeExpression).Value.Data; got != want {
			t.Errorf("Parse(%q) value = %#v, want the literal %q", text, got, want)
		}
	}
}

func TestInterpolation(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		{`('n=\(.count)')`, `"n=5"`},
		{`('\(.ratio) and \(.flag)')`, `"2.5 and true"`},
		{`('\(.user.email :upper)!')`, `"ALICE@EXAMPLE.COM!"`},
		{`('\(1 + 2)\(3 * 4)')`, `"312"`},
		{`('a\(('b' :upper))c')`, `"aBc"`},
		{`('tab\t\(.count)\n')`, `"tab\t5\n"`},
		// null is inserted as nothing.
		{`('[\(.missing)]')`, `"[]"`},
		{`('[\(1 / 0)]')`, `"[]"`},
		// An interpolated string is a string.
		{`('\(.count)' :upper)`, `"5"`},
		{`('a b \(.count)' :split :count)`, "3"},
	} {
		if got, err := evalText(t, tt.text); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v, want %s", tt.text, got, err, tt.want)
		}
	}

	// A list can't be inserted.
	if _, err := evalText(t, `('\(.team)')`); err == nil || !strings.Contains(err.Error(), "can't insert a list in a string; use :join") {
		t.Errorf("inserting a list error = %v, want one suggesting :join", err)
	}
	if got, err := evalText(t, `('\(.team :join(', '))')`); err != nil || got != `"bob, carol"` {
		t.Errorf("inserting a joined list = %s, %v, want the names", got, err)
	}
}

func TestInterpolationSyntaxErrors(t *testing.T) {
	for _, tt := range []struct {
		text       string
		start, end int
		message    string
	}{
		{`{a = 'x\()'}`, 9, 12, "expected a value"},
		{`{a = 'x\(1 + 2'}`, 14, 16, "expected an operator, a :function, or a closing parenthesis"},
		{`{a = 'x\(1)`, 5, 6, "this string is never closed"},
		{`{a = 'x\(:lower)'}`, 9, 18, "expected a value"},
	} {
		_, err := Parse(tt.text)
		syntaxErr, ok := err.(*SyntaxError)

		if !ok || syntaxErr.Start != tt.start || syntaxErr.End != tt.end || !strings.Contains(syntaxErr.Message, tt.message) {
			t.Errorf("Parse(%q) error = %#v, want [%d, %d) containing %q", tt.text, err, tt.start, tt.end, tt.message)
		}
	}
}

func TestRenderInterpolation(t *testing.T) {
	node := mustParse(t, `{a = 'id \(.user.email :lower)!'}`)
	rendered := node.Render()

	if got := ansi.ReplaceAllString(rendered, ""); got != node.String() {
		t.Errorf("Render() without colors = %q, want %q", got, node.String())
	}

	for _, tt := range []struct {
		name, style string
	}{
		{"text", styles.FilterValue.Render("id ")},
		{"opening", styles.FilterParenthesis.Render(`\(`)},
		{"closing", styles.FilterParenthesis.Render(")")},
		{"lookup", styles.FilterLookup.Render(".user.email")},
		{"function", styles.FilterLookup.Render(":lower")},
	} {
		if !strings.Contains(rendered, tt.style) {
			t.Errorf("%s isn't styled in %q", tt.name, rendered)
		}
	}
}
