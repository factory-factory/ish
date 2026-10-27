package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

var left = tea.KeyPressMsg{Code: tea.KeyLeft}

// customModel returns a model with the custom filter field selected, at rest,
// in a terminal of the given width.
func customModel(t *testing.T, width int) Model {
	t.Helper()

	m := newTestModel(t, map[string]string{"AAA": issueSource("First", "open")})
	m = update(t, m, tea.WindowSizeMsg{Width: width, Height: 30})
	m = update(t, editCustom(t, m), enter)

	if !m.filter.custom || m.filter.editing || !m.filter.selected {
		t.Fatalf("custom, editing, selected = %v, %v, %v, want the custom field selected at rest",
			m.filter.custom, m.filter.editing, m.filter.selected)
	}

	return m
}

func TestButtonsBesideCustomField(t *testing.T) {
	for _, width := range []int{80, 100, 200} {
		m := customModel(t, width)
		lines := strings.Split(escapes.ReplaceAllString(m.View().Content, ""), "\n")
		line := lines[styles.Header.GetPaddingTop()+1]

		// The buttons are on the line with the field's text, at the
		// heading's right edge, with two spaces either side of each label.
		i := strings.Index(line, "  Reset      Save  ")
		if i < 0 {
			t.Errorf("in %d columns, line %q, want the buttons after the field", width, line)
			continue
		}
		if got := lipgloss.Width(line); got != width {
			t.Errorf("in %d columns, line is %d wide, want it to fit", width, got)
		}
		if end := lipgloss.Width(line[:i+len("  Reset      Save  ")]); end != width-styles.Header.GetPaddingRight() {
			t.Errorf("in %d columns, Save ends at column %d, want %d", width, end, width-styles.Header.GetPaddingRight())
		}

		// The field shrinks to fit the buttons, but the trigger doesn't.
		available := width - styles.Header.GetHorizontalFrameSize() - m.filterIndent() - labelWidth()

		if got, want := m.customBoxWidth(), min(m.filterBoxWidth(), available-m.buttonsWidth()); got != want {
			t.Errorf("in %d columns, custom field is %d wide, want %d", width, got, want)
		}
		if got := m.filterBoxWidth(); got < minFilterBoxWidth {
			t.Errorf("in %d columns, trigger is %d wide, want at least %d", width, got, minFilterBoxWidth)
		}
	}
}

func TestButtonStyles(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		style                  lipgloss.Style
		foreground, background string
	}{
		{"button", styles.Button, "7", "8"},
		{"selected button", styles.ButtonSelected, "15", "5"},
	} {
		if tt.style.GetForeground() != lipgloss.Color(tt.foreground) || tt.style.GetBackground() != lipgloss.Color(tt.background) {
			t.Errorf("%s colors = %v on %v, want %s on %s", tt.name,
				tt.style.GetForeground(), tt.style.GetBackground(), tt.foreground, tt.background)
		}
		if tt.style.GetBorderStyle() != (lipgloss.Border{}) {
			t.Errorf("%s has a border, want none", tt.name)
		}
	}
}

func TestMoveBetweenFieldAndButtons(t *testing.T) {
	m := customModel(t, 100)

	if !strings.Contains(m.renderCustomFilter(), "┏") || strings.Contains(m.renderButtons(), styles.ButtonSelected.Render("  Reset  ")) {
		t.Error("with the field selected, want its border selected and no button highlighted")
	}

	for _, tt := range []struct {
		key   tea.KeyPressMsg
		part  int
		help  string
		label string
	}{
		{right, partReset, "enter reset to the filter menu", "Reset"},
		{right, partSave, "enter save", "Save"},
		{right, partSave, "enter save", "Save"}, // Stops at the last button.
		{left, partReset, "enter reset to the filter menu", "Reset"},
		{tea.KeyPressMsg{Code: 'h', Text: "h"}, partField, "enter edit filter", ""},
		{left, partField, "enter edit filter", ""}, // Stops at the field.
		{tea.KeyPressMsg{Code: 'l', Text: "l"}, partReset, "enter reset to the filter menu", "Reset"},
	} {
		m = update(t, m, tt.key)

		if m.filter.part != tt.part {
			t.Fatalf("after %s: part = %d, want %d", tt.key, m.filter.part, tt.part)
		}
		if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, tt.help) || !strings.Contains(footer, "←/→ move") {
			t.Errorf("after %s: footer %q, want %q and the move keys", tt.key, footer, tt.help)
		}

		var style lipgloss.Style

		switch tt.part {
		case partReset:
			style = styles.ButtonCancelSelected
		case partSave:
			style = styles.ButtonSelected
		}

		selected := tt.label != "" && strings.Contains(m.renderButtons(), style.Render("  "+tt.label+"  "))
		if tt.label != "" && !selected {
			t.Errorf("after %s: buttons %q, want %s highlighted", tt.key, m.renderButtons(), tt.label)
		}
		if tt.label != "" && strings.Contains(m.renderCustomFilter(), "┏") {
			t.Errorf("after %s: field border is selected while a button is", tt.key)
		}
	}
}

func TestSaveDoesNothingYet(t *testing.T) {
	m := customModel(t, 100)
	m = update(t, m, right, right)

	next, cmd := m.Update(enter)
	m = next.(Model)

	if cmd != nil || !m.filter.custom || m.filter.editing || m.filter.part != partSave {
		t.Errorf("enter on Save: cmd %v, custom %v, editing %v, part %d, want nothing to happen",
			cmd, m.filter.custom, m.filter.editing, m.filter.part)
	}
}

func TestEnterOnFieldEdits(t *testing.T) {
	m := customModel(t, 100)
	m = update(t, m, right, left, enter)

	if !m.filter.editing {
		t.Error("enter on the field didn't start editing it")
	}

	// While editing, the arrows move within the text, not between parts.
	m = update(t, m, right)

	if !m.filter.editing || m.filter.part != partField {
		t.Errorf("editing, part = %v, %d after → while editing, want the field still edited", m.filter.editing, m.filter.part)
	}
}

func TestLeavingRowSelectsFieldAgain(t *testing.T) {
	m := customModel(t, 100)
	m = update(t, m, right, down)

	if m.filter.selected || m.filter.part != partField {
		t.Fatalf("selected, part = %v, %d after ↓, want the issues selected", m.filter.selected, m.filter.part)
	}

	m = update(t, m, up)

	if !m.filter.selected || m.filter.part != partField {
		t.Errorf("selected, part = %v, %d after ↑, want the field selected", m.filter.selected, m.filter.part)
	}
}
