package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/filter"
)

// menuModel returns a model for alice, listing issues assigned to her and to
// others, some of them closed.
func menuModel(t *testing.T) Model {
	t.Helper()
	stubUser(t, "alice@example.com", "")

	return newTestModel(t, map[string]string{
		"AAA": issueWithMetadata("Mine", "- state: open", "- assignees: alice@example.com"),
		"BBB": issueWithMetadata("Shared", "- state: open", "- assignees: bob, alice@example.com"),
		"CCC": issueWithMetadata("Mine, closed", "- state: closed", "- assignees: alice@example.com"),
		"DDD": issueWithMetadata("Bob's", "- state: open", "- assignees: bob"),
		"EEE": issueWithMetadata("Mine, bracketed", "- state: open", "- assignees: <alice@example.com>"),
	})
}

// openMenu selects the filter menu and opens it.
func openMenu(t *testing.T, m Model) Model {
	t.Helper()

	m = update(t, m, up, enter)

	if !m.filter.menu.Open() {
		t.Fatal("filter menu didn't open")
	}

	return m
}

// choose opens the filter menu and chooses the item at index.
func choose(t *testing.T, m Model, index int) Model {
	t.Helper()

	m = openMenu(t, m)

	// The menu opens on the selected item, so start from the top.
	for range presetLabels {
		m = update(t, m, up)
	}

	for range index {
		m = update(t, m, down)
	}

	return press(t, m, enter)
}

func TestFilterMenuOptions(t *testing.T) {
	m := openMenu(t, menuModel(t))
	view := plainView(m)

	for _, label := range presetLabels {
		if !strings.Contains(view, label) {
			t.Errorf("open menu doesn't show %q:\n%s", label, view)
		}
	}

	if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, "enter choose") || !strings.Contains(footer, "esc cancel") {
		t.Errorf("footer = %q, want the open menu's keys", footer)
	}
	if got := len(strings.Split(m.View().Content, "\n")); got != m.height {
		t.Errorf("View() has %d lines with the menu open, want %d", got, m.height)
	}
}

func TestChooseIssuesAssignedToMe(t *testing.T) {
	m := choose(t, menuModel(t), presetAssignedToMe)

	if m.filter.menu.Open() || m.filter.preset != presetAssignedToMe {
		t.Fatalf("menu open, preset = %v, %d, want the menu closed on %d", m.filter.menu.Open(), m.filter.preset, presetAssignedToMe)
	}
	if m.filter.text != `{assignees = .user.email} and {state != 'closed'}` {
		t.Errorf("filter = %q, want alice's unclosed issues", m.filter.text)
	}

	// Issues assigned to alice, alone or with others, that aren't closed,
	// whether her email is in angle brackets or not.
	if got := ids(m); got != "AAA BBB EEE" {
		t.Errorf("listed %q, want the unclosed issues assigned to alice", got)
	}
	if view := plainView(m); !strings.Contains(view, "│ Assigned to me") {
		t.Errorf("View() = %q, want the menu showing the chosen filter", view)
	}

	// Choosing the first filter again applies it.
	m = choose(t, m, presetUnclosed)

	if got := ids(m); got != "AAA BBB DDD EEE" {
		t.Errorf("listed %q, want the unclosed issues", got)
	}
}

func TestPresetFilters(t *testing.T) {
	for _, tt := range []struct {
		preset int
		want   string
	}{
		{presetUnclosed, DefaultFilter},
		// The user's email is looked up from the application state.
		{presetAssignedToMe, "{assignees = .user.email} and " + DefaultFilter},
		{presetCreatedByMe, "{@author = .user.email} and " + DefaultFilter},
	} {
		text := presetFilter(tt.preset)

		if text != tt.want {
			t.Errorf("presetFilter(%s) = %q, want %q", presetLabels[tt.preset], text, tt.want)
		}
		if _, err := filter.Parse(text); err != nil {
			t.Errorf("presetFilter(%s) = %q, which doesn't parse: %v", presetLabels[tt.preset], text, err)
		}
	}
}

func TestChooseCreatedByMe(t *testing.T) {
	stubUser(t, "alice@example.com", "")

	m := newTestModel(t, map[string]string{
		"AAA": "# Hers\n\n[alice@example.com 2023-05-26T23:35:36-0400]\n\n- state: open\n",
		"BBB": "# Hers, closed\n\n[\"Alice\" <alice@example.com> 2023-05-26T23:35:36-0400]\n\n- state: closed\n",
		"CCC": "# Bob's\n\n[bob@example.com 2023-05-26T23:35:36-0400]\n\n- state: open\n",
	})
	m = choose(t, m, presetCreatedByMe)

	if got := ids(m); got != "AAA" {
		t.Errorf("listed %q, want alice's unclosed issue", got)
	}
}

func TestChooseCustomFilter(t *testing.T) {
	m := choose(t, menuModel(t), presetAssignedToMe)
	m = choose(t, m, presetCustom)

	// The custom field replaces the menu, editing the applied filter.
	if !m.filter.custom || !m.filter.editing {
		t.Fatalf("custom, editing = %v, %v, want to edit a custom filter", m.filter.custom, m.filter.editing)
	}
	if got := m.filter.input.Value(); got != `{assignees = .user.email} and {state != 'closed'}` {
		t.Errorf("custom filter starts as %q, want the applied filter", got)
	}
	if view := plainView(m); strings.Contains(view, "▼") || !strings.Contains(view, "┏") {
		t.Errorf("View() = %q, want the custom field in place of the menu", view)
	}

	// Apply a custom filter.
	m = update(t, m, ctrlU)
	m = typeText(t, m, "{state = 'closed'}")
	m = update(t, m, enter)

	if got := ids(m); got != "CCC" {
		t.Errorf("listed %q, want the closed issue", got)
	}
	// Esc no longer leaves the custom filter.
	if m = update(t, m, esc); !m.filter.custom {
		t.Fatal("esc on the custom filter returned to the menu, want only Reset to")
	}

	// Reset returns to the menu, applying the filter chosen from it before.
	m = update(t, m, right)

	if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, "enter reset to the filter menu") {
		t.Errorf("footer = %q, want a way back to the menu", footer)
	}

	m = update(t, m, enter)

	if m.filter.custom || m.filter.menu.SelectedIndex() != presetAssignedToMe || !m.filter.menu.Focused() {
		t.Errorf("custom, menu selection, focused = %v, %d, %v, want the menu back on %d",
			m.filter.custom, m.filter.menu.SelectedIndex(), m.filter.menu.Focused(), presetAssignedToMe)
	}
	if got := ids(m); got != "AAA BBB EEE" {
		t.Errorf("listed %q, want alice's unclosed issues again", got)
	}
}

func TestSlashEditsCustomFilter(t *testing.T) {
	m := update(t, menuModel(t), slash)

	if !m.filter.custom || !m.filter.editing || m.filter.input.Value() != DefaultFilter {
		t.Errorf("custom, editing, text = %v, %v, %q, want to edit the default filter",
			m.filter.custom, m.filter.editing, m.filter.input.Value())
	}
}

func TestCancelFilterMenu(t *testing.T) {
	m := openMenu(t, menuModel(t))
	m = update(t, m, down)
	m = press(t, m, esc)

	if m.filter.menu.Open() || m.filter.menu.SelectedIndex() != presetUnclosed || m.filter.text != DefaultFilter {
		t.Errorf("open, selection, filter = %v, %d, %q, want the menu closed and unchanged",
			m.filter.menu.Open(), m.filter.menu.SelectedIndex(), m.filter.text)
	}
}

func TestKeysWhileMenuOpen(t *testing.T) {
	m := openMenu(t, menuModel(t))

	// Keys bound in the list go to the open menu instead.
	for _, msg := range []tea.KeyPressMsg{quit, {Code: 'n', Text: "n"}, slash} {
		next, cmd := m.Update(msg)
		m = next.(Model)

		if cmd != nil {
			if _, ok := cmd().(tea.QuitMsg); ok {
				t.Errorf("%s quit with the menu open", msg)
			}
		}
		if m.issue != nil || m.filter.custom {
			t.Errorf("%s acted on the list with the menu open", msg)
		}
	}

	// Ctrl+c still quits.
	_, cmd := m.Update(ctrlC)

	if cmd == nil {
		t.Fatal("ctrl+c returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c with the menu open didn't quit")
	}
}

func TestMenuSeparatesCustomFilter(t *testing.T) {
	m := openMenu(t, menuModel(t))
	view := plainView(m)

	mine := strings.Index(view, presetLabels[presetRecentlyModified])
	custom := strings.Index(view, presetLabels[presetCustom])

	if mine < 0 || custom < 0 {
		t.Fatalf("View() = %q, want every item", view)
	}

	between := strings.Split(view[mine:custom], "\n")
	if len(between) != 3 || !strings.Contains(between[1], "────") {
		t.Errorf("between the saved filters and Custom...: %q, want a rule on its own line", between)
	}

	// The rule is the color of unselected fields' borders.
	rule := lipgloss.NewStyle().Foreground(lipgloss.Color(separatorColor)).Render("─")
	if separatorColor != "11" || !strings.Contains(m.View().Content, strings.TrimSuffix(rule, "\x1b[m")+"──") {
		t.Errorf("View() = %q, want the rule in color 11", m.View().Content)
	}

	// The rule isn't selectable: down from the last saved filter chooses
	// the custom filter.
	m = choose(t, menuModel(t), presetCustom)

	if !m.filter.custom {
		t.Error("choosing the item after the rule didn't choose Custom...")
	}
}
