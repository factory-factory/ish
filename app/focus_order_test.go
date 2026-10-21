package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var shiftTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}

// focusName names the stop that has the focus, for test messages.
func focusName(m Model) string {
	switch {
	case !m.filter.selected:
		return "table"
	case !m.filter.custom:
		return "menu"
	}

	return []string{partField: "field", partReset: "reset", partSave: "save"}[m.filter.part]
}

// pressAll sends keys, and returns where the focus is after each.
func pressAll(t *testing.T, m Model, keys ...tea.KeyPressMsg) (Model, []string) {
	t.Helper()

	var stops []string

	for _, k := range keys {
		m = update(t, m, k)
		stops = append(stops, focusName(m))
	}

	return m, stops
}

func TestTabBetweenMenuAndTable(t *testing.T) {
	m := menuModel(t)

	if _, got := pressAll(t, m, tab, tab, tab); strings.Join(got, " ") != "menu table menu" {
		t.Errorf("tab moves focus to %v, want menu, table, menu", got)
	}
	if _, got := pressAll(t, m, shiftTab, shiftTab); strings.Join(got, " ") != "menu table" {
		t.Errorf("shift+tab moves focus to %v, want menu, table", got)
	}

	// The menu is focused, so enter opens it.
	m = update(t, m, tab, enter)

	if !m.filter.menu.Open() {
		t.Error("enter after tab didn't open the menu")
	}

	// While the menu is open, tab doesn't move the focus.
	if m = update(t, m, tab); !m.filter.menu.Open() || focusName(m) != "menu" {
		t.Errorf("tab with the menu open: open %v, focus %s, want the menu still open", m.filter.menu.Open(), focusName(m))
	}
}

func TestTabThroughCustomFilter(t *testing.T) {
	m := customModel(t, 100)
	m = update(t, m, down)

	if _, got := pressAll(t, m, tab, tab, tab, tab, tab); strings.Join(got, " ") != "field reset save table field" {
		t.Errorf("tab moves focus to %v, want field, reset, save, table, field", got)
	}
	if _, got := pressAll(t, m, shiftTab, shiftTab, shiftTab, shiftTab, shiftTab); strings.Join(got, " ") != "save reset field table save" {
		t.Errorf("shift+tab moves focus to %v, want save, reset, field, table, save", got)
	}

	// The focused button is highlighted, and the field's border isn't.
	m = update(t, m, tab, tab)

	if strings.Contains(m.renderCustomFilter(), "┏") || !strings.Contains(escapes.ReplaceAllString(m.renderFooter(), ""), "enter reset to the filter menu") {
		t.Errorf("with Reset focused: field %q, footer %q, want Reset focused", m.renderCustomFilter(), m.renderFooter())
	}
}

func TestTabAppliesEditedFilter(t *testing.T) {
	for _, tt := range []struct {
		key  tea.KeyPressMsg
		want string
	}{
		{tab, "reset"},
		{shiftTab, "table"},
	} {
		m := editEmptyFilter(t, filterModel(t))
		m = typeText(t, m, "{state = 'open'}")
		m = update(t, m, tt.key)

		if m.filter.editing || m.filter.text != "{state = 'open'}" {
			t.Errorf("after %s: editing, filter = %v, %q, want the filter applied", tt.key, m.filter.editing, m.filter.text)
		}
		if got := focusName(m); got != tt.want {
			t.Errorf("after %s: focus = %s, want %s", tt.key, got, tt.want)
		}
		if got := ids(m); got != "AAA" {
			t.Errorf("after %s: listed %q, want the open issue", tt.key, got)
		}
	}
}

func TestTabKeepsInvalidFilter(t *testing.T) {
	m := editEmptyFilter(t, filterModel(t))

	if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, "tab/shift+tab apply and move") {
		t.Errorf("footer %q while editing, want tab to apply and move", footer)
	}

	m = typeText(t, m, "{state = }")
	m = update(t, m, tab)

	if !m.filter.editing || m.filter.err == nil || focusName(m) != "field" {
		t.Errorf("editing, err, focus = %v, %v, %s, want the field kept with the error", m.filter.editing, m.filter.err, focusName(m))
	}
}

func TestFocusHelp(t *testing.T) {
	m := menuModel(t)

	for _, state := range []string{"table", "menu", "custom"} {
		switch state {
		case "menu":
			m = update(t, m, tab)
		case "custom":
			m = customModel(t, 100)
		}

		if footer := escapes.ReplaceAllString(m.renderFooter(), ""); !strings.Contains(footer, "tab/shift+tab focus") {
			t.Errorf("%s: footer %q, want the focus keys", state, footer)
		}
	}
}
