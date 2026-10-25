package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/store"
)

// historyRepo returns a model listing the issues in a new git repository,
// and a function that commits the issue files, authored by author at date.
func historyRepo(t *testing.T, sources map[string]string) (Model, func(author, date string)) {
	t.Helper()

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

	commit := func(author, date string) {
		t.Helper()

		name, email, _ := strings.Cut(author, " <")
		git(nil, "add", ".")
		git([]string{
			"GIT_AUTHOR_NAME=" + name,
			"GIT_AUTHOR_EMAIL=" + strings.TrimSuffix(email, ">"),
			"GIT_AUTHOR_DATE=" + date,
		}, "commit", "-q", "-m", date)
	}

	git(nil, "init", "-q")

	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	for id, source := range sources {
		if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	commit("Alice <alice@example.com>", "2026-10-01T09:00:00Z")

	if _, err := store.Init(dir); err != nil {
		t.Fatal(err)
	}

	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	return update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30}), commit
}

// modifiedSince returns the IDs of the issues in the model's database
// modified after date.
func modifiedSince(t *testing.T, m Model, date string) string {
	t.Helper()

	node, err := filter.Parse("{@dateModified > '" + date + "'}")
	if err != nil {
		t.Fatal(err)
	}

	issues, err := store.ListIssues(m.dir, node)
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, issue := range issues {
		ids = append(ids, issue.ID)
	}

	return strings.Join(ids, " ")
}

func TestWaitForHistory(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	if cmd := m.waitForHistory(); cmd != nil {
		t.Error("waitForHistory() without WatchHistory = a command, want nil")
	}

	changes := make(chan struct{}, 1)
	m.WatchHistory(changes)
	changes <- struct{}{}

	if msg := m.waitForHistory()(); msg != (historyMsg{}) {
		t.Errorf("waitForHistory() after a change = %#v, want historyMsg", msg)
	}

	if _, cmd := m.Update(historyMsg{}); cmd == nil {
		t.Error("Update(historyMsg) returned no command, want one waiting for the next change")
	}
}

func TestHistoryChangeIndexesCommits(t *testing.T) {
	m, commit := historyRepo(t, map[string]string{"AAA": issueSource("First", "open")})

	changeFile(t, m, "AAA", issueSource("First, changed", "open"))
	commit("Alice <alice@example.com>", "2026-10-05T09:00:00Z")

	if got := modifiedSince(t, m, "2026-10-02"); got != "" {
		t.Fatalf("modified since 10-02 before the history changed = %q, want none", got)
	}

	m = update(t, m, historyMsg{})

	if got := modifiedSince(t, m, "2026-10-02"); got != "AAA" {
		t.Errorf("modified since 10-02 = %q, want AAA", got)
	}

	// The changed file was synced too.
	if line := row(t, m, "AAA"); !strings.Contains(line, "First, changed") {
		t.Errorf("row %q, want the committed title", line)
	}
}

func TestHistoryChangeIndexesUsers(t *testing.T) {
	m, commit := historyRepo(t, map[string]string{
		"AAA": issueSource("First", "open") + "- reviewer: <dana@example.com>\n",
	})

	changeFile(t, m, "AAA", issueSource("First", "in review")+"- reviewer: <dana@example.com>\n")
	commit("Dana <dana@example.com>", "2026-10-05T09:00:00Z")
	m = update(t, m, historyMsg{})

	m = update(t, m, enter)

	if view := m.View().Content; !strings.Contains(view, "@Dana") {
		t.Errorf("the new committer isn't shown by name:\n%s", view)
	}
}

func TestHistoryChangeAddsPulledIssues(t *testing.T) {
	m, commit := historyRepo(t, map[string]string{"AAA": issueSource("First", "open")})

	// A pull brings in a new issue, with an older commit, and only the
	// change to the history is noticed.
	changeFile(t, m, "BBB", issueSource("Pulled", "open"))
	commit("Bob <bob@example.com>", "2026-09-15T09:00:00Z")
	m = update(t, m, historyMsg{})

	if got := ids(m); got != "AAA BBB" {
		t.Errorf("list = %q, want the pulled issue", got)
	}

	if got := modifiedSince(t, m, "2026-09-01"); got != "AAA BBB" {
		t.Errorf("modified since 09-01 = %q, want both, with the pulled issue's commit", got)
	}
}

func TestHistoryChangeKeepsDeletedMessage(t *testing.T) {
	m, commit := historyRepo(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})
	m = update(t, m, enter)

	// A pull deletes the open issue.
	if err := os.Remove(filepath.Join(m.dir, "AAA.md")); err != nil {
		t.Fatal(err)
	}

	changeFile(t, m, "BBB", issueSource("Second, changed", "open"))
	commit("Bob <bob@example.com>", "2026-10-05T09:00:00Z")
	m = update(t, m, historyMsg{})

	if m.issue != nil {
		t.Fatal("the deleted issue is still open")
	}

	if view := m.View().Content; !strings.Contains(view, "AAA was deleted outside ish") {
		t.Errorf("view doesn't say the issue was deleted:\n%s", view)
	}
}
