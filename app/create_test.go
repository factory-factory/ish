package app

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
)

var (
	newKey  = tea.KeyPressMsg{Code: 'n', Text: "n"}
	created = time.Date(2026, 10, 10, 23, 7, 0, 0, time.FixedZone("", -4*60*60))
)

// stubUser replaces the clock and the lookups of the current user for the
// duration of a test.
func stubUser(t *testing.T, email string, emailErr error, name string, nameErr error) {
	t.Helper()

	oldNow, oldEmail, oldName := now, gitUserEmail, currentUserName

	now = func() time.Time { return created }
	gitUserEmail = func(string) (string, error) { return email, emailErr }
	currentUserName = func() (string, error) { return name, nameErr }

	t.Cleanup(func() {
		now, gitUserEmail, currentUserName = oldNow, oldEmail, oldName
	})
}

// openNew presses n and returns the model and the new issue.
func openNew(t *testing.T, m Model) (Model, models.Issue) {
	t.Helper()

	m = update(t, m, newKey)
	issue, ok := m.issue.(models.Issue)

	if !ok {
		t.Fatalf("issue = %T after n, want models.Issue", m.issue)
	}

	return m, issue
}

func TestNewKeyOpensEmptyIssue(t *testing.T) {
	stubUser(t, "alice@example.com", nil, "alice", nil)
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	m, issue := openNew(t, m)

	if !regexp.MustCompile(`^[0-9A-Z]{5}-[0-9A-Z]{5}$`).MatchString(issue.Id) {
		t.Errorf("Id = %q, want a timestamp and random part joined by -", issue.Id)
	}
	if issue.Title != "" || len(issue.Metadata) != 0 {
		t.Errorf("Title, Metadata = %q, %v, want empty", issue.Title, issue.Metadata)
	}
	if issue.Author != "alice@example.com" || issue.DateCreated != "2026-10-10T23:07:00-04:00" {
		t.Errorf("byline = %q %q, want alice@example.com 2026-10-10T23:07:00-04:00", issue.Author, issue.DateCreated)
	}
	if len(issue.Conversation) != 1 || *issue.Conversation[0].Content != "" ||
		issue.Conversation[0].Author != "alice@example.com" {
		t.Errorf("Conversation = %+v, want one empty comment by alice@example.com", issue.Conversation)
	}

	view := m.View().Content

	if !strings.Contains(view, "["+issue.Id+"]") || !strings.Contains(view, "alice@example.com") {
		t.Errorf("View() = %q, want the new issue's ID and author", view)
	}

	// The issue exists only in memory until it's edited.
	if _, err := os.Stat(filepath.Join(m.dir, issue.Id+".md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("issue file exists before any edits (err = %v)", err)
	}
}

func TestNewIssueWithoutEditsIsDiscarded(t *testing.T) {
	stubUser(t, "alice@example.com", nil, "alice", nil)
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	// The first esc cancels naming the issue, and the second closes it.
	m, issue := openNew(t, m)
	m = update(t, m, esc)
	m = closeIssue(t, m, esc)

	if m.issue != nil {
		t.Fatal("issue still open after esc")
	}
	if got := ids(m); got != "AAA" {
		t.Errorf("listed %q, want only AAA", got)
	}
	if _, err := os.Stat(filepath.Join(m.dir, issue.Id+".md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("issue file written for an unedited issue (err = %v)", err)
	}
}

func TestEditedNewIssueIsSavedAndSelected(t *testing.T) {
	stubUser(t, "alice@example.com", nil, "alice", nil)
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"ZZZ": issueSource("Last", "open"),
	})

	// Create an issue and name it, which can be done straight away.
	m, issue := openNew(t, m)
	m = typeText(t, m, "A new issue")
	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	saved, err := parse.LoadIssue(m.dir, issue.Id)

	if err != nil {
		t.Fatalf("LoadIssue() error = %v", err)
	}
	if saved.Title != "A new issue" || saved.Author != "alice@example.com" || len(saved.Conversation) != 1 {
		t.Errorf("saved issue = %+v, want the titled new issue with one comment", saved)
	}

	if got := selected(m); got != issue.Id {
		t.Errorf("selected %q, want the new issue %q", got, issue.Id)
	}
	if line := row(t, m, issue.Id); !strings.Contains(line, "A new issue") {
		t.Errorf("row %q, want the new issue's title", line)
	}
}

func TestNewIssueFromEmptyList(t *testing.T) {
	stubUser(t, "alice@example.com", nil, "alice", nil)
	m := newTestModel(t, nil)

	// Leave the title blank and write a comment instead.
	m, issue := openNew(t, m)
	m = update(t, m, esc, down, enter)
	m = typeText(t, m, "First comment")
	m = update(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = closeIssue(t, m, esc)

	if got := ids(m); got != issue.Id {
		t.Errorf("listed %q, want the new issue %q", got, issue.Id)
	}
}

func TestCurrentUser(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		emailErr error
		user     string
		userErr  error
		want     string
	}{
		{"git email", "alice@example.com", nil, "alice", nil, "alice@example.com"},
		{"git fails", "", errors.New("no git"), "alice", nil, "alice"},
		{"git email unset", "", nil, "alice", nil, "alice"},
		{"everything fails", "", errors.New("no git"), "", errors.New("no user"), "unknown"},
		{"spaces removed", "Alice Smith", nil, "alice", nil, "AliceSmith"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubUser(t, tt.email, tt.emailErr, tt.user, tt.userErr)

			if got := currentUser(t.TempDir()); got != tt.want {
				t.Errorf("currentUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewIssueOpensNamingTitle(t *testing.T) {
	stubUser(t, "alice@example.com", nil, "alice", nil)
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	next, cmd := m.Update(newKey)
	m = next.(Model)

	if cmd == nil {
		t.Error("n returned no command, want the title input's cursor to start")
	}

	// The title is selected and being edited, with the help footer showing the
	// editing keys.
	view := m.View().Content

	if !strings.Contains(view, "enter") || !strings.Contains(view, "save") || !strings.Contains(view, "cancel") {
		t.Errorf("View() = %q, want the editing keys in the help footer", view)
	}

	m = typeText(t, m, "Named")
	m = update(t, m, enter)

	if issue := m.issue.(models.Issue); issue.Title != "Named" {
		t.Errorf("Title = %q, want %q", issue.Title, "Named")
	}
}

func TestExistingIssueDoesNotOpenEditing(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	// Typing in an issue that has a title doesn't change it.
	m = update(t, m, enter)
	m = typeText(t, m, "xyz")

	if issue := m.issue.(models.Issue); issue.Title != "First" {
		t.Errorf("Title = %q, want it unchanged", issue.Title)
	}
}
