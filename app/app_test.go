package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
)

func issueSource(title, state string) string {
	source := "# " + title + "\n\n[author 2023-05-26T23:35:36-0400]\n\n"

	if state != "" {
		source += "- state: " + state + "\n"
	}

	return source + "- assignees: alice\n"
}

// newTestModel writes issues to a temporary directory and returns a model
// listing them, sized to a terminal.
func newTestModel(t *testing.T, sources map[string]string) Model {
	t.Helper()

	dir := t.TempDir()

	for id, source := range sources {
		if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	issues, err := parse.ListIssues(dir)

	if err != nil {
		t.Fatal(err)
	}

	return update(t, New(dir, issues), tea.WindowSizeMsg{Width: 100, Height: 30})
}

func update(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()

	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}

	return m
}

var (
	down  = tea.KeyPressMsg{Code: tea.KeyDown}
	up    = tea.KeyPressMsg{Code: tea.KeyUp}
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	quit  = tea.KeyPressMsg{Code: 'q', Text: "q"}
)

// row returns the line of the view containing id.
func row(t *testing.T, m Model, id string) string {
	t.Helper()

	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(line, id) {
			return line
		}
	}

	t.Fatalf("no row for %s in view:\n%s", id, m.View().Content)
	return ""
}

func TestViewListsIssues(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First issue", "open"),
		"BBB": issueSource("Second issue", "in progress"),
		"CCC": issueSource("Third issue", ""),
	})

	for _, want := range []struct{ id, title, state string }{
		{"AAA", "First issue", "open"},
		{"BBB", "Second issue", "in progress"},
		{"CCC", "Third issue", ""},
	} {
		line := row(t, m, want.id)

		if !strings.Contains(line, want.title) || !strings.Contains(line, want.state) {
			t.Errorf("row %q, want title %q and state %q", line, want.title, want.state)
		}
	}

	if line := row(t, m, "CCC"); strings.Contains(line, "open") {
		t.Errorf("row for issue without a state %q shows a state", line)
	}
}

func TestViewWithNoIssues(t *testing.T) {
	m := newTestModel(t, nil)

	if !strings.Contains(m.View().Content, "No issues in") {
		t.Errorf("View() = %q, want a message saying there are no issues", m.View().Content)
	}

	// Enter with nothing selected does nothing.
	if m = update(t, m, enter); m.issue != nil || m.err != nil {
		t.Errorf("after enter: issue = %v, err = %v, want neither", m.issue, m.err)
	}
}

func TestArrowsMoveSelection(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
		"CCC": issueSource("Third", "open"),
	})

	for _, step := range []struct {
		key  tea.KeyPressMsg
		want string
	}{
		{down, "BBB"},
		{down, "CCC"},
		{down, "CCC"},
		{up, "BBB"},
		{up, "AAA"},
		{up, "AAA"},
	} {
		m = update(t, m, step.key)

		if got := m.table.SelectedRow()[0]; got != step.want {
			t.Fatalf("selected %s after %s, want %s", got, step.key, step.want)
		}
	}
}

func TestEnterOpensSelectedIssue(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	m = update(t, m, down, enter)

	issue, ok := m.issue.(models.Issue)

	if !ok {
		t.Fatalf("issue = %T, want models.Issue", m.issue)
	}
	if issue.Id != "BBB" || issue.Title != "Second" {
		t.Errorf("opened issue %q %q, want BBB Second", issue.Id, issue.Title)
	}

	// The issue view is rendered, sized to the terminal.
	view := m.View().Content
	if !strings.Contains(view, "[BBB] Second") {
		t.Errorf("View() = %q, want the opened issue's title", view)
	}
	if got := len(strings.Split(view, "\n")); got != m.height {
		t.Errorf("View() has %d lines, want the terminal height %d", got, m.height)
	}
}

func TestEnterReparsesIssue(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("Before", "open")})

	if err := os.WriteFile(filepath.Join(m.dir, "AAA.md"), []byte(issueSource("After", "open")), 0o644); err != nil {
		t.Fatal(err)
	}

	m = update(t, m, enter)

	if got := m.issue.(models.Issue).Title; got != "After" {
		t.Errorf("opened issue title %q, want %q from the file on disk", got, "After")
	}
}

func TestEnterShowsLoadError(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	if err := os.WriteFile(filepath.Join(m.dir, "AAA.md"), []byte("Not an issue\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m = update(t, m, enter)

	if m.issue != nil {
		t.Fatal("opened an issue that failed to parse")
	}
	if m.err == nil || !strings.Contains(m.View().Content, "AAA.md") {
		t.Errorf("View() = %q, want an error naming AAA.md", m.View().Content)
	}
	if got := len(strings.Split(m.View().Content, "\n")); got > m.height {
		t.Errorf("View() has %d lines, want at most %d", got, m.height)
	}
}

func TestOpenIssueReceivesKeys(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = update(t, m, enter)

	_, cmd := m.Update(quit)

	if cmd == nil {
		t.Fatal("q in the issue view returned no command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q in the issue view returned %T, want tea.QuitMsg", cmd())
	}
}

// closeIssue sends the message produced by the back key in the open issue
// back to the model, as the program would.
func closeIssue(t *testing.T, m Model, msg tea.KeyPressMsg) Model {
	t.Helper()

	next, cmd := m.Update(msg)
	m = next.(Model)

	if cmd == nil {
		t.Fatalf("%s in the issue view returned no command", msg)
	}

	return update(t, m, cmd())
}

func TestBackKeysReturnToList(t *testing.T) {
	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyEscape},
		{Code: 'w', Text: "w"},
	} {
		m := newTestModel(t, map[string]string{
			"AAA": issueSource("First", "open"),
			"BBB": issueSource("Second", "open"),
		})

		m = update(t, m, down, enter)
		m = closeIssue(t, m, msg)

		if m.issue != nil {
			t.Fatalf("issue still open after %s", msg)
		}
		if got := m.table.SelectedRow()[0]; got != "BBB" {
			t.Errorf("selected %s after %s, want the issue that was open, BBB", got, msg)
		}
		if view := m.View().Content; !strings.Contains(view, "Issues") || !strings.Contains(view, "AAA") {
			t.Errorf("View() after %s = %q, want the issue list", msg, view)
		}

		// The list is still navigable, and the issue can be opened again.
		m = update(t, m, up, enter)
		if issue, ok := m.issue.(models.Issue); !ok || issue.Id != "AAA" {
			t.Errorf("reopened %v after %s, want AAA", m.issue, msg)
		}
	}
}

func TestListFitsTerminalResizedWhileIssueOpen(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	m = update(t, m, enter, tea.WindowSizeMsg{Width: 60, Height: 12})
	m = closeIssue(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	lines := strings.Split(m.View().Content, "\n")

	if len(lines) > 12 {
		t.Errorf("View() has %d lines, want at most 12", len(lines))
	}
	if got := m.table.Width(); got > 60 {
		t.Errorf("table width = %d, want at most 60", got)
	}
}

// writeIssue writes an issue file to the model's directory.
func writeIssue(t *testing.T, m Model, id, source string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(m.dir, id+".md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func removeIssue(t *testing.T, m Model, id string) {
	t.Helper()

	if err := os.Remove(filepath.Join(m.dir, id+".md")); err != nil {
		t.Fatal(err)
	}
}

// ids returns the IDs of the issues in the list, in order.
func ids(m Model) string {
	var ids []string
	for _, row := range m.table.Rows() {
		ids = append(ids, row[0])
	}
	return strings.Join(ids, " ")
}

func selected(m Model) string {
	if row := m.table.SelectedRow(); row != nil {
		return row[0]
	}
	return ""
}

var esc = tea.KeyPressMsg{Code: tea.KeyEscape}

func TestCloseReloadsChangedIssues(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	m = update(t, m, enter)
	writeIssue(t, m, "AAA", issueSource("A much longer title for the first issue", "closed"))
	m = closeIssue(t, m, esc)

	line := row(t, m, "AAA")

	if !strings.Contains(line, "A much longer title for the first issue") || !strings.Contains(line, "closed") {
		t.Errorf("row %q, want the title and state from disk", line)
	}
}

func TestCloseReloadsAddedAndRemovedIssues(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	m = update(t, m, enter)
	removeIssue(t, m, "BBB")
	writeIssue(t, m, "CCC", issueSource("Third", "open"))
	m = closeIssue(t, m, esc)

	if got := ids(m); got != "AAA CCC" {
		t.Errorf("listed %q, want %q", got, "AAA CCC")
	}
	if view := m.View().Content; strings.Contains(view, "BBB") || !strings.Contains(view, "Third") {
		t.Errorf("View() = %q, want CCC listed and BBB gone", view)
	}
}

func TestCloseKeepsSelectedIssueSelected(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"CCC": issueSource("Third", "open"),
	})

	m = update(t, m, down, enter)
	// BBB sorts before CCC, moving CCC down a row.
	writeIssue(t, m, "BBB", issueSource("Second", "open"))
	m = closeIssue(t, m, esc)

	if got := selected(m); got != "CCC" {
		t.Errorf("selected %q, want CCC", got)
	}
}

func TestCloseAfterSelectedIssueRemoved(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
		"CCC": issueSource("Third", "open"),
	})

	// The selection stays at the removed issue's position.
	m = update(t, m, down, enter)
	removeIssue(t, m, "BBB")
	m = closeIssue(t, m, esc)

	if got := selected(m); got != "CCC" {
		t.Errorf("selected %q, want CCC", got)
	}

	// Removing the last issue moves the selection up to the new last issue.
	m = update(t, m, enter)
	removeIssue(t, m, "CCC")
	m = closeIssue(t, m, esc)

	if got := selected(m); got != "AAA" {
		t.Errorf("selected %q, want AAA", got)
	}

	// Removing every issue leaves nothing selected.
	m = update(t, m, enter)
	removeIssue(t, m, "AAA")
	m = closeIssue(t, m, esc)

	if got := selected(m); got != "" {
		t.Errorf("selected %q, want nothing", got)
	}
	if view := m.View().Content; !strings.Contains(view, "No issues in") {
		t.Errorf("View() = %q, want a message saying there are no issues", view)
	}
}

func TestCloseWithReloadError(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	m = update(t, m, enter)
	writeIssue(t, m, "BBB", "Not an issue\n")
	m = closeIssue(t, m, esc)

	if m.issue != nil {
		t.Fatal("issue still open after a failed reload")
	}
	if got := ids(m); got != "AAA BBB" {
		t.Errorf("listed %q after a failed reload, want the previous list %q", got, "AAA BBB")
	}
	if m.err == nil || !strings.Contains(m.View().Content, "BBB.md") {
		t.Errorf("View() = %q, want an error naming BBB.md", m.View().Content)
	}
	if got := len(strings.Split(m.View().Content, "\n")); got > m.height {
		t.Errorf("View() has %d lines, want at most %d", got, m.height)
	}

	// Once the file is fixed, the next reload clears the error.
	writeIssue(t, m, "BBB", issueSource("Fixed", "open"))
	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	if m.err != nil {
		t.Errorf("err = %v after a successful reload, want nil", m.err)
	}
	if line := row(t, m, "BBB"); !strings.Contains(line, "Fixed") {
		t.Errorf("row %q, want the fixed title", line)
	}
}

// typeText sends a key press for each character in s.
func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()

	for _, r := range s {
		m = update(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	return m
}

var tab = tea.KeyPressMsg{Code: tea.KeyTab}

func TestBackWritesEditsToFile(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	// Open the issue, then edit its title.
	m = update(t, m, enter, tab, enter)
	m = typeText(t, m, " edited")
	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	issue, err := parse.LoadIssue(m.dir, "AAA")

	if err != nil {
		t.Fatal(err)
	}
	if issue.Title != "First edited" {
		t.Errorf("title on disk = %q, want %q", issue.Title, "First edited")
	}
	if line := row(t, m, "AAA"); !strings.Contains(line, "First edited") {
		t.Errorf("row %q, want the edited title", line)
	}
}

func TestQuitWritesEditsToFile(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	m = update(t, m, enter, tab, enter)
	m = typeText(t, m, " edited")
	m = update(t, m, enter)

	_, cmd := m.Update(quit)

	if cmd == nil {
		t.Fatal("q returned no command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q returned %T, want tea.QuitMsg", cmd())
	}

	issue, err := parse.LoadIssue(m.dir, "AAA")

	if err != nil {
		t.Fatal(err)
	}
	if issue.Title != "First edited" {
		t.Errorf("title on disk = %q, want %q", issue.Title, "First edited")
	}
}

func TestBackWithoutEditsLeavesFileUntouched(t *testing.T) {
	// Not in the canonical format, so it would change if rewritten.
	source := issueSource("First", "open") + "\n\n"
	m := newTestModel(t, map[string]string{"AAA": source})

	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	got, err := os.ReadFile(filepath.Join(m.dir, "AAA.md"))

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Errorf("file = %q after closing without edits, want it unchanged", got)
	}
}

func TestUnwritableEditKeepsIssueOpen(t *testing.T) {
	source := issueSource("First", "open") + "\n---\n\n[bob 2023-05-27T13:41:50-0400]\n\nComment\n"
	m := newTestModel(t, map[string]string{"AAA": source})

	// Add a horizontal rule to the comment, which can't be written.
	m = update(t, m, enter, down, enter)
	m = update(t, m, enter, enter)
	m = typeText(t, m, "---")
	m = update(t, m, enter, enter, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	next, cmd := m.Update(esc)
	m = next.(Model)

	if cmd != nil {
		t.Fatalf("esc after an unwritable edit returned %T, want the issue to stay open", cmd())
	}
	if m.issue == nil {
		t.Fatal("issue closed after an unwritable edit")
	}
	if view := m.View().Content; !strings.Contains(view, "Couldn't save changes") {
		t.Errorf("View() = %q, want the save error", view)
	}

	got, err := os.ReadFile(filepath.Join(m.dir, "AAA.md"))

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Errorf("file = %q after a refused save, want it unchanged", got)
	}

	// Leaving again discards the edit.
	m = closeIssue(t, m, esc)

	if m.issue != nil {
		t.Error("issue still open after leaving again")
	}
}
