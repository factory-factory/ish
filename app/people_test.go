package app

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/store"
)

// addUsers adds users to the model's metadata database, as indexing the git
// history would, and reloads the list.
func addUsers(t *testing.T, m Model, users ...store.User) Model {
	t.Helper()

	db, err := store.Open(filepath.Join(m.dir, store.DatabaseName))
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	for i, u := range users {
		if _, err := db.Exec("INSERT INTO user (id, name, email, dateCreated) VALUES (?, ?, ?, ?)", "id"+string(rune('a'+i)), u.Name, u.Email, "2026-10-01T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}

	m.reload()

	return m
}

func TestAssigneeColumnShowsNames(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueWithMetadata("First", "- state: open", "- assignees: <alice@example.com>, <nobody@ex.com>"),
	})
	m = addUsers(t, m, store.User{Name: "Alice", Email: "alice@example.com"})

	if line := row(t, m, "AAA"); !strings.Contains(line, "@Alice, nobody@ex.com") {
		t.Errorf("row %q, want Alice's name and the unknown email", line)
	}
}

func TestOpenedIssueShowsNames(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": "# First\n\n[alice@example.com 2023-05-26T23:35:36-0400]\n\n- assignees: <alice@example.com>\n",
	})
	m = addUsers(t, m, store.User{Name: "Alice", Email: "alice@example.com"})
	m = update(t, m, enter)

	issue := m.issue.(models.Issue)

	if got := issue.Metadata[0].Values[0]; got.Display != "Alice" || got.Canonical != "<alice@example.com>" {
		t.Errorf("assignee = %+v, want Alice's name for her email", got)
	}
	if view := plainView(m); !strings.Contains(view, "@Alice opened this issue") {
		t.Errorf("View() = %q, want the byline by name", view)
	}

	// Typing a name in a tag saves the email.
	m = update(t, m, tea.KeyPressMsg{Code: 'm', Text: "m"})
	m = typeText(t, m, "watchers:@Alice")
	m = update(t, m, enter)

	if tag := m.issue.(models.Issue).Metadata[1]; strings.Join(models.Canonical(tag.Values), ",") != "<alice@example.com>" {
		t.Errorf("watchers = %q, want Alice's email", models.Canonical(tag.Values))
	}
}

func TestLoadPeopleWithoutDatabase(t *testing.T) {
	if _, ok := loadPeople(t.TempDir()).Name("alice@example.com"); ok {
		t.Error("loadPeople() without a database has a user")
	}
}
