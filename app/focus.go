package app

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// focusKeys move the focus between the filter menu, or the custom filter
// field and its buttons, and the issue list.
var focusKeys = struct {
	Next  key.Binding
	Prev  key.Binding
	Apply key.Binding
}{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab/shift+tab", "focus"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
	),
	// Apply is only listed in the help footer while the filter is being
	// edited, when moving the focus applies it first.
	Apply: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab/shift+tab", "apply and move"),
	),
}

// focusTable stands for the issue list among the focus stops, which are
// otherwise the parts of the custom filter row.
const focusTable = -1

// focusStops returns the places the focus moves between, in order: the
// filter menu, or the custom filter field and its buttons, then the issue
// list. The menu is a single stop, partField.
func (m Model) focusStops() []int {
	if m.filter.custom {
		return []int{partField, partReset, partSave, focusTable}
	}

	return []int{partField, focusTable}
}

// focusStop returns the stop that has the focus.
func (m Model) focusStop() int {
	if !m.filter.selected {
		return focusTable
	}

	return m.filter.part
}

// moveFocus moves the focus to the next stop, or the previous one if delta is
// negative, wrapping around at either end.
func (m *Model) moveFocus(delta int) {
	stops := m.focusStops()
	i := max(slices.Index(stops, m.focusStop()), 0)
	next := stops[(i+delta+len(stops))%len(stops)]

	if next == focusTable {
		m.selectTable()
		return
	}

	m.filter.part = next
	m.selectFilter()
}

// focusDelta returns the direction a key moves the focus in, 1 or -1, or 0 if
// it doesn't move it.
func focusDelta(msg tea.KeyPressMsg) int {
	switch {
	case key.Matches(msg, focusKeys.Next):
		return 1
	case key.Matches(msg, focusKeys.Prev):
		return -1
	}

	return 0
}
