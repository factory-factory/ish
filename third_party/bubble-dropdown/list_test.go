package dropdownv2

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestListModelSetStyles(t *testing.T) {
	m := newListModel([]string{"a", "b", "c"}, 0, 8, 1, defaultListStyles("62"))

	m.setStyles(defaultListStyles("#FF0000"))

	view := m.View()
	wantBorder := fgSeq(t, "#FF0000")
	wantCursor := bgSeq(t, "#FF0000")
	if !strings.Contains(view, wantBorder) {
		t.Fatalf("View() missing new border fg sequence %q\n%q", wantBorder, view)
	}
	if !strings.Contains(view, wantCursor) {
		t.Fatalf("View() missing new cursor bg sequence %q\n%q", wantCursor, view)
	}
}

func TestListModelSetStylesNilSafe(t *testing.T) {
	var m *listModel
	// Must not panic.
	m.setStyles(defaultListStyles("#FF0000"))
}

// Local patch for ish: separators.
func TestListModelSeparators(t *testing.T) {
	m := newListModel([]string{"a", "b", "c"}, 0, 8, 1, defaultListStyles(""))
	m.setSeparators([]int{0, 2, 9})

	lines := strings.Split(m.View(), "\n")
	want := []string{"a", "b", "─", "c"}

	if len(lines) != len(want)+2 {
		t.Fatalf("View() has %d lines, want %d:\n%s", len(lines), len(want)+2, m.View())
	}
	for i, w := range want {
		if !strings.Contains(lines[i+1], w) {
			t.Errorf("line %d = %q, want %q", i+1, lines[i+1], w)
		}
	}

	if _, h := m.PanelSize(); h != len(lines) {
		t.Errorf("PanelSize() height = %d, want %d", h, len(lines))
	}

	// The rule isn't an item: the cursor moves past it, and the mouse can't
	// choose it.
	m.cursor = 1
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if m.cursor != 2 {
		t.Errorf("cursor = %d after down from b, want 2 (c)", m.cursor)
	}
	if _, ok := m.itemAtY(3); ok {
		t.Error("itemAtY() of the rule found an item")
	}
	if abs, ok := m.itemAtY(4); !ok || abs != 2 {
		t.Errorf("itemAtY(4) = %d, %v, want c (2)", abs, ok)
	}

	// Scrolled so that c is at the top, the rule above it isn't shown.
	m = newListModel([]string{"a", "b", "c"}, 2, 1, 1, defaultListStyles(""))
	m.setSeparators([]int{2})

	if got := len(strings.Split(m.View(), "\n")); got != 3 {
		t.Errorf("scrolled View() has %d lines, want 3:\n%s", got, m.View())
	}
}
