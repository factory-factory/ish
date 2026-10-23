package filter

import (
	"reflect"
	"strings"
	"testing"
)

var directory = []User{
	{"Alice Liddell", "alice@example.com"},
	{"Bob", "bob@example.com"},
	{"Sam", "sam@one.example"},
	{"sam", "sam@two.example"},
}

func TestLookUpUser(t *testing.T) {
	for name, want := range map[string]any{
		"Alice Liddell":     "alice@example.com",
		"alice liddell":     "alice@example.com", // Regardless of case.
		"@Alice Liddell":    "alice@example.com", // Ignoring a leading @.
		"@bob":              "bob@example.com",
		"Sam":               "sam@one.example", // An exact match first.
		"sam":               "sam@two.example",
		"SAM":               nil, // Two users, regardless of case.
		"Carol":             nil,
		"alice@example.com": nil, // Only names are looked up.
	} {
		if got := lookUpUser(name, directory); got != want {
			t.Errorf("lookUpUser(%q) = %v, want %v", name, got, want)
		}
	}
}

// userValue resolves {a = text} with the directory, and returns the value
// of its expression, as JSON.
func userValue(t *testing.T, text string) (string, error) {
	t.Helper()

	resolved, err := ResolveEnv(mustParse(t, "{a = "+text+"}"), Env{Lookup: lookupState, Users: directory})
	if err != nil {
		return "", err
	}

	data := resolved.(*AttributeExpression).Value.Data
	if _, ok := data.(unknown); ok {
		return "unknown", nil
	}

	v, err := data.(*Expression).Root.eval(Env{})

	return encodeValue(v), err
}

func TestResolveUser(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		{"('Bob' :user)", `"bob@example.com"`},
		{"('@alice liddell' :user)", `"alice@example.com"`},
		{"(['Bob', 'Carol', '@Sam'] :user)", `["bob@example.com",null,"sam@one.example"]`},
		{"('Bob' :user :upper)", `"BOB@EXAMPLE.COM"`},
		{`('\('Bob' :user)!')`, `"bob@example.com!"`},
		// No one is null, which is UNKNOWN.
		{"('Carol' :user)", "unknown"},
		{"(.missing :user)", "unknown"},
	} {
		if got, err := userValue(t, tt.text); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v, want %s", tt.text, got, err, tt.want)
		}
	}

	for _, tt := range []struct {
		text, message string
	}{
		{"(5 :user)", ":user needs a name or a list of names, not a number"},
		{"([1] :user)", ":user needs names, not a list with a number"},
	} {
		if _, err := userValue(t, tt.text); err == nil || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("%s error = %v, want one containing %q", tt.text, err, tt.message)
		}
	}

	if _, err := Parse("{a = ('Bob' :user('x'))}"); err == nil || !strings.Contains(err.Error(), ":user takes no arguments") {
		t.Errorf("Parse() with an argument to :user error = %v, want a syntax error", err)
	}
}

func TestUserPrintsAsWritten(t *testing.T) {
	node := mustParse(t, "{assignees = ('@Brendan Berg' :user)}")

	if got := node.String(); got != "{assignees = ('@Brendan Berg' :user)}" {
		t.Errorf("String() = %q", got)
	}

	// Resolving doesn't change the parsed filter.
	if _, err := ResolveEnv(node, Env{Users: directory}); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(node, mustParse(t, "{assignees = ('@Brendan Berg' :user)}")) {
		t.Error("resolving changed the parsed filter")
	}
}

func TestUnresolvedUserFindsNoOne(t *testing.T) {
	expr := mustParse(t, "{a = ('Bob' :user)}").(*AttributeExpression).Value.Data.(*Expression)

	if v, err := expr.Eval(nil); err != nil || v != nil {
		t.Errorf("Eval() without a directory = %v, %v, want null", v, err)
	}
}
