package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeListIssues writes sample issues to a new issue directory, and returns
// it and a home directory for the configuration.
func writeListIssues(t *testing.T) (dir, home string) {
	t.Helper()

	root := t.TempDir()
	dir = filepath.Join(root, ".ish")
	home = filepath.Join(root, "home")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	for name, source := range map[string]string{
		"BBBBB-22222.md": "# Second issue\n\n[\"Bob\" <bob@example.com> 2026-10-12T10:00:00+02:00]\n\n- state: closed\n",
		"AAAAA-11111.md": "# First issue\n\n[alice@example.com 2026-10-11T08:59:23-07:00]\n\n- state: open\n- assignees: <alice@example.com>, <bob@example.com>\n- label: tab\tseparated\n",
		"CCCCC-33333.md": "# Third issue\n\n[alice@example.com 2026-10-13T09:00:00Z]\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir, home
}

func TestRunList(t *testing.T) {
	dir, home := writeListIssues(t)

	tests := []struct {
		name             string
		queries, columns []string
		want             string
	}{
		{
			name: "every issue",
			want: "AAAAA-11111\t2026-10-11T08:59:23-07:00\talice@example.com\tFirst issue\n" +
				"BBBBB-22222\t2026-10-12T10:00:00+02:00\tbob@example.com\tSecond issue\n" +
				"CCCCC-33333\t2026-10-13T09:00:00Z\talice@example.com\tThird issue\n",
		},
		{
			name:    "a query",
			queries: []string{`{state != "closed"}`},
			want: "AAAAA-11111\t2026-10-11T08:59:23-07:00\talice@example.com\tFirst issue\n" +
				"CCCCC-33333\t2026-10-13T09:00:00Z\talice@example.com\tThird issue\n",
		},
		{
			name:    "queries joined with and",
			queries: []string{`{state != "closed"}`, `{@author = "alice@example.com"}`, `{state = "open"}`},
			want:    "AAAAA-11111\t2026-10-11T08:59:23-07:00\talice@example.com\tFirst issue\n",
		},
		{
			name:    "columns",
			columns: []string{"assignees", "state"},
			want: "AAAAA-11111\t2026-10-11T08:59:23-07:00\talice@example.com\talice@example.com,bob@example.com\topen\tFirst issue\n" +
				"BBBBB-22222\t2026-10-12T10:00:00+02:00\tbob@example.com\t\tclosed\tSecond issue\n" +
				"CCCCC-33333\t2026-10-13T09:00:00Z\talice@example.com\t\t\tThird issue\n",
		},
		{
			name:    "a column only some issues have",
			queries: []string{`{state = "closed"}`},
			columns: []string{"assignees"},
			want:    "BBBBB-22222\t2026-10-12T10:00:00+02:00\tbob@example.com\t\tSecond issue\n",
		},
		{
			name:    "tabs in values",
			queries: []string{`{state = "open"}`},
			columns: []string{"label"},
			want:    "AAAAA-11111\t2026-10-11T08:59:23-07:00\talice@example.com\ttab separated\tFirst issue\n",
		},
	}

	for _, tt := range tests {
		var stdout, stderr bytes.Buffer

		if code := runList(dir, home, tt.queries, tt.columns, &stdout, &stderr); code != 0 {
			t.Errorf("%s: runList() = %d, want 0 (stderr: %s)", tt.name, code, stderr.String())
		}
		if got := stdout.String(); got != tt.want {
			t.Errorf("%s: stdout = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRunListLookup(t *testing.T) {
	dir, home := writeListIssues(t)

	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[user]\nemail = \"bob@example.com\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer

	if code := runList(dir, home, []string{"{@author = .user.email}"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("runList() = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if got, want := stdout.String(), "BBBBB-22222\t2026-10-12T10:00:00+02:00\tbob@example.com\tSecond issue\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunListReportsToStderr(t *testing.T) {
	dir, home := writeListIssues(t)

	// Building the database is reported, but not to stdout.
	var stdout, stderr bytes.Buffer

	if code := runList(dir, home, nil, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("runList() = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "Indexed") || !strings.Contains(stderr.String(), "Indexed 3 issues") {
		t.Errorf("stdout = %q, stderr = %q, want the database report only on stderr", stdout.String(), stderr.String())
	}
}

func TestRunListErrors(t *testing.T) {
	dir, home := writeListIssues(t)
	missing := filepath.Join(t.TempDir(), ".ish")

	tests := []struct {
		name    string
		dir     string
		queries []string
		columns []string
		want    string
	}{
		{"bad query", dir, []string{`{state != "closed"}`, "{state"}, nil, `Couldn't parse the query "{state"`},
		{"no issue directory", missing, nil, nil, missing + " doesn't exist"},
		{"a column no issue has", dir, nil, []string{"state", "asignees"}, "No issue has the attribute asignees\n"},
		{"columns no issue has", dir, nil, []string{"x", "state", "@title", "x"}, "No issue has the attributes x, @title\n"},
	}

	for _, tt := range tests {
		var stdout, stderr bytes.Buffer

		if code := runList(tt.dir, home, tt.queries, tt.columns, &stdout, &stderr); code != 1 {
			t.Errorf("%s: runList() = %d, want 1", tt.name, code)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), tt.want) {
			t.Errorf("%s: stdout = %q, stderr = %q, want only %q on stderr", tt.name, stdout.String(), stderr.String(), tt.want)
		}
	}

	// The missing directory isn't created.
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Errorf("runList() created %s", missing)
	}
}

func TestParseQueries(t *testing.T) {
	// No queries match every issue.
	if node, err := parseQueries(nil); node != nil || err != nil {
		t.Errorf("parseQueries(nil) = %v, %v, want nil, nil", node, err)
	}

	node, err := parseQueries([]string{"{a = 'x'} or {b = 'y'}", "{c = 'z'}"})
	if err != nil {
		t.Fatal(err)
	}

	// The queries are grouped, so an or in one doesn't change the other.
	clause, _ := node.Where()
	if !strings.HasPrefix(clause, "((") || !strings.Contains(clause, ") AND (") {
		t.Errorf("parseQueries() clause = %s, want the queries grouped and joined with AND", clause)
	}
}

func TestBlankControls(t *testing.T) {
	if got, want := blankControls.Replace("a\tb\nc\r\nd\re"), "a b c d e"; got != want {
		t.Errorf("blankControls.Replace() = %q, want %q", got, want)
	}
}
