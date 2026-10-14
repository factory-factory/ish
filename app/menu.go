package app

import (
	"encoding/json"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	dropdown "github.com/madicen/bubble-dropdown/v2"
)

// The filters offered in the filter menu, in order. The last item replaces the
// menu with the custom filter field.
const (
	presetUnclosed = iota
	presetMine
	presetCustom
)

// presetLabels are the filter menu's items, indexed by preset.
var presetLabels = []string{
	presetUnclosed: "All unclosed issues",
	presetMine:     "Issues assigned to me",
	presetCustom:   "Custom...",
}

// menuAccent is the color of the filter menu's border and highlighted item,
// matching the selected filter field and comment cards.
const menuAccent = "4"

// presetFilter returns the filter expression for a preset, other than the
// custom one, for the user who's running ish. Neither lists closed issues.
func presetFilter(preset int, user string) string {
	if preset == presetMine {
		// A JSON string is a valid filter string, with any quotes escaped.
		quoted, _ := json.Marshal(user)
		return "{assignees = " + string(quoted) + "} and " + DefaultFilter
	}

	return DefaultFilter
}

// newFilterMenu returns the filter menu, with the default filter selected.
func newFilterMenu() *dropdown.Dropdown {
	return dropdown.New(
		dropdown.WithOptions(presetLabels),
		dropdown.WithInitialIndex(presetUnclosed),
		dropdown.WithAccentColor(menuAccent),
	)
}

// menuKeys are the key bindings for the filter menu.
var menuKeys = struct {
	Open   key.Binding
	Move   key.Binding
	Choose key.Binding
	Cancel key.Binding
	Menu   key.Binding
}{
	Open: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "choose filter"),
	),
	Move: key.NewBinding(
		key.WithKeys("up", "down", "k", "j"),
		key.WithHelp("↑/↓", "move"),
	),
	Choose: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "choose"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Menu: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "filter menu"),
	),
}

// selectedMenuKeys are listed in the help footer while the filter menu is
// selected but closed.
type selectedMenuKeys struct{}

func (selectedMenuKeys) ShortHelp() []key.Binding {
	return []key.Binding{menuKeys.Open, filterKeys.Table, filterKeys.Start, keys.New, keys.Quit}
}

func (k selectedMenuKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// openMenuKeys are listed in the help footer while the filter menu is open.
type openMenuKeys struct{}

func (openMenuKeys) ShortHelp() []key.Binding {
	return []key.Binding{menuKeys.Move, menuKeys.Choose, menuKeys.Cancel}
}

func (k openMenuKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// updateMenu passes a message to the filter menu.
func (m *Model) updateMenu(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.filter.menu, cmd = m.filter.menu.Update(msg)

	return cmd
}

// chooseFilter applies the filter chosen from the menu. Choosing the custom
// filter replaces the menu with the custom filter field, ready to edit.
func (m *Model) chooseFilter(preset int) tea.Cmd {
	if preset == presetCustom {
		return m.editCustomFilter()
	}

	m.filter.preset = preset
	m.applyFilterText(presetFilter(preset, m.user))

	return nil
}

// editCustomFilter replaces the filter menu with the custom filter field, and
// starts editing it, beginning with the filter that's applied.
func (m *Model) editCustomFilter() tea.Cmd {
	m.filter.custom = true
	m.filter.menu.SetSelectedIndex(presetCustom)

	return m.editFilter()
}

// leaveCustomFilter replaces the custom filter field with the filter menu,
// and applies the filter that was chosen from it before.
func (m *Model) leaveCustomFilter() {
	m.filter.custom = false
	m.filter.err = nil
	m.filter.menu.SetSelectedIndex(m.filter.preset)
	m.applyFilterText(presetFilter(m.filter.preset, m.user))
	m.selectFilter()
}
