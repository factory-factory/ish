package filter

import (
	"encoding/json"
	"image/color"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

var ansiEscapes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestRenderMatchesString(t *testing.T) {
	texts := []string{
		"{state != 'closed'}",
		`({state = "open"} or {done = true}) and not {a = null}`,
		"(({a ~ /^in\\s/}))",
		"{estimate >= -1.50e3} and {b <= 0} or not ({c = false} and {d > 2})",
	}

	for _, text := range texts {
		node := mustParse(t, text)

		if got := ansiEscapes.ReplaceAllString(node.Render(), ""); got != node.String() {
			t.Errorf("Render() of %q without colors = %q, want String() %q", text, got, node.String())
		}
	}

	// Parentheses added to trees built in code are rendered too.
	a := attr("a", Equal, "1", json.Number("1"))
	b := attr("b", Equal, "2", json.Number("2"))
	built := &NotExpression{Expression: &OrExpression{LeftExpression: a, RightExpression: b}}

	if got := ansiEscapes.ReplaceAllString(built.Render(), ""); got != built.String() {
		t.Errorf("Render() of a built tree without colors = %q, want %q", got, built.String())
	}
	if !strings.Contains(built.Render(), styles.FilterParenthesis.Render("(")) {
		t.Error("added parenthesis isn't colored")
	}
}

func TestRenderColorsEachToken(t *testing.T) {
	rendered := mustParse(t, `({state != 'closed'} and not {done = true}) or {owner = null} or {estimate > 1.5} or {state ~ /^in/}`).Render()

	for _, tt := range []struct {
		name  string
		style lipgloss.Style
		token string
	}{
		{"brace", styles.FilterBrace, "{"},
		{"closing brace", styles.FilterBrace, "}"},
		{"attribute", styles.FilterAttribute, "state"},
		{"operator", styles.FilterOperator, "!="},
		{"two-character operator", styles.FilterOperator, ">"},
		{"match operator", styles.FilterOperator, "~"},
		{"string", styles.FilterValue, "'closed'"},
		{"number", styles.FilterValue, "1.5"},
		{"regular expression", styles.FilterValue, "/^in/"},
		{"boolean", styles.FilterBoolean, "true"},
		{"null", styles.FilterNull, "null"},
		{"parenthesis", styles.FilterParenthesis, "("},
		{"closing parenthesis", styles.FilterParenthesis, ")"},
		{"and", styles.FilterKeyword, "and"},
		{"or", styles.FilterKeyword, "or"},
		{"not", styles.FilterKeyword, "not"},
	} {
		if !strings.Contains(rendered, tt.style.Render(tt.token)) {
			t.Errorf("%s %q isn't rendered in its style in %q", tt.name, tt.token, rendered)
		}
	}
}

func TestHighlightStyles(t *testing.T) {
	for _, tt := range []struct {
		name  string
		style lipgloss.Style
		color color.Color
		bold  bool
	}{
		{"braces are blue", styles.FilterBrace, lipgloss.Color("4"), false},
		{"attributes are magenta", styles.FilterAttribute, lipgloss.Color("5"), false},
		{"operators are bold bright white", styles.FilterOperator, lipgloss.Color("15"), true},
		{"strings and numbers are green", styles.FilterValue, lipgloss.Color("2"), false},
		{"booleans are bold green", styles.FilterBoolean, lipgloss.Color("2"), true},
		{"null is grey", styles.FilterNull, lipgloss.Color("8"), false},
		{"parentheses are yellow", styles.FilterParenthesis, lipgloss.Color("3"), false},
		{"logical operators are white", styles.FilterKeyword, lipgloss.Color("7"), false},
	} {
		if tt.style.GetForeground() != tt.color || tt.style.GetBold() != tt.bold {
			t.Errorf("%s: foreground %v, bold %v, want %v, %v",
				tt.name, tt.style.GetForeground(), tt.style.GetBold(), tt.color, tt.bold)
		}
	}
}
