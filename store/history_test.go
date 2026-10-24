package store

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// stubHistory replaces gitHistory with one that returns out, and records
// the dates it's asked for history since.
func stubHistory(t *testing.T, out string, err error) *[]string {
	t.Helper()

	var since []string
	original := gitHistory
	gitHistory = func(dir, s string) ([]byte, error) {
		since = append(since, s)
		return []byte(out), err
	}
	t.Cleanup(func() { gitHistory = original })

	return &since
}

// commits returns gitHistory output for commits, each a date followed by
// the files it changed.
func commits(entries ...[]string) string {
	var b strings.Builder

	for _, e := range entries {
		b.WriteString(commitMarker + e[0] + "\n\n")

		for _, f := range e[1:] {
			b.WriteString(f + "\n")
		}
	}

	return b.String()
}

// historyRows returns the history in the database in dir as "id date".
func historyRows(t *testing.T, dir string) []string {
	t.Helper()

	return query(t, openDB(t, dir), "SELECT issueId || ' ' || dateModified FROM history ORDER BY issueId, dateModified")
}

func TestParseHistory(t *testing.T) {
	out := commits(
		[]string{"2026-10-02T09:00:00-07:00", ".ish/AAAAA-11111.md", ".ish/BBBBB-22222.md"},
		[]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md", ".ish/config.toml", ".ish/.gitignore"},
		[]string{"not a date", ".ish/CCCCC-33333.md"},
	)

	got := parseHistory([]byte(out))
	want := []modification{
		{"AAAAA-11111", "2026-10-02T16:00:00Z"},
		{"BBBBB-22222", "2026-10-02T16:00:00Z"},
		{"AAAAA-11111", "2026-10-01T09:00:00Z"},
	}

	if !slices.Equal(got, want) {
		t.Errorf("parseHistory() = %v, want %v", got, want)
	}
}

func TestInitIndexesHistory(t *testing.T) {
	stubAuthors(t, "", nil)
	since := stubHistory(t, commits(
		[]string{"2026-10-02T09:00:00Z", ".ish/AAAAA-11111.md", ".ish/GONE-00000.md"},
		[]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md", ".ish/BBBBB-22222.md"},
		// A commit that's already counted for an issue isn't again.
		[]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md"},
	), nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", nil),
		"BBBBB-22222": issueSource("Second", nil),
		"CCCCC-33333": issueSource("Uncommitted", nil),
	})
	result := initDir(t, dir)

	// The whole history, without issues that aren't in the database.
	if !slices.Equal(*since, []string{""}) {
		t.Errorf("asked for history since %q, want all of it", *since)
	}
	if result.Modifications != 3 {
		t.Errorf("Modifications = %d, want 3", result.Modifications)
	}

	want := []string{
		"AAAAA-11111 2026-10-01T09:00:00Z",
		"AAAAA-11111 2026-10-02T09:00:00Z",
		"BBBBB-22222 2026-10-01T09:00:00Z",
	}

	if got := historyRows(t, dir); !slices.Equal(got, want) {
		t.Errorf("history = %q, want %q", got, want)
	}
}

func TestIndexHistoryAddsNewCommits(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHistory(t, commits([]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md"}), nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)

	// At start-up, only commits since the newest modification are asked
	// for, and those already counted are skipped.
	since := stubHistory(t, commits(
		[]string{"2026-10-03T09:00:00Z", ".ish/AAAAA-11111.md"},
		[]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md"},
	), nil)

	added, err := IndexHistory(dir)

	if err != nil || added != 1 {
		t.Fatalf("IndexHistory() = %d, %v, want 1", added, err)
	}
	if !slices.Equal(*since, []string{"2026-10-01T09:00:00Z"}) {
		t.Errorf("asked for history since %q, want the newest modification", *since)
	}

	// Without git, nothing is added, and nothing fails.
	stubHistory(t, "", errors.New("not a git repository"))

	if added, err := IndexHistory(dir); err != nil || added != 0 {
		t.Errorf("IndexHistory() without git = %d, %v, want 0, nil", added, err)
	}
	if _, err := IndexHistory(t.TempDir()); err == nil {
		t.Error("IndexHistory() without a database error = nil, want an error")
	}
}

func TestReindexingKeepsHistory(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHistory(t, commits([]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md", ".ish/BBBBB-22222.md"}), nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", nil),
		"BBBBB-22222": issueSource("Second", nil),
	})
	initDir(t, dir)

	// A changed file is re-indexed by the start-up sync, and a saved issue
	// by IndexIssue. Both keep the issue's history.
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First, changed", nil)})

	if result, err := Sync(dir); err != nil || len(result.Updated) != 1 {
		t.Fatalf("Sync() = %+v, %v, want AAAAA-11111 updated", result, err)
	}

	if err := IndexIssue(dir, "BBBBB-22222"); err != nil {
		t.Fatal(err)
	}

	want := []string{"AAAAA-11111 2026-10-01T09:00:00Z", "BBBBB-22222 2026-10-01T09:00:00Z"}

	if got := historyRows(t, dir); !slices.Equal(got, want) {
		t.Errorf("history = %q, want %q", got, want)
	}

	// A deleted issue's history goes with it.
	if err := os.Remove(filepath.Join(dir, "BBBBB-22222.md")); err != nil {
		t.Fatal(err)
	}

	if _, err := Sync(dir); err != nil {
		t.Fatal(err)
	}

	if got := historyRows(t, dir); !slices.Equal(got, want[:1]) {
		t.Errorf("history after deleting = %q, want only AAAAA-11111's", got)
	}
}

func TestHistoryIndex(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHistory(t, "", nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)
	db := openDB(t, dir)

	if got := query(t, db, "SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'history' AND sql IS NOT NULL"); !slices.Equal(got, []string{"historyIssueDate"}) {
		t.Errorf("history indexes = %v, want historyIssueDate", got)
	}

	row := "('u', 'AAAAA-11111', '2026-10-01T09:00:00Z')"

	if _, err := db.Exec("INSERT INTO history VALUES " + row); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO history VALUES ('v', 'AAAAA-11111', '2026-10-01T09:00:00Z')"); err == nil {
		t.Error("inserted the same issue and date twice")
	}
	if _, err := db.Exec("INSERT INTO history VALUES ('w', 'NOPE', '2026-10-01T09:00:00Z')"); err == nil {
		t.Error("inserted history for an issue that doesn't exist")
	}
}

func TestGitHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git isn't installed")
	}

	root := t.TempDir()
	dir := filepath.Join(root, ".ish")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	git := func(env []string, args ...string) {
		t.Helper()

		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(cmd.Environ(), append(env, "GIT_COMMITTER_NAME=ish", "GIT_COMMITTER_EMAIL=ish@example.com")...)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	git(nil, "init", "-q")

	for i, date := range []string{"2026-10-01T09:00:00-07:00", "2026-10-02T09:00:00-07:00"} {
		writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("Version "+string(rune('1'+i)), nil)})
		writeIssues(t, root, map[string]string{"README": "not an issue"})

		git(nil, "add", ".")
		git([]string{"GIT_AUTHOR_NAME=a", "GIT_AUTHOR_EMAIL=a@example.com", "GIT_AUTHOR_DATE=" + date}, "commit", "-q", "-m", "change")
	}

	out, err := gitHistory(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// Author dates, in UTC, for the issue files only.
	want := []modification{
		{"AAAAA-11111", "2026-10-02T16:00:00Z"},
		{"AAAAA-11111", "2026-10-01T16:00:00Z"},
	}

	if got := parseHistory(out); !slices.Equal(got, want) {
		t.Errorf("history = %v, want %v", got, want)
	}

	if _, err := gitHistory(t.TempDir(), ""); err == nil {
		t.Error("gitHistory() outside a repository error = nil, want an error")
	}
}
