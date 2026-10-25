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

// changeFile writes an issue file in the model's directory, as an outside
// editor would, without updating the metadata database.
func changeFile(t *testing.T, m Model, id, source string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(m.dir, id+".md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

// deleteFile deletes an issue file in the model's directory, without
// updating the metadata database.
func deleteFile(t *testing.T, m Model, id string) {
	t.Helper()

	if err := os.Remove(filepath.Join(m.dir, id+".md")); err != nil {
		t.Fatal(err)
	}
}

// titleOnDisk returns the title of the issue's file.
func titleOnDisk(t *testing.T, m Model, id string) string {
	t.Helper()

	issue, err := parse.LoadIssue(m.dir, id)
	if err != nil {
		t.Fatal(err)
	}

	return issue.Title
}

// editTitle opens the selected issue and starts editing its title, adding
// text to it. The edit is kept if commit is true, and goes on otherwise.
func editTitle(t *testing.T, m Model, text string, commit bool) Model {
	t.Helper()

	m = update(t, m, enter, tab, enter)
	m = typeText(t, m, text)

	if commit {
		m = update(t, m, enter)
	}

	return m
}

func TestWaitForChanges(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	if cmd := m.waitForChanges(); cmd != nil {
		t.Error("waitForChanges() without Watch = a command, want nil")
	}

	changes := make(chan struct{}, 1)
	m.Watch(changes)

	changes <- struct{}{}

	if msg := m.waitForChanges()(); msg != (changedMsg{}) {
		t.Errorf("waitForChanges() after a change = %#v, want changedMsg", msg)
	}

	close(changes)

	if msg := m.waitForChanges()(); msg != nil {
		t.Errorf("waitForChanges() after the channel closed = %#v, want nil", msg)
	}
}

func TestChangesWaitForTheNextChange(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	changes := make(chan struct{}, 1)
	m.Watch(changes)

	if _, cmd := m.Update(changedMsg{}); cmd == nil {
		t.Error("Update(changedMsg) returned no command, want one waiting for the next change")
	}
}

func TestChangesUpdateList(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
		"CCC": issueSource("Third", "open"),
	})
	m = update(t, m, down)

	changeFile(t, m, "AAA", issueSource("First, renamed", "open"))
	changeFile(t, m, "DDD", issueSource("Fourth", "open"))
	deleteFile(t, m, "CCC")
	m = update(t, m, changedMsg{})

	if got := ids(m); got != "AAA BBB DDD" {
		t.Errorf("list = %q, want the added and removed issues", got)
	}

	if line := row(t, m, "AAA"); !strings.Contains(line, "First, renamed") {
		t.Errorf("row %q, want the changed title", line)
	}

	if got := selected(m); got != "BBB" {
		t.Errorf("selected %q, want BBB still selected", got)
	}
}

func TestChangeReloadsOpenIssue(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = update(t, m, enter, tab)

	changeFile(t, m, "AAA", issueSource("Changed outside", "in review"))
	m = update(t, m, changedMsg{})

	view := m.View().Content

	if !strings.Contains(view, "Changed outside") || !strings.Contains(view, "in review") {
		t.Errorf("view doesn't show the changes:\n%s", view)
	}

	if strings.Contains(view, "outside ish") {
		t.Errorf("view shows the conflict dialog without edits:\n%s", view)
	}

	// The title stays selected, so enter edits it.
	m = update(t, m, enter)
	m = typeText(t, m, "!")
	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	if got := titleOnDisk(t, m, "AAA"); got != "Changed outside!" {
		t.Errorf("title on disk = %q, want the reloaded title edited", got)
	}
}

func TestOwnSaveIsNotAConflict(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = editTitle(t, m, " edited", true)
	m = closeIssue(t, m, esc)

	// The save's own notification arrives after the issue is closed and
	// opened again.
	m = update(t, m, enter, changedMsg{})

	if view := m.View().Content; strings.Contains(view, "outside ish") || !strings.Contains(view, "First edited") {
		t.Errorf("view after ish's own save:\n%s", view)
	}
}

func TestChangeWithEditsAsksWhichToKeep(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = editTitle(t, m, " edited", true)

	changeFile(t, m, "AAA", issueSource("Changed outside", "open"))
	m = update(t, m, changedMsg{})

	view := m.View().Content

	for _, want := range []string{"AAA.md was changed outside ish.", discardLabel, keepLabel} {
		if !strings.Contains(view, want) {
			t.Errorf("view doesn't contain %q:\n%s", want, view)
		}
	}

	// Other keys wait for the question to be answered.
	next, cmd := m.Update(quit)
	m = next.(Model)

	if cmd != nil || m.conflict == nil {
		t.Errorf("q during the dialog returned %v, with the dialog shown %v; want it ignored", cmd, m.conflict != nil)
	}
}

func TestDiscardEditsLoadsChanges(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = editTitle(t, m, " edited", true)

	changeFile(t, m, "AAA", issueSource("Changed outside", "open"))
	m = update(t, m, changedMsg{}, left, enter)

	if m.conflict != nil {
		t.Fatal("the dialog is still shown")
	}

	if view := m.View().Content; !strings.Contains(view, "Changed outside") || strings.Contains(view, "First edited") {
		t.Errorf("view doesn't show the outside changes:\n%s", view)
	}

	if issue := m.issue.(models.Issue); issue.Unsaved() {
		t.Error("the issue has unsaved changes after discarding them")
	}

	m = closeIssue(t, m, esc)

	if got := titleOnDisk(t, m, "AAA"); got != "Changed outside" {
		t.Errorf("title on disk = %q, want the outside change", got)
	}
}

func TestKeepEditsOverwritesChanges(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = editTitle(t, m, " edited", true)

	changeFile(t, m, "AAA", issueSource("Changed outside", "open"))
	m = update(t, m, changedMsg{}, enter)

	if m.conflict != nil {
		t.Fatal("the dialog is still shown")
	}

	if got := titleOnDisk(t, m, "AAA"); got != "First edited" {
		t.Errorf("title on disk = %q, want the edits saved over the change", got)
	}

	// Saving the edits isn't another change.
	m = update(t, m, changedMsg{})

	if m.conflict != nil {
		t.Error("saving the kept edits asked again")
	}
}

func TestKeepEditsGoesOnEditing(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	// The title is still being edited when the file changes.
	m = editTitle(t, m, " edited", false)

	changeFile(t, m, "AAA", issueSource("Changed outside", "open"))
	m = update(t, m, changedMsg{})

	if m.conflict == nil {
		t.Fatal("no dialog for a change while a field is being edited")
	}

	m = update(t, m, enter)

	// The file has the issue as it was before the field being edited.
	if got := titleOnDisk(t, m, "AAA"); got != "First" {
		t.Errorf("title on disk = %q, want the issue as it was saved over the change", got)
	}

	// And the edit can be finished.
	m = typeText(t, m, " more")
	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	if got := titleOnDisk(t, m, "AAA"); got != "First edited more" {
		t.Errorf("title on disk = %q, want the finished edit", got)
	}
}

func TestDeletedIssueCloses(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})
	m = update(t, m, enter)

	deleteFile(t, m, "AAA")
	m = update(t, m, changedMsg{})

	if m.issue != nil {
		t.Fatal("the deleted issue is still open")
	}

	if got := ids(m); got != "BBB" {
		t.Errorf("list = %q, want only BBB", got)
	}

	if view := m.View().Content; !strings.Contains(view, "AAA was deleted outside ish") {
		t.Errorf("view doesn't say the issue was deleted:\n%s", view)
	}
}

func TestDeletedSingleIssueShowsList(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	opened, err := Open(m.dir, "AAA")
	if err != nil {
		t.Fatal(err)
	}

	m = update(t, opened, tea.WindowSizeMsg{Width: 100, Height: 30})

	deleteFile(t, m, "AAA")
	m = update(t, m, changedMsg{})

	if m.issue != nil || m.single {
		t.Fatal("the deleted issue is still open")
	}

	if got := ids(m); got != "BBB" {
		t.Errorf("list = %q, want only BBB", got)
	}
}

func TestDeletedIssueWithEdits(t *testing.T) {
	for _, tt := range []struct {
		name   string
		keys   []tea.Msg
		exists bool
	}{
		{"keep", []tea.Msg{enter}, true},
		{"discard", []tea.Msg{left, enter}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
			m = editTitle(t, m, " edited", true)

			deleteFile(t, m, "AAA")
			m = update(t, m, changedMsg{})

			if view := m.View().Content; !strings.Contains(view, "AAA.md was deleted outside ish.") {
				t.Fatalf("view doesn't ask about the deleted file:\n%s", view)
			}

			m = update(t, m, tt.keys...)

			_, err := os.Stat(filepath.Join(m.dir, "AAA.md"))

			if exists := err == nil; exists != tt.exists {
				t.Errorf("file exists %v, want %v", exists, tt.exists)
			}

			if open := m.issue != nil; open != tt.exists {
				t.Errorf("issue open %v, want %v", open, tt.exists)
			}

			if tt.exists {
				if got := titleOnDisk(t, m, "AAA"); got != "First edited" {
					t.Errorf("title on disk = %q, want the kept edits", got)
				}
			}
		})
	}
}

func TestUnreadableChangeKeepsIssue(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = update(t, m, enter)

	changeFile(t, m, "AAA", "Not an issue\n")
	m = update(t, m, changedMsg{})

	view := m.View().Content

	if !strings.Contains(view, "First") || !strings.Contains(view, "Couldn't load the changes made to AAA.md outside ish") {
		t.Errorf("view doesn't keep the issue with the error:\n%s", view)
	}

	// Once the file can be read, it's loaded, and the error goes.
	changeFile(t, m, "AAA", issueSource("Fixed", "open"))
	m = update(t, m, changedMsg{})

	if view := m.View().Content; !strings.Contains(view, "Fixed") || strings.Contains(view, "Couldn't load") {
		t.Errorf("view after the file was fixed:\n%s", view)
	}
}
