package filter

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/factory-factory/ish/styles"
)

func TestParseLookups(t *testing.T) {
	for _, tt := range []struct {
		text string
		want Node
	}{
		{"{assignees = .user.email}", attr("assignees", Equal, ".user.email", Lookup(".user.email"))},
		{"{a ~ .patterns.0}", attr("a", Match, ".patterns.0", Lookup(".patterns.0"))},
		{"{a<.b_c-d}", attr("a", Less, ".b_c-d", Lookup(".b_c-d"))},
		{"{@author != .user.email}", attr("@author", NotEqual, ".user.email", Lookup(".user.email"))},
	} {
		if got := mustParse(t, tt.text); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Parse(%q) = %#v, want %#v", tt.text, got, tt.want)
		}
	}

	// Printed as they're written.
	for _, text := range []string{"{assignees = .user.email}", "({a = .x} or not {b != .y.z})"} {
		if got := mustParse(t, text).String(); got != text {
			t.Errorf("String() = %q, want %q", got, text)
		}
	}
}

func TestLookupErrors(t *testing.T) {
	for _, tt := range []struct {
		text       string
		start, end int
		message    string
	}{
		{"{a = .}", 5, 6, "expected a key after the dot"},
		{"{a = .user.}", 10, 11, "expected a key after the dot"},
		{"{a = .user..email}", 10, 11, "expected a key after the dot"},
		{"{a = .us$er}", 8, 9, "keys may only contain letters, digits, _, and -"},
	} {
		_, err := Parse(tt.text)
		syntaxErr, ok := err.(*SyntaxError)

		if !ok || syntaxErr.Start != tt.start || syntaxErr.End != tt.end || !strings.Contains(syntaxErr.Message, tt.message) {
			t.Errorf("Parse(%q) error = %#v, want [%d, %d) containing %q", tt.text, err, tt.start, tt.end, tt.message)
		}
	}
}

func TestRenderLookup(t *testing.T) {
	rendered := mustParse(t, "{assignees = .user.email}").Render()

	if !strings.Contains(rendered, styles.FilterLookup.Render(".user.email")) {
		t.Errorf("Render() = %q, want the lookup in its style", rendered)
	}
	if got := styles.FilterLookup.GetForeground(); got != styles.Mention.GetForeground() {
		t.Errorf("lookup color = %v, want aqua (6)", got)
	}
}

// state is the application state lookups are resolved from in tests.
var state = map[string]any{
	".user.email": "alice@example.com",
	".team":       []any{"bob", "carol"},
	".empty":      []any{},
	".count":      int64(5),
	".ratio":      2.5,
	".flag":       true,
	".pattern":    "^in ",
	".bad":        "(",
	".table":      map[string]any{"x": "y"},
	".nested":     []any{"a", map[string]any{"x": "y"}},
	".numbers":    []any{int64(1), int64(2)},
}

func lookupState(path string) (any, bool) {
	v, ok := state[path]
	return v, ok
}

func TestResolve(t *testing.T) {
	for _, tt := range []struct {
		text string
		want any
	}{
		{"{a = .user.email}", "alice@example.com"},
		{"{a = .count}", json.Number("5")},
		{"{a > .ratio}", json.Number("2.5")},
		{"{a = .flag}", true},
		{"{a ~ .pattern}", "^in "},
		{"{a = .team}", valueList{"bob", "carol"}},
		{"{a = .empty}", valueList{}},
		{"{a = .missing}", unknown{}},
	} {
		node := mustParse(t, tt.text)
		resolved, err := Resolve(node, lookupState)

		if err != nil {
			t.Errorf("Resolve(%q) error = %v", tt.text, err)
			continue
		}

		got := resolved.(*AttributeExpression).Value

		if !reflect.DeepEqual(got.Data, tt.want) {
			t.Errorf("Resolve(%q) = %#v, want %#v", tt.text, got.Data, tt.want)
		}

		// It still prints as written, and the parsed node is unchanged.
		if resolved.String() != tt.text || node.String() != tt.text {
			t.Errorf("Resolve(%q) prints %q, and the parsed node %q", tt.text, resolved, node)
		}
		if _, ok := node.(*AttributeExpression).Value.Data.(Lookup); !ok {
			t.Errorf("Resolve(%q) changed the parsed node", tt.text)
		}
	}
}

func TestResolveTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		text, message string
	}{
		{"{a = .table}", ".table is a table, which can't be compared"},
		{"{a = .nested}", ".nested is a table, which can't be compared"},
		{"{a ~ .count}", `.count is "5": ~ needs a regular expression`},
		{"{a ~ .numbers}", `.numbers is "1": ~ needs a regular expression`},
		{"{a < .flag}", `.flag is "true": true can only be compared with = or !=`},
		{"{a ~ .bad}", `.bad is "(": invalid regular expression`},
	} {
		_, err := Resolve(mustParse(t, tt.text), lookupState)

		if _, ok := err.(*TypeError); !ok || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("Resolve(%q) error = %v, want a type error containing %q", tt.text, err, tt.message)
		}
	}
}

func TestResolveLooksUpEachPath(t *testing.T) {
	var paths []string

	_, err := Resolve(mustParse(t, "{a = .x} and ({b = .y} or not {c = .x})"), func(path string) (any, bool) {
		paths = append(paths, path)
		return "v", true
	})

	if err != nil || strings.Join(paths, " ") != ".x .y .x" {
		t.Errorf("looked up %v, %v, want each lookup from the snapshot", paths, err)
	}
}

func TestUnresolvedLookupIsUnknown(t *testing.T) {
	clause, args := mustParse(t, "{a = .x}").Where()

	if clause != "NULL" || len(args) != 0 {
		t.Errorf("Where() = %q, %v, want NULL", clause, args)
	}
}
