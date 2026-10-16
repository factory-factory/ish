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
	dropdown "github.com/madicen/bubble-dropdown/v2"
)

// DefaultFilter is the filter the issue list starts with, which lists every
// issue that isn't closed.
const DefaultFilter = "{state != 'closed'}"

// filterField is the row above the issue list that selects which issues are
// listed: a menu of filters, or a field for a custom filter expression.
type filterField struct {
	// text is the applied filter as it was typed, and node is its parsed
	// form. An empty filter has a nil node, and lists every issue.
	text string
	node filter.Node

	// menu offers the preset filters, and preset is the last one chosen from
	// it. When custom is set, the custom filter field replaces the menu.
	menu   *dropdown.Dropdown
	preset int
	custom bool

	// selected reports whether the filter row has the focus instead of the
	// table, and editing whether the custom filter is being edited in input.
	selected bool
	editing  bool
	input    textarea.Model

	// err is the error from the last attempt to apply an edited filter.
	err error
}

// filterLabel is shown before the filter menu or custom filter field.
const filterLabel = "Filter:"

// The filter menu's trigger and the custom filter field are boxes as wide as
// the space after the label, within these limits, including their borders.
const (
	minFilterBoxWidth = 60
	maxFilterBoxWidth = 90
)

// filterArrow marks the filter menu's trigger as a menu.
const filterArrow = "▼"

// filterIndent returns the width of the heading before the filter label: the
// badge, and the space after it, less the label's own padding.
func (m Model) filterIndent() int {
	return m.badge.width() + badgeGap - styles.FilterLabel.GetPaddingLeft()
}

// labelWidth returns the width of the rendered filter label.
func labelWidth() int {
	return lipgloss.Width(styles.FilterLabel.Render(filterLabel))
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

// selectedFilterKeys are listed in the help footer while the custom filter
// field is selected.
type selectedFilterKeys struct{}

func (selectedFilterKeys) ShortHelp() []key.Binding {
	return []key.Binding{filterKeys.Edit, menuKeys.Menu, filterKeys.Table, keys.New, keys.Quit}
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

// selectFilter gives the filter row the focus: the menu, or the custom filter
// field if it's replaced the menu.
func (m *Model) selectFilter() {
	m.filter.selected = true
	m.filter.menu.SetFocused(!m.filter.custom)
	m.table.Blur()
	m.resize()
}

// selectTable gives the table the focus.
func (m *Model) selectTable() {
	m.filter.selected = false
	m.filter.menu.SetFocused(false)
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

// applyFilterText applies a valid filter expression, such as a preset's.
func (m *Model) applyFilterText(text string) {
	node, err := filter.Parse(text)

	if err != nil {
		m.filter.err = err
		m.resize()
		return
	}

	m.filter.text = text
	m.filter.node = node
	m.filter.err = nil
	m.reload()
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

// filterWidth returns the width inside the custom filter field's border and
// padding.
func (m Model) filterWidth() int {
	return m.filterBoxWidth() - styles.FilterBox.GetHorizontalFrameSize()
}

// filterBoxWidth returns the width of the filter menu's trigger and the
// custom filter field, including their borders.
func (m Model) filterBoxWidth() int {
	available := m.width - styles.Header.GetHorizontalFrameSize() - m.filterIndent() - labelWidth()

	return min(max(available, minFilterBoxWidth), maxFilterBoxWidth)
}

// renderTrigger renders the filter menu's trigger: a box with the chosen
// filter's name, and an arrow at its right edge. It's grey, and bright white
// with a blue border while it's selected, whether or not the menu is open.
func (m Model) renderTrigger() string {
	style := styles.FilterTrigger
	if m.filter.selected {
		style = styles.FilterTriggerSelected
	}

	name := m.filter.menu.Selected()
	space := m.filterBoxWidth() - style.GetHorizontalFrameSize() - lipgloss.Width(name) - lipgloss.Width(filterArrow)

	return style.Render(name + strings.Repeat(" ", max(space, 1)) + filterArrow)
}

// renderFilter renders the applied filter shown in the field when it isn't
// being edited, with its syntax highlighted.
func renderFilter(node filter.Node) string {
	return node.Render()
}

// renderHeading renders the filter row, which takes the place of a title
// above the issue list: a label, then the filter menu, or the custom filter
// field if it's replaced the menu.
func (m Model) renderHeading() string {
	label := styles.FilterLabel.Render(filterLabel)
	box := m.renderTrigger()

	if m.filter.custom {
		box = m.renderCustomFilter()
	}

	// The label lines up with the first line of text inside the box.
	row := lipgloss.JoinHorizontal(lipgloss.Top, "\n"+label, box)

	gap := strings.Repeat(" ", m.filterIndent()-m.badge.width())
	parts := []string{lipgloss.JoinHorizontal(lipgloss.Top, m.badge.View().Content, gap, row)}

	if m.filter.err != nil {
		// The error lines up with the filter, after the badge.
		parts = append(parts, lipgloss.NewStyle().
			Width(m.filterWidth()+m.filterIndent()).
			PaddingLeft(m.filterIndent()).
			Render(styles.Error.Render(m.filter.err.Error())))
	}

	return styles.Header.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

// renderCustomFilter renders the custom filter field: the raw filter while
// it's being edited, and the applied filter, highlighted, otherwise.
func (m Model) renderCustomFilter() string {
	width := m.filterWidth()

	var content string

	switch {
	case m.filter.editing:
		content = m.filter.input.View()
	case m.filter.node != nil:
		content = fitLine(renderFilter(m.filter.node), width)
	default:
		content = fitLine(styles.FilterPlaceholder.Render("No filter: every issue is listed"), width)
	}

	box := styles.FilterBox
	if m.filter.selected {
		box = styles.FilterBoxSelected
	}

	return box.Render(content)
}

// fitLine fits styled text to a single line of the given width, padding it,
// or cutting it short with an ellipsis, so the custom filter field is as tall
// as the filter menu's trigger while it isn't being edited.
func fitLine(text string, width int) string {
	if lipgloss.Width(text) > width {
		text = lipgloss.NewStyle().MaxWidth(width-1).Render(text) + "…"
	}

	return lipgloss.NewStyle().Width(width).Render(text)
}
