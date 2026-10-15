package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/models"
)

// linkedSources are issues where AAA is blocked by BBB.
var linkedSources = map[string]string{
	"AAA": "# First\n\n[author 2023-05-26T23:35:36-0400]\n\n- relation.blocked-by: #BBB\n",
	"BBB": issueSource("Second", "open"),
}

var right = tea.KeyPressMsg{Code: tea.KeyRight}

// follow sends msg, then sends the messages its commands produce back to the
// model as the program would, while they're messages from the issue view
// asking to open or close an issue. It returns every message the commands
// produced.
func follow(t *testing.T, m Model, msg tea.Msg) (Model, []tea.Msg) {
	t.Helper()

	var produced []tea.Msg

	for {
		next, cmd := m.Update(msg)
		m = next.(Model)

		if cmd == nil {
			return m, produced
		}

		msg = cmd()
		produced = append(produced, msg)

		switch msg.(type) {
		case models.OpenMsg, models.CloseMsg:
		default:
			return m, produced
		}
	}
}

// sent reports whether msgs include a message of the same type as want.
func sent(msgs []tea.Msg, want tea.Msg) bool {
	for _, msg := range msgs {
		if reflect.TypeOf(msg) == reflect.TypeOf(want) {
			return true
		}
	}

	return false
}

// followLink selects the link in AAA's metadata and opens it.
func followLink(t *testing.T, m Model) (Model, []tea.Msg) {
	t.Helper()

	m = update(t, m, tab, tab, right)

	return follow(t, m, enter)
}

func openIssueId(m Model) string {
	if issue, ok := m.issue.(models.Issue); ok {
		return issue.Id
	}

	return ""
}

func TestLinkOpensIssueFromList(t *testing.T) {
	m := newTestModel(t, linkedSources)
	m = update(t, m, enter)

	m, msgs := followLink(t, m)

	if !sent(msgs, models.OpenMsg{}) {
		t.Fatalf("enter on the link sent %v, want a models.OpenMsg", msgs)
	}
	if got := openIssueId(m); got != "BBB" || m.openId != "BBB" {
		t.Fatalf("open issue %q, openId %q after following the link, want BBB", got, m.openId)
	}
	if view := m.View().Content; !strings.Contains(view, "[BBB] Second") {
		t.Errorf("View() = %q, want the linked issue", view)
	}

	// Going back returns to the list, with the linked issue selected.
	m, _ = follow(t, m, esc)

	if m.issue != nil || selected(m) != "BBB" {
		t.Errorf("after esc: issue %v, selected %q, want the list with BBB selected", m.issue, selected(m))
	}
}

func TestLinkedIssueIsLinkedOnlyIfItExists(t *testing.T) {
	sources := map[string]string{
		"AAA": "# First\n\n[author 2023-05-26T23:35:36-0400]\n\n- relation.blocked-by: #CCC\n",
	}

	m := newTestModel(t, sources)
	m = update(t, m, enter, tab, tab)

	if m, msgs := follow(t, m, right); msgs != nil || openIssueId(m) != "AAA" {
		t.Errorf("→ on a reference to a missing issue sent %v, want nothing", msgs)
	}

	m, msgs := follow(t, m, enter)
	if sent(msgs, models.OpenMsg{}) {
		t.Errorf("enter after → on a reference to a missing issue sent %v, want no models.OpenMsg", msgs)
	}
}

func TestLinkToIssueThatCantBeLoaded(t *testing.T) {
	m := newTestModel(t, linkedSources)
	m = update(t, m, enter)

	// The file is removed after the issue was opened, while it's still
	// linked.
	if err := os.Remove(filepath.Join(m.dir, "BBB.md")); err != nil {
		t.Fatal(err)
	}

	m, _ = followLink(t, m)

	if m.issue != nil || m.err == nil {
		t.Fatalf("issue %v, err %v, want the list with an error", m.issue, m.err)
	}
	if !strings.Contains(m.err.Error(), "BBB.md") || !strings.Contains(m.View().Content, "no such file") {
		t.Errorf("err = %v, View() = %q, want the error naming BBB.md shown", m.err, m.View().Content)
	}
}

func TestOpenShowsSingleIssue(t *testing.T) {
	list := newTestModel(t, linkedSources)

	m, err := Open(list.dir, "AAA")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if view := m.View().Content; !strings.Contains(view, "[AAA] First") {
		t.Fatalf("View() = %q, want the issue", view)
	}

	// Links open in place.
	m, _ = followLink(t, m)

	if got := openIssueId(m); got != "BBB" {
		t.Fatalf("open issue %q after following the link, want BBB", got)
	}

	// Esc quits, rather than showing the list.
	for _, key := range []tea.KeyPressMsg{esc, {Code: 'w', Text: "w"}} {
		if _, msgs := follow(t, m, key); !sent(msgs, tea.QuitMsg{}) {
			t.Errorf("%s sent %v, want tea.QuitMsg", key, msgs)
		}
	}
}

func TestOpenMissingIssue(t *testing.T) {
	list := newTestModel(t, linkedSources)

	if _, err := Open(list.dir, "ZZZ"); err == nil {
		t.Error("Open() of a missing issue error = nil, want an error")
	}
}
