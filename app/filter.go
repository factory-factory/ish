package app

import (
	"math"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/styles"
)

// DefaultFilter is the filter the issue list starts with, which lists every
// issue that isn't closed.
const DefaultFilter = "{state != 'closed'}"

// filterField is the field above the issue list that holds the filter
// expression selecting which issues are listed.
type filterField struct {
	// text is the applied filter as it was typed, and node is its parsed
	// form. An empty filter has a nil node, and lists every issue.
	text string
	node filter.Node

	// selected reports whether the field has the focus instead of the
	// table, and editing whether its text is being edited in input.
	selected bool
	editing  bool
	input    textarea.Model

	// err is the error from the last attempt to apply an edited filter.
	err error
}

// maxFilterHeight is the most lines the filter field grows to while editing.
// Longer filters scroll within it.
const maxFilterHeight = 5

// filterKeys are the key bindings for the filter field.
var filterKeys = struct {
	Edit   key.Binding
	Start  key.Binding
	Table  key.Binding
	Apply  key.Binding
	Cancel key.Binding
}{
	Edit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "edit filter"),
	),
	Start: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Table: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "issues"),
	),
	Apply: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "apply"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// selectedFilterKeys are listed in the help footer while the filter field is
// selected.
type selectedFilterKeys struct{}

func (selectedFilterKeys) ShortHelp() []key.Binding {
	return []key.Binding{filterKeys.Edit, filterKeys.Table, keys.New, keys.Quit}
}

func (k selectedFilterKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// editingFilterKeys are listed in the help footer while the filter is being
// edited.
type editingFilterKeys struct{}

func (editingFilterKeys) ShortHelp() []key.Binding {
	return []key.Binding{filterKeys.Apply, filterKeys.Cancel}
}

func (k editingFilterKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// selectFilter gives the filter field the focus.
func (m *Model) selectFilter() {
	m.filter.selected = true
	m.table.Blur()
	m.resize()
}

// selectTable gives the table the focus.
func (m *Model) selectTable() {
	m.filter.selected = false
	m.table.Focus()
	m.resize()
}

// editFilter selects the filter field and starts editing its text.
func (m *Model) editFilter() tea.Cmd {
	input := textarea.New()
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.DynamicHeight = true
	input.MinHeight = 1
	input.MaxHeight = maxFilterHeight
	// Allow a filter of any length. MaxHeight only limits how much is shown.
	input.MaxContentHeight = math.MaxInt
	// Enter applies the filter, so it doesn't start a new line.
	input.KeyMap.InsertNewline.SetEnabled(false)
	input.SetStyles(textarea.DefaultStyles(!m.light))
	input.SetValue(m.filter.text)
	input.MoveToEnd()

	m.filter.input = input
	m.filter.editing = true
	m.filter.err = nil
	// Selecting the filter sizes the input to fit the field.
	m.selectFilter()

	return m.filter.input.Focus()
}

// updateFilter handles a key press while the filter is being edited.
func (m *Model) updateFilter(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, filterKeys.Apply):
		m.applyFilter()
		return nil

	case key.Matches(msg, filterKeys.Cancel):
		m.filter.editing = false
		m.filter.err = nil
		m.resize()
		return nil
	}

	var cmd tea.Cmd
	m.filter.input, cmd = m.filter.input.Update(msg)

	// The field grows and shrinks with its text.
	m.resize()

	return cmd
}

// applyFilter parses the edited filter and, if it's valid, lists the issues
// it matches. An empty filter lists every issue. If it isn't valid, editing
// continues and the error is shown.
func (m *Model) applyFilter() {
	text := strings.TrimSpace(m.filter.input.Value())

	var node filter.Node

	if text != "" {
		var err error

		if node, err = filter.Parse(text); err != nil {
			m.filter.err = err
			m.resize()
			return
		}
	}

	m.filter.text = text
	m.filter.node = node
	m.filter.editing = false
	m.filter.err = nil
	m.reload()
}

// filterWidth returns the width inside the filter field's border and
// padding.
func (m Model) filterWidth() int {
	width := m.width - styles.Header.GetHorizontalFrameSize() - styles.FilterBox.GetHorizontalFrameSize()

	if m.width <= 0 {
		width = 40
	}

	return max(width, 1)
}

// renderFilter renders the applied filter shown in the field when it isn't
// being edited, with its syntax highlighted.
func renderFilter(node filter.Node) string {
	return node.Render()
}

// renderHeading renders the filter field and its label, which take the place
// of a title above the issue list.
func (m Model) renderHeading() string {
	width := m.filterWidth()

	var content string

	switch {
	case m.filter.editing:
		content = m.filter.input.View()
	case m.filter.node != nil:
		content = lipgloss.NewStyle().Width(width).Render(renderFilter(m.filter.node))
	default:
		content = lipgloss.NewStyle().Width(width).Render(styles.FilterPlaceholder.Render("No filter: every issue is listed"))
	}

	box := styles.FilterBox
	if m.filter.selected {
		box = styles.FilterBoxSelected
	}

	parts := []string{styles.FilterLabel.Render("Issue Filter"), box.Render(content)}

	if m.filter.err != nil {
		parts = append(parts, lipgloss.NewStyle().Width(width).Render(styles.Error.Render(m.filter.err.Error())))
	}

	return styles.Header.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}
