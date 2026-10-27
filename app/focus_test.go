package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// selectedPrefix returns the escape sequences the selected row starts with,
// while the table has the focus or doesn't.
func selectedPrefix(focused bool) string {
	return strings.Split(tableStyles(focused).Selected.Render("x"), "x")[0]
}

// rowLine returns the line of the view, with its escape sequences, showing
// the issue with the given ID.
func rowLine(t *testing.T, m Model, id string) string {
	t.Helper()

	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(escapes.ReplaceAllString(line, ""), "  "+id+"  ") {
			return line
		}
	}

	t.Fatalf("no row for %s in view:\n%s", id, m.View().Content)
	return ""
}

func TestTableStyles(t *testing.T) {
	focused, unfocused := tableStyles(true).Selected, tableStyles(false).Selected

	if !focused.GetBold() || focused.GetBackground() != lipgloss.Color("4") {
		t.Errorf("focused selection bold %v on %v, want bold on blue (4)", focused.GetBold(), focused.GetBackground())
	}
	if unfocused.GetBold() || unfocused.GetBackground() != lipgloss.Color("8") {
		t.Errorf("unfocused selection bold %v on %v, want normal text on grey (8)", unfocused.GetBold(), unfocused.GetBackground())
	}
	if selectedPrefix(true) == selectedPrefix(false) {
		t.Error("focused and unfocused selections render the same")
	}
}

func TestSelectionDimmedWhileFilterSelected(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
	})

	if line := rowLine(t, m, "AAA"); !strings.HasPrefix(strings.TrimLeft(line, " "), selectedPrefix(true)) {
		t.Errorf("row %q, want the focused selection", line)
	}

	// Up from the first issue selects the filter menu, dimming the row.
	m = update(t, m, up)

	if !m.filter.selected || selected(m) != "AAA" {
		t.Fatalf("filter selected, selected issue = %v, %q, want the filter, with AAA still selected", m.filter.selected, selected(m))
	}
	if line := rowLine(t, m, "AAA"); !strings.HasPrefix(strings.TrimLeft(line, " "), selectedPrefix(false)) {
		t.Errorf("row %q, want the unfocused selection", line)
	}

	// Down returns to the table, highlighting it again.
	m = update(t, m, down)

	if line := rowLine(t, m, "AAA"); m.filter.selected || !strings.HasPrefix(strings.TrimLeft(line, " "), selectedPrefix(true)) {
		t.Errorf("filter selected %v, row %q, want the focused selection back", m.filter.selected, line)
	}
}

func TestSlashKeepsSelectedIssue(t *testing.T) {
	m := newTestModel(t, map[string]string{
		"AAA": issueSource("First", "open"),
		"BBB": issueSource("Second", "open"),
		"CCC": issueSource("Third", "open"),
	})
	m = update(t, m, down, down, slash)

	if !m.filter.selected || selected(m) != "CCC" {
		t.Fatalf("filter selected, selected issue = %v, %q, want the filter selected, with CCC still selected", m.filter.selected, selected(m))
	}
	if line := rowLine(t, m, "CCC"); !strings.HasPrefix(strings.TrimLeft(line, " "), selectedPrefix(false)) {
		t.Errorf("row %q, want the unfocused selection", line)
	}

	// Returning to the table keeps it.
	m = update(t, m, down)

	if selected(m) != "CCC" {
		t.Errorf("selected %q after returning to the table, want CCC", selected(m))
	}
	if line := rowLine(t, m, "CCC"); !strings.HasPrefix(strings.TrimLeft(line, " "), selectedPrefix(true)) {
		t.Errorf("row %q, want the focused selection back", line)
	}
}
