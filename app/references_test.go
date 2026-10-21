package app

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/store"
)

func TestUserNames(t *testing.T) {
	users := []store.User{
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Alice", Email: "alice@work.example"},
		{Name: "", Email: "nobody@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
	}

	if got, want := userNames(users), []string{"Alice", "Bob"}; !slices.Equal(got, want) {
		t.Errorf("userNames() = %q, want %q", got, want)
	}
}

func TestOpenedIssueSuggestsIssueIDs(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	// Add a tag to the first issue, accepting the suggested reference to
	// the second.
	m = update(t, m, enter, tea.KeyPressMsg{Code: 'm', Text: "m"})
	m = typeText(t, m, "related:#BB")
	m = update(t, m, enter)

	issue := m.issue.(models.Issue)
	tag := issue.Metadata[len(issue.Metadata)-1]

	if tag.Key != "related" || !slices.Equal(models.Canonical(tag.Values), []string{"#BBB"}) {
		t.Errorf("tag %s: %q, want related: [#BBB]", tag.Key, tag.Values)
	}
}
