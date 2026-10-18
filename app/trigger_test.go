package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

func TestFilterBoxWidth(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})

	for _, tt := range []struct {
		terminal, want int
	}{
		// The box fills the space after the label...
		{100, 100 - styles.Header.GetHorizontalFrameSize() - m.filterIndent() - labelWidth()},
		// ...within 60 to 90 columns, including the border.
		{200, 90},
		{80, 60},
	} {
		m = update(t, m, tea.WindowSizeMsg{Width: tt.terminal, Height: 30})

		if got := m.filterBoxWidth(); got != tt.want {
			t.Errorf("filterBoxWidth() in %d columns = %d, want %d", tt.terminal, got, tt.want)
		}
		if got := lipgloss.Width(m.renderTrigger()); got != tt.want {
			t.Errorf("trigger in %d columns is %d wide, want %d", tt.terminal, got, tt.want)
		}
	}
}

func TestTrigger(t *testing.T) {
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	lines := strings.Split(escapes.ReplaceAllString(m.renderTrigger(), ""), "\n")

	// A thin, rounded box, with the filter's name and the arrow at its right
	// edge.
	want := []string{"╭", "│ All unclosed issues", "╰"}
	for i, prefix := range want {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("trigger line %d = %q, want it to start with %q", i, lines[i], prefix)
		}
	}
	if !strings.HasSuffix(lines[1], " ▼ │") {
		t.Errorf("trigger line %q, want the arrow at the right edge", lines[1])
	}

	// It's grey until it's selected, and stays selected while the menu is
	// open.
	if m.renderTrigger() != renderedTrigger(m, styles.FilterTrigger) {
		t.Error("unselected trigger isn't in the unselected style")
	}

	m = update(t, m, up)

	if m.renderTrigger() != renderedTrigger(m, styles.FilterTriggerSelected) {
		t.Error("selected trigger isn't in the selected style")
	}

	m = update(t, m, enter)

	if !m.filter.menu.Open() || m.renderTrigger() != renderedTrigger(m, styles.FilterTriggerSelected) {
		t.Errorf("open menu %v: trigger isn't in the selected style", m.filter.menu.Open())
	}
}

// renderedTrigger renders the trigger's content in style.
func renderedTrigger(m Model, style lipgloss.Style) string {
	line := strings.Split(escapes.ReplaceAllString(m.renderTrigger(), ""), "\n")[1]
	content := strings.TrimSuffix(strings.TrimPrefix(line, "│ "), " │")

	return style.Render(content)
}

func TestTriggerStyles(t *testing.T) {
	for _, tt := range []struct {
		name               string
		style              lipgloss.Style
		border, foreground string
		bold               bool
	}{
		{"unselected", styles.FilterTrigger, "11", "7", false},
		{"selected", styles.FilterTriggerSelected, "4", "15", true},
	} {
		if tt.style.GetBorderStyle() != lipgloss.RoundedBorder() {
			t.Errorf("%s trigger border isn't thin and rounded", tt.name)
		}
		if tt.style.GetBorderTopForeground() != lipgloss.Color(tt.border) || tt.style.GetForeground() != lipgloss.Color(tt.foreground) {
			t.Errorf("%s trigger colors = border %v, text %v, want %s, %s", tt.name,
				tt.style.GetBorderTopForeground(), tt.style.GetForeground(), tt.border, tt.foreground)
		}
		if tt.style.GetBold() != tt.bold {
			t.Errorf("%s trigger bold = %v, want %v", tt.name, tt.style.GetBold(), tt.bold)
		}
	}

	// The custom field shares the border colors, with a thick border when
	// selected to tell it apart.
	if styles.FilterBox.GetBorderTopForeground() != styles.FilterTrigger.GetBorderTopForeground() ||
		styles.FilterBoxSelected.GetBorderTopForeground() != styles.FilterTriggerSelected.GetBorderTopForeground() {
		t.Error("custom field border colors differ from the trigger's")
	}
}

func TestCustomFieldMatchesTrigger(t *testing.T) {
	// In a wide terminal, the field and the buttons beside it fit at the
	// trigger's width.
	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = update(t, m, tea.WindowSizeMsg{Width: 200, Height: 30})
	trigger := m.renderTrigger()

	m = update(t, m, slash, enter)

	if m.filter.editing || !m.filter.custom {
		t.Fatalf("editing, custom = %v, %v, want a custom filter at rest", m.filter.editing, m.filter.custom)
	}

	field := m.renderCustomFilter()

	if lipgloss.Width(field) != lipgloss.Width(trigger) || lipgloss.Height(field) != lipgloss.Height(trigger) {
		t.Errorf("custom field is %dx%d, want the trigger's %dx%d",
			lipgloss.Width(field), lipgloss.Height(field), lipgloss.Width(trigger), lipgloss.Height(trigger))
	}

	// A filter too long for the field is cut short to fit.
	m = applyFilter(t, m, strings.Repeat("{state != 'closed'} and ", 5)+"{state != 'open'}")
	m = update(t, m, down)
	field = m.renderCustomFilter()

	if lipgloss.Height(field) != lipgloss.Height(trigger) || lipgloss.Width(field) != lipgloss.Width(trigger) {
		t.Errorf("custom field with a long filter is %dx%d, want %dx%d",
			lipgloss.Width(field), lipgloss.Height(field), lipgloss.Width(trigger), lipgloss.Height(trigger))
	}
	if !strings.Contains(field, "…") {
		t.Errorf("custom field %q, want the long filter cut short with an ellipsis", field)
	}
}
