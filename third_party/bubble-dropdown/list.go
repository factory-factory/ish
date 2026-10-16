package dropdownv2

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const defaultMaxVisible = 8

// accentColor is a suggested accent color for WithAccentColor / SetAccentColor.
// It is NOT applied by default: an unstyled dropdown renders neutrally (plain
// border and a reverse-video highlight) until an accent or custom style is set.
const accentColor = "62"

// listStyles bundles the lipgloss styles used to render the open panel.
type listStyles struct {
	border lipgloss.Style
	normal lipgloss.Style
	cursor lipgloss.Style
}

// defaultListStyles returns the built-in panel styles. With an empty accent the
// panel is neutral: a default-colored rounded border, plain padded rows, and a
// reverse-video highlight. With an accent set, the border and highlight take
// the accent color.
func defaultListStyles(accent string) listStyles {
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	normal := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
	cursor := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)

	if accent == "" {
		// Neutral: show the highlight via reverse video so it stays visible
		// against the terminal's default palette.
		cursor = cursor.Reverse(true)
	} else {
		border = border.BorderForeground(lipgloss.Color(accent))
		cursor = cursor.
			Background(lipgloss.Color(accent)).
			Foreground(lipgloss.Color("255"))
	}

	return listStyles{border: border, normal: normal, cursor: cursor}
}

// listModel is the open dropdown panel. It is created fresh each time the
// dropdown opens; its View is composited by Dropdown.ViewWithOverlay.
//
// Mouse coordinates received by Update are expected to be relative to the
// panel's top-left cell (0,0 = top-left border rune). Dropdown.Update handles
// the translation from screen coords before forwarding here.
type listModel struct {
	items       []string
	cursor      int // absolute index of the highlighted item
	offset      int // index of the first visible item (scroll offset)
	maxVisible  int
	minContentW int // minimum content-area width; expands to fit longest item

	borderStyle lipgloss.Style
	normalStyle lipgloss.Style
	cursorStyle lipgloss.Style

	// separators holds the indices of the items with a rule drawn above
	// them. Local patch for ish; see WithSeparators.
	separators     map[int]bool
	separatorColor string
}

// setSeparators sets the items to draw a rule above, and the rules' color,
// or the border's if it's empty. A rule above the first item would separate
// nothing, so it's ignored. Local patch for ish.
func (m *listModel) setSeparators(before []int, color string) {
	m.separators = map[int]bool{}
	m.separatorColor = color

	for _, i := range before {
		if i > 0 && i < len(m.items) {
			m.separators[i] = true
		}
	}
}

// rows returns the panel's rows between its borders, in order: the index of
// each visible item, or -1 for a rule above one. Local patch for ish.
func (m listModel) rows() []int {
	end := min(m.offset+m.maxVisible, len(m.items))
	rows := make([]int, 0, end-m.offset)

	for i := m.offset; i < end; i++ {
		// A rule at the top of the panel, above a scrolled list, would
		// separate nothing visible.
		if m.separators[i] && i > m.offset {
			rows = append(rows, -1)
		}

		rows = append(rows, i)
	}

	return rows
}

func newListModel(items []string, initialCursor, maxVisible, minContentW int, styles listStyles) listModel {
	if maxVisible <= 0 {
		maxVisible = defaultMaxVisible
	}
	if initialCursor < 0 {
		initialCursor = 0
	}
	if len(items) > 0 && initialCursor >= len(items) {
		initialCursor = len(items) - 1
	}

	m := listModel{
		items:       items,
		cursor:      initialCursor,
		maxVisible:  maxVisible,
		minContentW: minContentW,
		borderStyle: styles.border,
		normalStyle: styles.normal,
		cursorStyle: styles.cursor,
	}
	m.scrollToCursor()
	return m
}

// setStyles replaces the panel's border, normal, and cursor styles in place.
// Dropdown.SetAccentColor uses this to recolor an already-open panel without
// rebuilding the listModel (which would reset cursor/scroll state).
func (m *listModel) setStyles(s listStyles) {
	if m == nil {
		return
	}
	m.borderStyle = s.border
	m.normalStyle = s.normal
	m.cursorStyle = s.cursor
}

// contentWidth returns the rendered width of each item row (excluding border cells).
func (m listModel) contentWidth() int {
	w := m.minContentW
	// +2 for the 1-cell padding on each side
	for _, item := range m.items {
		if iw := lipgloss.Width(item) + 2; iw > w {
			w = iw
		}
	}
	if w < 4 {
		w = 4
	}
	return w
}

// PanelSize returns the display-cell dimensions of View().
// Use this before opening so Dropdown can compute the overlay origin.
func (m listModel) PanelSize() (w, h int) {
	// +2 for top and bottom border. Local patch for ish: rows include rules.
	return m.contentWidth() + 2, len(m.rows()) + 2
}

// View renders the bordered item list.
func (m listModel) View() string {
	cw := m.contentWidth()

	normal := m.normalStyle.Width(cw)
	cursor := m.cursorStyle.Width(cw)

	// Local patch for ish: rules are drawn between items, in their color
	// or the border's.
	var ruleColor color.Color = m.borderStyle.GetBorderTopForeground()
	if m.separatorColor != "" {
		ruleColor = lipgloss.Color(m.separatorColor)
	}
	rule := lipgloss.NewStyle().Foreground(ruleColor).Render(strings.Repeat("─", cw))

	rows := make([]string, 0, m.maxVisible)
	for _, i := range m.rows() {
		if i < 0 {
			rows = append(rows, rule)
		} else if i == m.cursor {
			rows = append(rows, cursor.Render(m.items[i]))
		} else {
			rows = append(rows, normal.Render(m.items[i]))
		}
	}

	return m.borderStyle.Render(strings.Join(rows, "\n"))
}

func (m listModel) scrollToCursor() listModel {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.maxVisible {
		m.offset = m.cursor - m.maxVisible + 1
	}
	return m
}

func (m listModel) itemAtY(y int) (abs int, ok bool) {
	// Local patch for ish: rows include rules, which aren't items.
	row := y - 1 // 0-based row below the top border
	rows := m.rows()
	if row < 0 || row >= len(rows) || rows[row] < 0 {
		return 0, false
	}
	return rows[row], true
}

// Update handles key and mouse events for the open panel.
// Mouse coords must be relative to the panel's top-left (see Dropdown.Update).
func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case keyIsUp(msg):
			if m.cursor > 0 {
				m.cursor--
				m = m.scrollToCursor()
			}
			return m, nil
		case keyIsDown(msg):
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m = m.scrollToCursor()
			}
			return m, nil
		case keyIsEnterOrSpace(msg):
			if len(m.items) == 0 {
				return m, nil
			}
			idx := m.cursor
			val := m.items[idx]
			return m, func() tea.Msg { return ItemChosenMsg{Index: idx, Value: val} }
		case keyIsEscape(msg):
			return m, func() tea.Msg { return ItemCanceledMsg{} }
		}

	case tea.MouseMotionMsg:
		if abs, ok := m.itemAtY(msg.Y); ok {
			m.cursor = abs
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if abs, ok := m.itemAtY(msg.Y); ok {
				m.cursor = abs
				val := m.items[abs]
				return m, func() tea.Msg { return ItemChosenMsg{Index: abs, Value: val} }
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			if m.cursor > 0 {
				m.cursor--
				m = m.scrollToCursor()
			}
		case tea.MouseWheelDown:
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m = m.scrollToCursor()
			}
		}
	}
	return m, nil
}
