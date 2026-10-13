package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	slash = tea.KeyPressMsg{Code: '/', Text: "/"}
	ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	ctrlU = tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
)

// editEmptyFilter starts editing the filter and clears its text.
func editEmptyFilter(t *testing.T, m Model) Model {
	t.Helper()

	return update(t, m, slash, ctrlU)
}

// filterModel returns a model listing three issues in different states.
func filterModel(t *testing.T) Model {
	t.Helper()

	return newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "closed"),
		"CCC": issueSource("Third", "in progress"),
	})
}

// plainView returns the model's view without ANSI escape sequences.
func plainView(m Model) string {
	return escapes.ReplaceAllString(m.View().Content, "")
}

// applyFilter types text into the filter field and applies it.
func applyFilter(t *testing.T, m Model, text string) Model {
	t.Helper()

	m = editEmptyFilter(t, m)
	m = typeText(t, m, text)

	return update(t, m, enter)
}

func TestFilterFieldInitially(t *testing.T) {
	m := filterModel(t)
	view := plainView(m)

	if !strings.Contains(view, "Issue Filter") || strings.Contains(view, "Issues\n") {
		t.Errorf("View() = %q, want the filter label in place of the title", view)
	}
	// The default filter is applied, so the closed issue isn't listed.
	if !strings.Contains(view, DefaultFilter) {
		t.Errorf("View() = %q, want the default filter", view)
	}

	// The field has a grey rounded border until it's selected.
	if !strings.Contains(view, "╭") || strings.Contains(view, "┏") {
		t.Error("filter field doesn't have the unselected border")
	}
	if !strings.Contains(m.View().Content, "\x1b[90m╭") {
		t.Error("filter field's border isn't grey")
	}

	if got := ids(m); got != "AAA CCC" {
		t.Errorf("listed %q, want the issues that aren't closed", got)
	}
}

func TestApplyFilter(t *testing.T) {
	m := filterModel(t)

	// Editing shows the raw text in a selected field.
	m = editEmptyFilter(t, m)
	m = typeText(t, m, "{state!='closed'}")

	if !m.filter.editing || !strings.Contains(plainView(m), "┏") {
		t.Fatal("not editing in a selected field after /")
	}
	if view := plainView(m); !strings.Contains(view, "{state!='closed'}") {
		t.Errorf("View() = %q, want the raw filter text", view)
	}
	if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, "enter apply") || !strings.Contains(footer, "esc cancel") {
		t.Errorf("footer = %q, want the editing keys", footer)
	}

	// Applying lists only the matching issues and shows the parsed filter.
	m = update(t, m, enter)

	if m.filter.editing || !m.filter.selected {
		t.Errorf("editing, selected = %v, %v after applying, want false, true", m.filter.editing, m.filter.selected)
	}
	if got := ids(m); got != "AAA CCC" {
		t.Errorf("listed %q, want the unclosed issues", got)
	}
	if view := plainView(m); !strings.Contains(view, "{state != 'closed'}") {
		t.Errorf("View() = %q, want the parsed filter", view)
	}

	// Editing again shows the text as it was typed.
	m = update(t, m, enter)

	if got := m.filter.input.Value(); got != "{state!='closed'}" {
		t.Errorf("editing text = %q, want the text as typed", got)
	}
}

func TestInvalidFilter(t *testing.T) {
	m := filterModel(t)
	m = applyFilter(t, m, "{state = }")

	if !m.filter.editing {
		t.Error("stopped editing after an invalid filter")
	}
	if view := plainView(m); !strings.Contains(view, "column 10: expected a value") {
		t.Errorf("View() = %q, want the parse error", view)
	}
	if got := ids(m); got != "AAA CCC" {
		t.Errorf("listed %q, want the list unchanged", got)
	}
	if got := len(strings.Split(m.View().Content, "\n")); got > m.height {
		t.Errorf("View() has %d lines, want at most %d", got, m.height)
	}

	// Fixing it applies it and clears the error.
	m = update(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	m = typeText(t, m, "'open'")
	m = update(t, m, enter)

	if m.filter.err != nil || ids(m) != "AAA" {
		t.Errorf("err, listed = %v, %q, want nil, AAA", m.filter.err, ids(m))
	}
}

func TestCancelAndClearFilter(t *testing.T) {
	m := filterModel(t)
	m = applyFilter(t, m, "{state = 'open'}")

	// Cancelling keeps the applied filter.
	m = update(t, m, enter)
	m = typeText(t, m, " or {state = 'closed'}")
	m = update(t, m, esc)

	if m.filter.editing || m.filter.text != "{state = 'open'}" || ids(m) != "AAA" {
		t.Errorf("editing, text, listed = %v, %q, %q, want the filter unchanged", m.filter.editing, m.filter.text, ids(m))
	}

	// Applying an empty filter lists every issue again.
	m = update(t, m, enter)

	for range len("{state = 'open'}") {
		m = update(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	m = update(t, m, enter)

	if m.filter.node != nil || ids(m) != "AAA BBB CCC" || !strings.Contains(plainView(m), "No filter") {
		t.Errorf("node, listed = %v, %q, want no filter and every issue", m.filter.node, ids(m))
	}
}

func TestSelectFilterField(t *testing.T) {
	m := filterModel(t)

	// Up from the top of the table selects the field.
	m = update(t, m, up)

	if !m.filter.selected || !strings.Contains(plainView(m), "┏") {
		t.Fatal("filter field not selected after up from the top row")
	}
	if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, "enter edit filter") {
		t.Errorf("footer = %q, want the selected field's keys", footer)
	}

	// Enter edits it, rather than opening an issue.
	m = update(t, m, enter)

	if !m.filter.editing || m.issue != nil {
		t.Fatalf("editing, issue = %v, %v, want to edit the filter", m.filter.editing, m.issue)
	}

	// Down goes back to the table once editing stops.
	m = update(t, m, esc, down)

	if m.filter.selected || strings.Contains(plainView(m), "┏") {
		t.Error("filter field still selected after down")
	}

	// Up from a lower row moves up the table instead.
	m = update(t, m, down, up)

	if m.filter.selected || selected(m) != "AAA" {
		t.Errorf("selected filter, row = %v, %q, want the first row", m.filter.selected, selected(m))
	}
}

func TestKeysWhileEditingFilter(t *testing.T) {
	m := filterModel(t)
	m = editEmptyFilter(t, m)

	// Keys bound in the list are typed into the filter.
	m = typeText(t, m, "qn/")

	if !m.filter.editing || m.filter.input.Value() != "qn/" || m.issue != nil {
		t.Errorf("editing, text, issue = %v, %q, %v, want the keys typed", m.filter.editing, m.filter.input.Value(), m.issue)
	}

	// Ctrl+c still quits.
	_, cmd := m.Update(ctrlC)

	if cmd == nil {
		t.Fatal("ctrl+c returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c while editing the filter didn't quit")
	}
}

func TestFilterPersistsAfterIssue(t *testing.T) {
	m := filterModel(t)
	m = applyFilter(t, m, "{state != 'closed'}")
	m = update(t, m, down)

	// Close the first issue in the issue view, by editing its state tag;
	// returning to the list applies the filter again, so it's no longer
	// listed.
	m = update(t, m, enter, tab, tab, enter)
	m = update(t, m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	m = typeText(t, m, "closed")
	m = update(t, m, enter)
	m = closeIssue(t, m, esc)

	if got := ids(m); got != "CCC" {
		t.Errorf("listed %q, want only CCC once AAA is closed", got)
	}
	if m.filter.text != "{state != 'closed'}" {
		t.Errorf("filter = %q, want it kept", m.filter.text)
	}
}

func TestNoIssuesMatchFilter(t *testing.T) {
	m := filterModel(t)
	m = applyFilter(t, m, "{state = 'missing'}")

	if got := ids(m); got != "" {
		t.Errorf("listed %q, want none", got)
	}
	if view := plainView(m); !strings.Contains(view, "No issues match the filter") {
		t.Errorf("View() = %q, want a message that nothing matches", view)
	}
}

func TestFilterFieldGrows(t *testing.T) {
	m := filterModel(t)
	m = editEmptyFilter(t, m)

	before := lipgloss.Height(m.renderHeading())
	m = typeText(t, m, strings.Repeat("{state = 'open'} or ", 12)+"{state = 'closed'}")
	after := lipgloss.Height(m.renderHeading())

	if after <= before {
		t.Errorf("heading height = %d after a long filter, want more than %d", after, before)
	}
	if after-before > maxFilterHeight-1 {
		t.Errorf("heading grew by %d lines, want at most %d", after-before, maxFilterHeight-1)
	}

	// The list shrinks to keep the screen the terminal's height.
	if got := len(strings.Split(m.View().Content, "\n")); got != m.height {
		t.Errorf("View() has %d lines, want %d", got, m.height)
	}

	// Applying shows the whole parsed filter, wrapped.
	m = update(t, m, enter)

	if got := ids(m); got != "AAA BBB" {
		t.Errorf("listed %q, want the open and closed issues", got)
	}
	if got := len(strings.Split(m.View().Content, "\n")); got != m.height {
		t.Errorf("View() has %d lines after applying, want %d", got, m.height)
	}
}

func TestEmptyListMessages(t *testing.T) {
	// A project whose only issue is closed has issues, but none match the
	// default filter.
	m := newTestModel(t, map[string]string{"AAA": issueSource("Done", "closed")})

	if view := plainView(m); !strings.Contains(view, "No issues match the filter") {
		t.Errorf("View() = %q, want a message that nothing matches", view)
	}

	// Clearing the filter lists it.
	m = editEmptyFilter(t, m)
	m = update(t, m, enter)

	if got := ids(m); got != "AAA" {
		t.Errorf("listed %q, want the closed issue", got)
	}

	// A project with no issues says so, whatever the filter.
	m = newTestModel(t, nil)

	if view := plainView(m); !strings.Contains(view, "No issues in") {
		t.Errorf("View() = %q, want a message that there are no issues", view)
	}
}
