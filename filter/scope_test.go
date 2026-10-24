package filter_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/store"
)

var escapes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestScopesPrintAsWritten(t *testing.T) {
	for _, text := range []string{
		"{estimate >= 3 and <= 7}",
		"{assignees ~ /^a/ and not = 'alice'}",
		"{@comments author = 'a@example.com' and text ~ /LGTM/}",
		"{@comments not (author = 'a' and text ~ 'x') and dateCreated > ('-1w' :fromnow)}",
		"{@comments author = .user.email}",
		"{@comments text = (['a', 'b'])}",
		"not ({@title ~ 'a' and ~ 'b'}) or {state = 'open'}",
	} {
		node, err := filter.Parse(text)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", text, err)
		}

		if got := node.String(); got != text {
			t.Errorf("String() = %q, want %q", got, text)
		}

		if got := escapes.ReplaceAllString(node.Render(), ""); got != text {
			t.Errorf("Render() without colors = %q, want %q", got, text)
		}
	}
}

func TestParseScopes(t *testing.T) {
	// A single comparison is still an attribute expression.
	for _, text := range []string{"{state != 'closed'}", "{@comments.author = 'a'}", "{a = null}"} {
		if node, err := filter.Parse(text); err != nil {
			t.Errorf("Parse(%q) error = %v", text, err)
		} else if _, ok := node.(*filter.AttributeExpression); !ok {
			t.Errorf("Parse(%q) = %T, want an attribute expression", text, node)
		}
	}

	node, err := filter.Parse("{@comments author = 'a' and not text ~ 'b'}")
	if err != nil {
		t.Fatal(err)
	}

	scope, ok := node.(*filter.ScopeExpression)
	if !ok || scope.Subject != "@comments" {
		t.Fatalf("Parse() = %#v, want a scope of @comments", node)
	}

	and, ok := scope.Condition.(*filter.ConditionAnd)
	if !ok {
		t.Fatalf("Condition = %#v, want an and", scope.Condition)
	}

	if left, ok := and.Left.(*filter.Comparison); !ok || left.Field != "author" || left.Operator != filter.Equal || left.Value.Data != "a" {
		t.Errorf("Left = %#v, want author = 'a'", and.Left)
	}

	if not, ok := and.Right.(*filter.ConditionNot); !ok {
		t.Errorf("Right = %#v, want a not", and.Right)
	} else if right, ok := not.X.(*filter.Comparison); !ok || right.Field != "text" || right.Operator != filter.Match {
		t.Errorf("Right.X = %#v, want text ~ 'b'", not.X)
	}

	// A collection with one comparison is a scope, too.
	if node, err := filter.Parse("{@comments author = 'a'}"); err != nil {
		t.Error(err)
	} else if _, ok := node.(*filter.ScopeExpression); !ok {
		t.Errorf("Parse() = %T, want a scope", node)
	}
}

func TestScopeErrors(t *testing.T) {
	for _, tt := range []struct {
		text       string
		start, end int
		message    string
	}{
		{"{estimate > 3 or < 1}", 14, 16, "or can't join conditions in braces"},
		{"{a = 1 and != 2}", 11, 15, "!= can only be used on its own in braces"},
		{"{a != 1 and = 2}", 3, 7, "!= can only be used on its own in braces"},
		{"{a = 1 and = null}", 11, 17, "null can only be compared on its own in braces"},
		{"{@comments}", 10, 11, "expected a field of @comments: author, text, and dateCreated"},
		{"{@comments = 'a'}", 11, 12, "expected a field of @comments"},
		{"{@comments state = 'a'}", 11, 16, "unknown field state of @comments; its fields are author, text, and dateCreated"},
		{"{@comments author = 'a' and @title = 'b'}", 28, 34, "expected a field of @comments"},
		{"{a (= 1 and = 2}", 15, 16, "expected and or a closing parenthesis"},
		{"{a (= 1 and = 2", 3, 4, "this parenthesis is never closed"},
		{"{a (= 1 = 2)}", 8, 9, "expected and or a closing parenthesis"},
	} {
		_, err := filter.Parse(tt.text)
		syntaxErr, ok := err.(*filter.SyntaxError)

		if !ok || syntaxErr.Start != tt.start || syntaxErr.End != tt.end || !strings.Contains(syntaxErr.Message, tt.message) {
			t.Errorf("Parse(%q) error = %#v, want [%d, %d) containing %q", tt.text, err, tt.start, tt.end, tt.message)
		}
	}
}

func TestWhereAttributeScopes(t *testing.T) {
	dir := indexedDir(t)

	for _, tt := range []struct {
		text string
		want string
	}{
		{"{estimate >= 3 and <= 7}", "A"},
		{"{estimate >= 1 and <= 10}", "A C"},
		{"{estimate >= 3 and (<= 7)}", "A"},
		// One value must satisfy every condition.
		{"{assignees ~ /^a/ and ~ /e$/}", "A"},
		{"{assignees ~ /^b/ and ~ /e$/}", ""},
		{"{assignees ~ /^b/} and {assignees ~ /e$/}", "A"},
		// not applies to the value, so this is "an assignee other than
		// alice", not "not assigned to alice".
		{"{assignees ~ /./ and not = 'alice'}", "A B E"},
		{"not {assignees ~ /./ and not = 'alice'}", "C D"},
		{"{assignees ~ /./ and = (['carol', 'bob'])}", "A B"},
		{"{assignees ~ /./ and = ('CAROL' :lower)}", "B"},
		// Fields have one value per issue.
		{"{@title ~ 'Fix' and ~ 'parser'}", "A"},
		{"{@title ~ 'Fix' and not ~ 'parser'}", ""},
	} {
		node, err := filter.Parse(tt.text)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", tt.text, err)
		}

		resolved, err := filter.ResolveEnv(node, filter.Env{})
		if err != nil {
			t.Fatalf("ResolveEnv(%q) error = %v", tt.text, err)
		}

		if got := matchNode(t, dir, tt.text, resolved); got != tt.want {
			t.Errorf("%s matches %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestWhereCommentScopes(t *testing.T) {
	dir := commentsDir(t)
	users := []filter.User{{Name: "Bob", Email: "bob@example.com"}}
	env := filter.Env{
		Users:  users,
		Now:    time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		Lookup: func(path string) (any, bool) { return map[string]any{".me": "bob@example.com"}[path], path == ".me" },
	}

	for _, tt := range []struct {
		text string
		want string
	}{
		// The same comment must match both conditions.
		{"{@comments author = 'bob@example.com' and text ~ 'parser'}", ""},
		{"{@comments author = 'bob@example.com'} and {@comments text ~ 'parser'}", "X"},
		{"{@comments author = 'bob@example.com' and text ~ 'LGTM'}", "X"},
		{"{@comments author = ('Bob' :user) and text ~ 'feature'}", "Y"},
		{"{@comments author = .me and text ~ 'feature'}", "Y"},
		{"{@comments text ~ /\\./ and not author = 'alice@example.com'}", "X Y"},
		{"{@comments author = (['carol@example.com', 'alice@example.com']) and text ~ 'parser'}", "X"},
		{"{@comments author = 'bob@example.com' and dateCreated > ('-7d' :fromnow)}", "X"},
		{"not {@comments author = 'bob@example.com' and text ~ 'LGTM'}", "Y Z"},
		// A single comparison on a collection matches as the field does.
		{"{@comments author = 'carol@example.com'}", "X"},
		// A missing lookup makes the scope unknown, so neither it nor its
		// opposite matches.
		{"{@comments author = .missing and text ~ 'a'}", ""},
		{"not {@comments author = .missing and text ~ 'a'}", ""},
	} {
		node, err := filter.Parse(tt.text)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", tt.text, err)
		}

		resolved, err := filter.ResolveEnv(node, env)
		if err != nil {
			t.Fatalf("ResolveEnv(%q) error = %v", tt.text, err)
		}

		if got := matchNode(t, dir, tt.text, resolved); got != tt.want {
			t.Errorf("%s matches %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestWhereDateModifiedScope(t *testing.T) {
	dir := indexedDir(t)

	db, err := store.Open(filepath.Join(dir, store.DatabaseName))
	if err != nil {
		t.Fatal(err)
	}

	for i, row := range [][2]string{
		{"A", "2026-10-01T09:00:00Z"},
		{"A", "2026-10-10T09:00:00Z"},
		{"B", "2026-10-05T09:00:00Z"},
	} {
		if _, err := db.Exec("INSERT INTO history VALUES (?, ?, ?)", string(rune('a'+i)), row[0], row[1]); err != nil {
			t.Fatal(err)
		}
	}

	db.Close()

	// A was changed before and after the range, but not during it.
	for _, tt := range []struct {
		text string
		want string
	}{
		{"{@dateModified > '2026-10-03' and < '2026-10-07'}", "B"},
		{"{@dateModified > '2026-10-03'} and {@dateModified < '2026-10-07'}", "A B"},
	} {
		if got := match(t, dir, tt.text); got != tt.want {
			t.Errorf("%s matches %q, want %q", tt.text, got, tt.want)
		}
	}
}
