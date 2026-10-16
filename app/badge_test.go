package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

func TestBadgeView(t *testing.T) {
	view := Badge{}.View().Content
	lines := strings.Split(escapes.ReplaceAllString(view, ""), "\n")
	want := []string{"╭     ╮", "  ish  ", "╰     ╯"}

	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("badge = %q, want %q", lines, want)
	}

	for _, part := range []string{
		styles.BadgeCorner.Render("╭     ╮"),
		styles.BadgeCorner.Render("╰     ╯"),
		styles.BadgeName.Render("ish"),
	} {
		if !strings.Contains(view, part) {
			t.Errorf("badge %q doesn't contain %q", view, part)
		}
	}

	if styles.BadgeCorner.GetForeground() != lipgloss.Color("6") || styles.BadgeName.GetForeground() != lipgloss.Color("5") {
		t.Errorf("badge colors = %v, %v, want aqua (6) and magenta (5)",
			styles.BadgeCorner.GetForeground(), styles.BadgeName.GetForeground())
	}
}

func TestBadgeBesideFilter(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	lines := strings.Split(escapes.ReplaceAllString(m.View().Content, ""), "\n")
	left := styles.Header.GetPaddingLeft()
	top := styles.Header.GetPaddingTop()

	// The badge is in the upper left corner, and the filter is 5 spaces
	// after it, on the line with its name.
	for i, want := range []string{"╭     ╮", "  ish       Filter: │ All unclosed issues", "╰     ╯"} {
		if got := lines[top+i][left:]; !strings.HasPrefix(got, want) {
			t.Errorf("heading line %d = %q, want it to start with %q", i, got, want)
		}
	}

	// The menu's panel opens under its trigger, after the badge and label.
	m = update(t, m, up, enter)
	lines = strings.Split(escapes.ReplaceAllString(m.View().Content, ""), "\n")
	column := left + m.filterIndent() + labelWidth()

	if got := []rune(lines[top+3]); len(got) <= column || got[column] != '╭' {
		t.Errorf("line under the trigger = %q, want the menu's panel from column %d", string(got), column)
	}
	if got := lines[top+4]; !strings.Contains(got, "All unclosed issues") {
		t.Errorf("panel line %q, want the menu's items", got)
	}
}

func TestCustomFilterBesideBadge(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = update(t, m, slash)
	lines := strings.Split(escapes.ReplaceAllString(m.View().Content, ""), "\n")
	top := styles.Header.GetPaddingTop()

	// The custom field's text lines up with the badge's name.
	if got := lines[top+1]; !strings.Contains(got, "  ish       Filter: ") {
		t.Errorf("heading line %q, want the badge's name, then the label", got)
	}

	// The field fits beside the badge.
	for _, line := range lines {
		if w := lipgloss.Width(line); w > m.width {
			t.Errorf("line %q is %d wide, want at most %d", line, w, m.width)
		}
	}
}

func TestFilterErrorLinesUpWithFilter(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = editEmptyFilter(t, m)
	m = typeText(t, m, "{state = }")
	m = update(t, m, enter)

	indent := styles.Header.GetPaddingLeft() + m.filterIndent()

	for _, line := range strings.Split(escapes.ReplaceAllString(m.View().Content, ""), "\n") {
		if i := strings.Index(line, "column 10: expected a value"); i >= 0 {
			if i != indent {
				t.Errorf("error line %q starts at column %d, want %d, under the filter", line, i, indent)
			}

			return
		}
	}

	t.Errorf("View() = %q, want the filter's error", m.View().Content)
}
