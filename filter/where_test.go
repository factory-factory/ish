package filter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/store"
)

// issues are written to a temporary directory and indexed, to run filters
// against.
var issues = map[string][]string{
	"A": {"- state: open", "- assignees: alice, bob", "- estimate: 5", "- done: true"},
	"B": {"- state: closed", "- assignees: carol", "- estimate: 8h"},
	"C": {"- state: in progress", "- estimate: 2.5", "- priority: high"},
	"D": nil,
	"E": {"- state: <script>", "- assignees: it's"},
}

// headers are the titles and bylines of the issues that have their own,
// for filters on fields. dateCreated is stored in UTC, which for B and D is
// a different day than in their own time zones.
var headers = map[string]string{
	"A": "# Fix the parser\n\n[alice 2026-10-12T10:00:00-07:00]\n", // 2026-10-12T17:00:00Z
	"B": "# Filter by field\n\n[bob 2026-10-13T20:30:00-07:00]\n",  // 2026-10-14T03:30:00Z
	"C": "# 42\n\n[alice 2026-09-30T23:00:00Z]\n",
	"D": "# D\n\n[carol 2026-10-14T01:00:00+02:00]\n", // 2026-10-13T23:00:00Z
}

// match returns the IDs of the issues that text matches, in order.
func match(t *testing.T, dir, text string) string {
	t.Helper()

	node, err := filter.Parse(text)

	if err != nil {
		t.Fatalf("Parse(%q) error = %v", text, err)
	}

	clause, args := node.Where()

	db, err := store.Open(filepath.Join(dir, store.DatabaseName))

	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	rows, err := db.Query("SELECT issue.id FROM issue WHERE "+clause+" ORDER BY issue.id", args...)

	if err != nil {
		t.Fatalf("%q: query error = %v\n%s", text, err, clause)
	}

	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return strings.Join(ids, " ")
}

func indexedDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	for id, metadata := range issues {
		source, ok := headers[id]
		if !ok {
			source = "# " + id + "\n\n[alice 2026-10-12T10:00:00-07:00]\n"
		}

		if len(metadata) > 0 {
			source += "\n" + strings.Join(metadata, "\n") + "\n"
		}

		if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := store.Init(dir)

	if err != nil || len(result.Failed) > 0 {
		t.Fatalf("store.Init() = %+v, %v", result, err)
	}

	return dir
}

func TestWhere(t *testing.T) {
	dir := indexedDir(t)

	tests := []struct {
		text string
		want string
	}{
		{"{state = 'open'}", "A"},
		{`{state = "open"}`, "A"},

		// An issue without the attribute matches only !=.
		{"{state != 'closed'}", "A C D E"},
		{"{priority = 'high'}", "C"},
		{"{priority != 'high'}", "A B D E"},

		// Any of several values can match =, and != is its opposite.
		{"{assignees = 'bob'}", "A"},
		{"{assignees = 'alice'}", "A"},
		{"{assignees != 'bob'}", "B C D E"},

		// null tests whether the attribute is present.
		{"{assignees = null}", "C D"},
		{"{assignees != null}", "A B E"},

		// Numbers compare numerically, and only with values that are
		// numbers.
		{"{estimate > 3}", "A"},
		{"{estimate >= 2.5}", "A C"},
		{"{estimate < 5}", "C"},
		{"{estimate <= 5}", "A C"},
		{"{estimate = 5.0}", "A"},
		{"{estimate = 5e0}", "A"},
		{"{estimate != 5}", "B C D E"},

		// Strings compare as text.
		{"{estimate = '5'}", "A"},
		{"{estimate = '5.0'}", ""},
		{"{estimate = '8h'}", "B"},
		{"{state < 'd'}", "B E"},
		{"{state >= 'open'}", "A"},
		{"{state = 'OPEN'}", ""},

		// true and false match the text.
		{"{done = true}", "A"},
		{"{done != true}", "B C D E"},
		{"{done = false}", ""},

		// Regular expressions match any value.
		{"{state ~ '^in '}", "C"},
		{"{state ~ '(?i)OPEN'}", "A"},
		{"{assignees ~ 'o'}", "A B"},
		{`{estimate ~ '^\\d+$'}`, "A"},
		{"{state ~ 'x'}", ""},

		// Regular expression literals pass backslashes through unchanged.
		{`{state ~ /^in\s/}`, "C"},
		{`{estimate ~ /^\d+$/}`, "A"},
		{`{assignees ~ /\bbob\b/}`, "A"},
		{`{assignees ~ /\bbo\b/}`, ""},
		{`{state ~ /(?i)^OPEN$/}`, "A"},
		{`not {state ~ /^<.*>$/}`, "A B C D"},

		// Logical operators and precedence.
		{"not {state = 'open'}", "B C D E"},
		{"{state = 'open'} or {state = 'closed'} and {estimate = '8h'}", "A B"},
		{"({state = 'open'} or {state = 'closed'}) and {assignees = 'carol'}", "B"},
		{"{state = 'open'} or {state = 'closed'} and {assignees = 'nobody'}", "A"},
		{"not ({state = 'open'} or {state = null})", "B C E"},
		{"not not {state = 'open'}", "A"},

		// Fields compare the issue's title, author, and creation time.
		{"{@title = 'Fix the parser'}", "A"},
		{"{@title != 'Fix the parser'}", "B C D E"},
		{"{@title ~ /^Fi/}", "A B"},
		{"{@title ~ '(?i)PARSER'}", "A"},
		{"{@title < 'E'}", "C D"},
		{"{@author = 'alice'}", "A C E"},
		{"{@author != 'alice'}", "B D"},
		{"{@author = 'alice'} and {state != 'closed'}", "A C E"},
		{"not {@author = 'alice'} or {priority = 'high'}", "B C D"},

		// A field compares numerically with a number, and != is the
		// opposite of = even where the field isn't a number.
		{"{@title = 42}", "C"},
		{"{@title = 42.0}", "C"},
		{"{@title != 42}", "A B D E"},
		{"{@title > 40}", "C"},
		{"{@title = true}", ""},
		{"{@title != true}", "A B C D E"},

		// Every issue has every field.
		{"{@author = null}", ""},
		{"{@author != null}", "A B C D E"},

		// dateCreated is RFC 3339 text in UTC, so dates compare as text.
		{"{@dateCreated = '2026-10-12T17:00:00Z'}", "A E"},
		{"{@dateCreated >= '2026-10-13'}", "B D"},
		{"{@dateCreated < '2026-10-01'}", "C"},
		{"{@dateCreated ~ /^2026-10-13/}", "D"},

		// Values are passed as parameters, so quotes and SQL in them are
		// just text.
		{`{state = "x' OR 1=1 --"}`, ""},
		{`{assignees = 'it\'s'}`, "E"},
		{"{state = '<script>'}", "E"},
	}

	for _, tt := range tests {
		if got := match(t, dir, tt.text); got != tt.want {
			t.Errorf("%s matches %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestWhereUsesParameters(t *testing.T) {
	node, err := filter.Parse(`{state = "x' OR 1=1 --"} and {a.b = 1}`)

	if err != nil {
		t.Fatal(err)
	}

	clause, args := node.Where()

	if strings.Contains(clause, "OR 1=1") || strings.Contains(clause, "a.b") {
		t.Errorf("clause %q contains a value or attribute name, want only placeholders", clause)
	}
	if len(args) != 4 {
		t.Errorf("args = %v, want the attribute and value of each expression", args)
	}
}

func TestWhereFieldColumns(t *testing.T) {
	for _, tt := range []struct {
		text   string
		column string
	}{
		{"{@title = 'x'}", "issue.title"},
		{"{@author != 'x'}", "issue.author"},
		{"{@dateCreated ~ /x/}", "issue.dateCreated"},
	} {
		clause, args := mustWhere(t, tt.text)

		if !strings.Contains(clause, tt.column) || strings.Contains(clause, "attribute") {
			t.Errorf("%s: clause %q, want a condition on %s and not the attribute table", tt.text, clause, tt.column)
		}
		if len(args) != 1 || args[0] != "x" {
			t.Errorf("%s: args = %v, want [x]", tt.text, args)
		}
	}
}

func mustWhere(t *testing.T, text string) (string, []any) {
	t.Helper()

	node, err := filter.Parse(text)

	if err != nil {
		t.Fatalf("Parse(%q) error = %v", text, err)
	}

	return node.Where()
}
