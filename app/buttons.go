package app

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

// The parts of the custom filter row that can be selected, in order: the
// field, then the buttons to its right.
const (
	partField = iota
	partReset
	partSave
)

// buttonLabels are the custom filter buttons' labels, indexed by part.
var buttonLabels = []string{
	partReset: "Reset",
	partSave:  "Save",
}

// buttonPadding is the space either side of a button's label, within its
// background, and buttonGap the space between buttons, and between the field
// and the first button at the least.
const (
	buttonPadding = 2
	buttonGap     = 2
)

// partKeys are the key bindings for moving between the custom filter field
// and its buttons, and pressing them.
var partKeys = struct {
	Move  key.Binding
	Reset key.Binding
	Save  key.Binding
}{
	Move: key.NewBinding(
		key.WithKeys("left", "right", "h", "l"),
		key.WithHelp("←/→", "move"),
	),
	Reset: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "reset to the filter menu"),
	),
	Save: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "save"),
	),
}

// renderButton renders the button for part, highlighted if it's selected.
func (m Model) renderButton(part int) string {
	style := styles.Button
	if m.filter.selected && m.filter.part == part {
		switch part {
		case partReset:
			style = styles.ButtonCancelSelected
		case partSave:
			style = styles.ButtonSelected
		}
	}

	padding := strings.Repeat(" ", buttonPadding)

	return style.Render(padding + buttonLabels[part] + padding)
}

// renderButtons renders the custom filter buttons, side by side.
func (m Model) renderButtons() string {
	return m.renderButton(partReset) + strings.Repeat(" ", buttonGap) + m.renderButton(partSave)
}

// buttonsWidth returns the width of the custom filter buttons, with the gap
// before them.
func (m Model) buttonsWidth() int {
	return buttonGap + lipgloss.Width(m.renderButtons())
}

// movePart moves the selection between the custom filter field and its
// buttons: right with → or l, and left otherwise, stopping at either end.
func (m *Model) movePart(msg tea.KeyPressMsg) {
	delta := -1
	if s := msg.String(); s == "right" || s == "l" {
		delta = 1
	}

	m.filter.part = min(max(m.filter.part+delta, partField), partSave)
}

// pressPart edits the custom filter field, or presses the selected button.
// Reset returns to the filter menu. Save doesn't do anything yet.
func (m *Model) pressPart() tea.Cmd {
	switch m.filter.part {
	case partField:
		return m.editFilter()

	case partReset:
		m.leaveCustomFilter()
	}

	return nil
}
