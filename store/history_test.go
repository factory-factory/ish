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
// the revisions and files it's asked for history of, as "revs" or
// "revs -- paths".
func stubHistory(t *testing.T, out string, err error) *[]string {
	t.Helper()

	var asked []string
	original := gitHistory
	gitHistory = func(dir, revs string, paths []string) ([]byte, error) {
		if len(paths) > 0 {
			revs += " -- " + strings.Join(paths, " ")
		}

		asked = append(asked, revs)
		return []byte(out), err
	}
	t.Cleanup(func() { gitHistory = original })

	return &asked
}

// stubHead replaces gitHead with one that returns head, or an error if it's
// empty, and gitIsAncestor with one that returns ancestor.
func stubHead(t *testing.T, head string, ancestor bool) {
	t.Helper()

	originalHead, originalAncestor := gitHead, gitIsAncestor
	gitHead = func(string) (string, error) {
		if head == "" {
			return "", errors.New("not a git repository")
		}

		return head, nil
	}
	gitIsAncestor = func(string, string, string) bool { return ancestor }
	t.Cleanup(func() { gitHead, gitIsAncestor = originalHead, originalAncestor })
}

// headRow returns the commit the database in dir's history was indexed at.
func headRow(t *testing.T, dir string) string {
	t.Helper()

	head, err := indexedHead(openDB(t, dir))
	if err != nil {
		t.Fatal(err)
	}

	return head
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
	stubHead(t, "c1", true)
	asked := stubHistory(t, commits(
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

	// The whole history of HEAD, without issues that aren't in the
	// database.
	if !slices.Equal(*asked, []string{"c1"}) {
		t.Errorf("asked for history of %q, want all of HEAD's", *asked)
	}
	if got := headRow(t, dir); got != "c1" {
		t.Errorf("indexed at %q, want c1", got)
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
	stubHead(t, "c1", true)
	stubHistory(t, commits([]string{"2026-10-02T09:00:00Z", ".ish/AAAAA-11111.md"}), nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)

	// After a merge, the commits since the indexed one are asked for,
	// including ones older than those already indexed.
	stubHead(t, "c2", true)
	asked := stubHistory(t, commits([]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md"}), nil)

	added, err := IndexHistory(dir)

	if err != nil || added != 1 {
		t.Fatalf("IndexHistory() = %d, %v, want 1", added, err)
	}
	if !slices.Equal(*asked, []string{"c1..c2"}) {
		t.Errorf("asked for history of %q, want c1..c2", *asked)
	}
	if got := headRow(t, dir); got != "c2" {
		t.Errorf("indexed at %q, want c2", got)
	}

	want := []string{"AAAAA-11111 2026-10-01T09:00:00Z", "AAAAA-11111 2026-10-02T09:00:00Z"}

	if got := historyRows(t, dir); !slices.Equal(got, want) {
		t.Errorf("history = %q, want %q", got, want)
	}

	// With HEAD where it was, git isn't asked again.
	*asked = nil

	if added, err := IndexHistory(dir); err != nil || added != 0 || len(*asked) > 0 {
		t.Errorf("IndexHistory() again = %d, %v, asking for %q; want nothing", added, err, *asked)
	}
}

func TestIndexHistoryRebuildsRewrittenHistory(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHead(t, "c1", true)
	stubHistory(t, commits(
		[]string{"2026-10-02T09:00:00Z", ".ish/AAAAA-11111.md"},
		[]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md"},
	), nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)

	// The indexed commit isn't an ancestor of HEAD, as after a reset, so
	// the history is HEAD's alone.
	stubHead(t, "c0", false)
	asked := stubHistory(t, commits([]string{"2026-10-01T09:00:00Z", ".ish/AAAAA-11111.md"}), nil)

	if _, err := IndexHistory(dir); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(*asked, []string{"c0"}) {
		t.Errorf("asked for history of %q, want all of c0's", *asked)
	}

	if got := historyRows(t, dir); !slices.Equal(got, []string{"AAAAA-11111 2026-10-01T09:00:00Z"}) {
		t.Errorf("history = %q, want only c0's", got)
	}
}

func TestIndexHistoryWithoutGit(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHead(t, "", false)
	asked := stubHistory(t, "", errors.New("not a git repository"))

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)

	if added, err := IndexHistory(dir); err != nil || added != 0 || len(*asked) > 0 {
		t.Errorf("IndexHistory() without git = %d, %v, asking for %q; want 0, nil", added, err, *asked)
	}
	if got := headRow(t, dir); got != "" {
		t.Errorf("indexed at %q, want nothing", got)
	}
	if _, err := IndexHistory(t.TempDir()); err == nil {
		t.Error("IndexHistory() without a database error = nil, want an error")
	}
}

func TestSyncAddsHistoryOfNewFiles(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHead(t, "c1", true)
	stubHistory(t, "", nil)

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)

	// A file that was committed before it was added to the database.
	asked := stubHistory(t, commits([]string{"2026-10-01T09:00:00Z", ".ish/BBBBB-22222.md"}), nil)
	writeIssues(t, dir, map[string]string{"BBBBB-22222": issueSource("Second", nil)})

	if result, err := Sync(dir); err != nil || len(result.Added) != 1 {
		t.Fatalf("Sync() = %+v, %v, want BBBBB-22222 added", result, err)
	}
	if !slices.Equal(*asked, []string{"c1 -- BBBBB-22222.md"}) {
		t.Errorf("asked for history of %q, want the new file's up to c1", *asked)
	}
	if got := historyRows(t, dir); !slices.Equal(got, []string{"BBBBB-22222 2026-10-01T09:00:00Z"}) {
		t.Errorf("history = %q, want the new file's", got)
	}

	// Without new files, git isn't asked.
	*asked = nil
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First, changed", nil)})

	if _, err := Sync(dir); err != nil || len(*asked) > 0 {
		t.Errorf("Sync() = %v, asking for %q; want git not asked", err, *asked)
	}
}

func TestReindexingKeepsHistory(t *testing.T) {
	stubAuthors(t, "", nil)
	stubHead(t, "c1", true)
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

	out, err := gitHistory(dir, "HEAD", nil)
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

	if _, err := gitHistory(t.TempDir(), "HEAD", nil); err == nil {
		t.Error("gitHistory() outside a repository error = nil, want an error")
	}
}

func TestIndexHistoryFollowsGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git isn't installed")
	}

	root := t.TempDir()
	dir := filepath.Join(root, ".ish")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	// git runs a git command in the repository, with its commits authored
	// at date.
	git := func(date string, args ...string) {
		t.Helper()

		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=a", "GIT_AUTHOR_EMAIL=a@example.com", "GIT_AUTHOR_DATE="+date,
			"GIT_COMMITTER_NAME=ish", "GIT_COMMITTER_EMAIL=ish@example.com",
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// commit commits a new version of the issue, authored at date.
	commit := func(date string) {
		t.Helper()

		writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("Version "+date, nil)})
		git(date, "add", ".")
		git(date, "commit", "-q", "-m", date)
	}

	git("", "init", "-q", "-b", "main")
	commit("2026-10-01T09:00:00Z")
	initDir(t, dir)

	// A side branch with an older commit.
	git("", "checkout", "-q", "-b", "side")
	commit("2026-09-15T09:00:00Z")
	git("", "checkout", "-q", "main")
	commit("2026-10-05T09:00:00Z")

	if _, err := IndexHistory(dir); err != nil {
		t.Fatal(err)
	}

	want := []string{"AAAAA-11111 2026-10-01T09:00:00Z", "AAAAA-11111 2026-10-05T09:00:00Z"}

	if got := historyRows(t, dir); !slices.Equal(got, want) {
		t.Errorf("history after a commit = %q, want %q", got, want)
	}

	// Merging adds the side branch's commit, though it's older.
	git("2026-10-06T09:00:00Z", "merge", "-q", "--no-edit", "-X", "ours", "side")

	if _, err := IndexHistory(dir); err != nil {
		t.Fatal(err)
	}

	merged := []string{"AAAAA-11111 2026-09-15T09:00:00Z", want[0], want[1]}

	if got := historyRows(t, dir); !slices.Equal(got, merged) {
		t.Errorf("history after a merge = %q, want %q", got, merged)
	}

	// Undoing the merge takes its commit out of the history again.
	git("", "reset", "-q", "--hard", "HEAD~1")

	if _, err := IndexHistory(dir); err != nil {
		t.Fatal(err)
	}

	if got := historyRows(t, dir); !slices.Equal(got, want) {
		t.Errorf("history after a reset = %q, want %q", got, want)
	}
}
