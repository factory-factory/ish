// Package app provides the top-level ish terminal UI, which lists the issues
// in a directory and opens the selected issue.
package app

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
	"github.com/factory-factory/ish/styles"
)

// Model lists issues until one is opened, then hands control to the issue
// view.
type Model struct {
	// dir is the directory issues are loaded from.
	dir    string
	issues []models.Issue

	table table.Model
	help  help.Model

	// issue is the open issue's view, or nil while the list is shown.
	issue tea.Model

	// err is the error from the last attempt to open an issue, if any.
	err error

	width  int
	height int
}

// keyMap holds the key bindings for the issue list. It satisfies help.KeyMap
// so the bindings can be listed in the help footer.
type keyMap struct {
	Up   key.Binding
	Down key.Binding
	Open key.Binding
	Quit key.Binding
}

var keys = func() keyMap {
	nav := table.DefaultKeyMap()

	return keyMap{
		Up:   nav.LineUp,
		Down: nav.LineDown,
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}()

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// New returns a model listing the given issues, which were loaded from dir.
func New(dir string, issues []models.Issue) Model {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		BorderBottom(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("4"))

	m := Model{
		dir:    dir,
		issues: issues,
		help:   help.New(),
	}

	// Columns must be set before rows, which are rendered as they're added.
	m.table = table.New(
		table.WithColumns(m.columns(0)),
		table.WithRows(rows(issues)),
		table.WithStyles(s),
		table.WithFocused(true),
	)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Track the terminal size while an issue is open, so the list fits the
	// terminal when the issue is closed.
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	}

	if m.issue != nil {
		if _, ok := msg.(models.CloseMsg); ok {
			m.issue = nil
			m.reload()
			return m, nil
		}

		var cmd tea.Cmd
		m.issue, cmd = m.issue.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {

	case tea.BackgroundColorMsg:
		m.help.Styles = help.DefaultStyles(msg.IsDark())
		return m, nil

	case tea.WindowSizeMsg:
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Open):
			return m.open()
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

// rows returns a table row for each issue.
func rows(issues []models.Issue) []table.Row {
	r := make([]table.Row, len(issues))
	for i, issue := range issues {
		r[i] = table.Row{issue.Id, issue.Title, issue.State()}
	}
	return r
}

// reload loads the issues from disk again, keeping the selected issue
// selected. If it's gone, the selection stays at the same position in the
// list. If the issues fail to load, the list is left as it was and the error
// is shown.
func (m *Model) reload() {
	issues, err := parse.ListIssues(m.dir)

	if err != nil {
		m.err = err
		m.resize()
		return
	}

	cursor := m.table.Cursor()

	if row := m.table.SelectedRow(); row != nil {
		for i, issue := range issues {
			if issue.Id == row[0] {
				cursor = i
				break
			}
		}
	}

	m.issues = issues
	m.err = nil

	// Columns must be set before rows, which are rendered as they're added.
	m.table.SetColumns(m.columns(0))
	m.table.SetRows(rows(issues))
	m.table.SetCursor(min(cursor, len(issues)-1))

	m.resize()
}

// open loads the selected issue from disk and switches to its view.
func (m Model) open() (tea.Model, tea.Cmd) {
	row := m.table.SelectedRow()

	if row == nil {
		return m, nil
	}

	issue, err := parse.LoadIssue(m.dir, row[0])

	if err != nil {
		m.err = err
		m.resize()
		return m, nil
	}

	m.err = nil
	issue.SetClosable(true)
	issue.SetSave(m.saveIssue)
	m.issue = *issue

	// The issue view hasn't seen the terminal size, which is only sent when
	// the program starts or the terminal is resized.
	if m.width > 0 && m.height > 0 {
		m.issue, _ = m.issue.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	}

	return m, m.issue.Init()
}

// saveIssue writes an issue's changes to its file in the list's directory.
func (m Model) saveIssue(issue models.Issue) error {
	return parse.SaveIssue(m.dir, issue)
}

// resize fits the table into the space left between the heading and the
// footer, and sizes its columns to the terminal width.
func (m *Model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	m.help.SetWidth(m.width)

	chrome := lipgloss.Height(m.renderHeading()) + lipgloss.Height(m.renderFooter())

	m.table.SetColumns(m.columns(m.width - styles.Body.GetHorizontalFrameSize()))
	m.table.SetWidth(m.width - styles.Body.GetHorizontalFrameSize())
	m.table.SetHeight(max(m.height-chrome, 0))
}

// columns returns the table columns, giving the title whatever width the ID
// and state columns leave over. If width is zero, the title column is as wide
// as the longest title.
func (m Model) columns(width int) []table.Column {
	id, title, state := len("ID"), len("Title"), len("State")

	for _, issue := range m.issues {
		id = max(id, lipgloss.Width(issue.Id))
		title = max(title, lipgloss.Width(issue.Title))
		state = max(state, lipgloss.Width(issue.State()))
	}

	if width > 0 {
		// Each cell is padded by one column on either side.
		padding := 3 * table.DefaultStyles().Cell.GetHorizontalPadding()
		title = max(width-id-state-padding, len("Title"))
	}

	return []table.Column{
		{Title: "ID", Width: id},
		{Title: "Title", Width: title},
		{Title: "State", Width: state},
	}
}

func (m Model) renderHeading() string {
	return styles.Header.Render(styles.Title.Render("Issues"))
}

func (m Model) renderFooter() string {
	footer := styles.Footer.Render(m.help.View(keys))

	if m.err != nil {
		footer = lipgloss.JoinVertical(lipgloss.Left,
			styles.Footer.Render(styles.Error.Render(m.err.Error())),
			footer,
		)
	}

	return footer
}

func (m Model) View() tea.View {
	if m.issue != nil {
		return m.issue.View()
	}

	body := m.table.View()

	if len(m.issues) == 0 {
		body = styles.Byline.Render("No issues in " + m.dir)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeading(),
		styles.Body.Render(body),
		m.renderFooter(),
	)

	if m.width > 0 && m.height > 0 {
		content = lipgloss.NewStyle().
			MaxWidth(m.width).
			MaxHeight(m.height).
			Render(content)
	}

	v := tea.NewView(content)
	v.AltScreen = true

	return v
}
