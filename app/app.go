// Package app provides the top-level ish terminal UI, which lists the issues
// in a directory and opens the selected issue.
package app

import (
	"maps"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
	"github.com/factory-factory/ish/state"
	"github.com/factory-factory/ish/store"
	"github.com/factory-factory/ish/styles"
	dropdown "github.com/madicen/bubble-dropdown/v2"
)

// Model lists issues until one is opened, then hands control to the issue
// view.
type Model struct {
	// dir is the directory issues are loaded from.
	dir    string
	issues []store.Summary

	table table.Model
	help  help.Model
	badge Badge

	// issue is the open issue's view, or nil while the list is shown, and
	// openId is its ID.
	issue  tea.Model
	openId string

	// single reports whether ish was started with a single issue, rather
	// than the list, so closing the issue quits. See Open.
	single bool

	// err is the error from the last attempt to open an issue, if any.
	err error

	// filter selects the issues that are listed, and unfiltered reports
	// whether there are any issues at all, to tell an empty project from a
	// filter that matches nothing.
	filter     filterField
	unfiltered bool

	// light reports whether the terminal has a light background.
	light bool

	// people maps the repository's users' emails to their names, which
	// are shown in place of the emails.
	people *models.Directory

	// state is the application state, such as the configuration, for
	// filters to read. See SetState.
	state *state.State

	// changes receives a value when issue files change outside ish, and
	// conflict is set while the open issue's unsaved edits conflict with
	// such a change. See Watch.
	changes  <-chan struct{}
	conflict *conflict

	width  int
	height int
}

// keyMap holds the key bindings for the issue list. It satisfies help.KeyMap
// so the bindings can be listed in the help footer.
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Open   key.Binding
	New    key.Binding
	Filter key.Binding
	Quit   key.Binding
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
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Filter: filterKeys.Start,
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}()

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.New, k.Filter, focusKeys.Next, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// New returns a model listing the issues in the metadata database in dir that
// match DefaultFilter.
func New(dir string) (Model, error) {
	node, err := filter.Parse(DefaultFilter)
	if err != nil {
		return Model{}, err
	}

	issues, err := store.ListIssues(dir, node)
	if err != nil {
		return Model{}, err
	}

	m := Model{
		dir:    dir,
		issues: issues,
		help:   help.New(),
		filter: filterField{text: DefaultFilter, node: node, menu: newFilterMenu()},
		people: loadPeople(dir),
	}

	if m.unfiltered, err = anyIssues(dir, issues); err != nil {
		return Model{}, err
	}

	// Columns must be set before rows, which are rendered as they're added.
	m.table = table.New(
		table.WithColumns(m.columns(0)),
		table.WithRows(rows(issues, m.people)),
		table.WithStyles(tableStyles(true)),
		table.WithFocused(true),
	)

	return m, nil
}

// Open returns a model showing the issue with the given ID in dir, without
// the list: closing the issue quits, and links to other issues open them in
// its place.
func Open(dir, id string) (Model, error) {
	m, err := New(dir)
	if err != nil {
		return Model{}, err
	}

	issue, err := parse.LoadIssue(dir, id)
	if err != nil {
		return Model{}, err
	}

	m.single = true

	// The issue's Init command is the same as the model's.
	shown, _ := m.show(*issue)

	return shown.(Model), nil
}

// tableStyles returns the issue table's styles. The selected issue is bold,
// bright white on blue while the table has the focus, and in normal text on
// grey while the filter has it instead.
func tableStyles(focused bool) table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		BorderBottom(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("4"))

	if !focused {
		s.Selected = lipgloss.NewStyle().Background(lipgloss.Color("8"))
	}

	return s
}

// SetState sets the application state, such as the configuration, that ish
// reads while it runs. An issue that's already open, as one opened with Open
// is, takes its author for new comments from it.
func (m *Model) SetState(st *state.State) {
	m.state = st

	if issue, ok := m.issue.(models.Issue); ok {
		issue.SetAuthor(m.author())
		m.issue = issue
	}
}

// State returns the application state, or one with an empty configuration
// if none was set.
func (m Model) State() *state.State {
	if m.state == nil {
		return state.New(nil)
	}

	return m.state
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.waitForChanges())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Track the terminal size while an issue is open, so the list fits the
	// terminal when the issue is closed.
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	}

	if _, ok := msg.(changedMsg); ok {
		return m.syncFiles()
	}

	// While the open issue's edits conflict with changes made outside ish,
	// keys answer the question of which to keep.
	if msg, ok := msg.(tea.KeyPressMsg); ok && m.conflict != nil {
		return m.updateConflict(msg)
	}

	if m.issue != nil {
		switch msg := msg.(type) {
		case models.CloseMsg:
			if m.single {
				return m, tea.Quit
			}

			m.issue = nil
			m.reload()
			m.openId = ""
			return m, nil

		case models.OpenMsg:
			return m.openLinked(msg.Id)
		}

		var cmd tea.Cmd
		m.issue, cmd = m.issue.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {

	case tea.BackgroundColorMsg:
		m.light = !msg.IsDark()
		m.help.Styles = help.DefaultStyles(msg.IsDark())

		if m.filter.editing {
			m.filter.input.SetStyles(textarea.DefaultStyles(msg.IsDark()))
		}

		return m, nil

	case tea.WindowSizeMsg:
		// The filter menu needs the size to place its panel.
		cmd := m.updateMenu(msg)
		return m, cmd

	case dropdown.ItemChosenMsg:
		menuCmd := m.updateMenu(msg)
		cmd := m.chooseFilter(msg.Index)
		return m, tea.Batch(menuCmd, cmd)

	case dropdown.ItemCanceledMsg:
		cmd := m.updateMenu(msg)
		return m, cmd

	case tea.KeyPressMsg:
		// While the filter is being edited, or its menu is open, keys other
		// than ctrl+c go to it.
		if m.filter.editing || m.filter.menu.Open() {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

			if m.filter.editing {
				cmd := m.updateFilter(msg)
				return m, cmd
			}

			cmd := m.updateMenu(msg)
			return m, cmd
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case focusDelta(msg) != 0:
			m.moveFocus(focusDelta(msg))
			return m, nil

		case key.Matches(msg, keys.New):
			return m.create()

		case key.Matches(msg, keys.Filter):
			cmd := m.editCustomFilter()
			return m, cmd
		}

		if m.filter.selected {
			switch {
			case key.Matches(msg, filterKeys.Table):
				m.selectTable()

			case m.filter.custom && key.Matches(msg, partKeys.Move):
				m.movePart(msg)

			case m.filter.custom && key.Matches(msg, filterKeys.Edit):
				cmd := m.pressPart()
				return m, cmd

			case !m.filter.custom && key.Matches(msg, menuKeys.Open):
				// The menu opens on enter or space. Other keys, such as up,
				// would change its selection without choosing it.
				cmd := m.updateMenu(msg)
				return m, cmd
			}

			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Open):
			return m.open()

		// Moving up from the top of the table selects the filter.
		case key.Matches(msg, keys.Up) && m.table.Cursor() <= 0:
			m.selectFilter()
			return m, nil
		}
	}

	// Other messages, such as the cursor blinking, go to whichever has the
	// focus.
	var cmd tea.Cmd

	if m.filter.editing {
		m.filter.input, cmd = m.filter.input.Update(msg)
		return m, cmd
	}

	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

// anyIssues reports whether the database in dir has any issues, given the
// issues a filter listed.
func anyIssues(dir string, listed []store.Summary) (bool, error) {
	if len(listed) > 0 {
		return true, nil
	}

	count, err := store.CountIssues(dir)

	return count > 0, err
}

// rows returns a table row for each issue.
func rows(issues []store.Summary, people *models.Directory) []table.Row {
	r := make([]table.Row, len(issues))
	for i, issue := range issues {
		r[i] = table.Row{issue.ID, issue.Title, assigneesText(issue, people), opened(issue, now()), stateText(issue)}
	}
	return r
}

// assigneesText formats an issue's assignees for the list, with users'
// names in place of their emails.
func assigneesText(issue store.Summary, people *models.Directory) string {
	shown := make([]string, len(issue.Assignees))

	for i, a := range issue.Assignees {
		shown[i] = people.Show(a)
	}

	return strings.Join(shown, ", ")
}

// loadPeople returns the users in the metadata database in dir, or none if
// they can't be read.
func loadPeople(dir string) *models.Directory {
	users, err := store.Users(dir)
	if err != nil {
		return models.NewDirectory(nil)
	}

	people := make([]models.User, len(users))
	for i, u := range users {
		people[i] = models.User{Name: u.Name, Email: u.Email}
	}

	return models.NewDirectory(people)
}

// stateText formats an issue's state for the list.
func stateText(issue store.Summary) string {
	return strings.Join(issue.State, ", ")
}

// opened formats the date an issue was opened for the list: the month and
// day if it's in the same year as now, and also the year otherwise. The date
// is shown in the time zone of the issue's byline, as in the issue view.
func opened(issue store.Summary, now time.Time) string {
	if issue.Opened.Year() == now.Year() {
		return issue.Opened.Format("Jan 2")
	}

	return issue.Opened.Format(models.DateTime)
}

// reload loads the issues from disk again, selecting the issue that was just
// open, or keeping the selected issue selected. If it's gone, the selection
// stays at the same position in the list. If the issues fail to load, the
// list is left as it was and the error is shown.
func (m *Model) reload() {
	// The filter's lookups are resolved from the state as it is now. If
	// they can't be, the error is shown under the filter, and the list is
	// left as it was.
	node, err := resolveFilter(m.filter.node, m.State(), m.dir)
	if err != nil {
		m.filter.err = err
		m.resize()
		return
	}

	issues, err := store.ListIssues(m.dir, node)

	if err == nil {
		m.unfiltered, err = anyIssues(m.dir, issues)
	}

	if err != nil {
		m.err = err
		m.resize()
		return
	}

	cursor := m.table.Cursor()
	selected := m.openId

	if row := m.table.SelectedRow(); selected == "" && row != nil {
		selected = row[0]
	}

	for i, issue := range issues {
		if issue.ID == selected {
			cursor = i
			break
		}
	}

	m.issues = issues
	m.err = nil

	// Columns must be set before rows, which are rendered as they're added.
	m.table.SetColumns(m.columns(0))
	m.people = loadPeople(m.dir)
	m.table.SetRows(rows(issues, m.people))
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

	return m.show(*issue)
}

// openLinked replaces the open issue with the issue with the given ID, which
// a link in it refers to. If the issue can't be loaded, the list is shown
// with the error.
func (m Model) openLinked(id string) (tea.Model, tea.Cmd) {
	issue, err := parse.LoadIssue(m.dir, id)

	if err != nil {
		m.issue = nil
		m.reload()
		m.openId = ""
		m.err = err
		m.resize()
		return m, nil
	}

	return m.show(*issue)
}

// create opens a new, empty issue with an unused ID, by the current user. It
// isn't written to disk unless it's edited.
func (m Model) create() (tea.Model, tea.Cmd) {
	id, err := parse.NewIssueID(m.dir)

	if err != nil {
		m.err = err
		m.resize()
		return m, nil
	}

	return m.show(models.NewIssue(id, m.author(), now()))
}

// show switches to the view of issue, which is saved to the list's directory
// when it's closed.
func (m Model) show(issue models.Issue) (tea.Model, tea.Cmd) {
	m.err = nil
	m.issue = m.prepare(issue)
	m.openId = issue.Id

	// The issue view hasn't seen the terminal size, which is only sent when
	// the program starts or the terminal is resized.
	var cmd tea.Cmd

	if m.width > 0 && m.height > 0 {
		m.issue, cmd = m.issue.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	}

	return m, tea.Batch(m.issue.Init(), cmd)
}

// prepare readies issue to be shown: it's saved to the list's directory when
// it's closed, and links to and suggests the other issues and the
// repository's users.
func (m *Model) prepare(issue models.Issue) models.Issue {
	issue.SetClosable(true)
	issue.SetSave(m.saveIssue)
	issue.SetAuthor(m.author())

	// Without the IDs of the other issues, references aren't linked, and
	// without their metadata and the repository's users, nothing is
	// suggested as tags are edited.
	var refs models.References

	if ids, err := store.IssueIDs(m.dir); err == nil {
		issue.SetIssueLookup(func(id string) bool { return ids[id] })

		refs.IssueIDs = slices.Sorted(maps.Keys(ids))
	}

	if values, err := store.MetadataValues(m.dir); err == nil {
		issue.SetMetadataValues(values)
	}

	if users, err := store.Users(m.dir); err == nil {
		refs.UserNames = userNames(users)
	}

	issue.SetReferences(refs)

	// Users' emails are shown, and their names typed, as their names.
	m.people = loadPeople(m.dir)
	issue.SetDirectory(m.people)

	return issue
}

// userNames returns the distinct names of users, in order. A user with
// several emails has one name.
func userNames(users []store.User) []string {
	var names []string

	for _, user := range users {
		if user.Name != "" && !slices.Contains(names, user.Name) {
			names = append(names, user.Name)
		}
	}

	return names
}

// saveIssue writes an issue's changes to its file in the list's directory,
// and updates the metadata database the list is read from.
func (m Model) saveIssue(issue models.Issue) error {
	return store.SaveIssue(m.dir, issue)
}

// resize fits the table into the space left between the heading and the
// footer, and sizes its columns to the terminal width.
func (m *Model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	m.help.SetWidth(m.width)

	if m.filter.editing {
		m.filter.input.SetWidth(m.filterWidth())
	}

	// The menu's panel opens under its trigger, which follows the badge
	// and the label.
	m.filter.menu.SetBounds(
		styles.Header.GetPaddingTop(),
		styles.Header.GetPaddingLeft()+m.filterIndent()+labelWidth(),
		m.filterBoxWidth(), lipgloss.Height(m.renderTrigger()),
	)

	chrome := lipgloss.Height(m.renderHeading()) + lipgloss.Height(m.renderFooter())

	m.table.SetColumns(m.columns(m.width - styles.Body.GetHorizontalFrameSize()))
	m.table.SetWidth(m.width - styles.Body.GetHorizontalFrameSize())
	m.table.SetHeight(max(m.height-chrome, 0))
}

// maxAssigneeWidth limits the width of the assignee column, so long lists of
// assignees don't crowd out titles. Wider values are truncated.
const maxAssigneeWidth = 30

// columns returns the table columns, giving the title whatever width the
// other columns leave over. If width is zero, the title column is as wide as
// the longest title.
func (m Model) columns(width int) []table.Column {
	id, title, assignee, date, state := len("ID"), len("Title"), len("Assignee"), len("Opened"), len("State")
	today := now()

	for _, issue := range m.issues {
		id = max(id, lipgloss.Width(issue.ID))
		title = max(title, lipgloss.Width(issue.Title))
		assignee = max(assignee, lipgloss.Width(assigneesText(issue, m.people)))
		date = max(date, lipgloss.Width(opened(issue, today)))
		state = max(state, lipgloss.Width(stateText(issue)))
	}

	assignee = min(assignee, maxAssigneeWidth)

	if width > 0 {
		// Each cell is padded by one column on either side.
		padding := 5 * table.DefaultStyles().Cell.GetHorizontalPadding()
		title = max(width-id-assignee-date-state-padding, len("Title"))
	}

	return []table.Column{
		{Title: "ID", Width: id},
		{Title: "Title", Width: title},
		{Title: "Assignee", Width: assignee},
		{Title: "Opened", Width: date},
		{Title: "State", Width: state},
	}
}

func (m Model) renderFooter() string {
	var keyMap help.KeyMap = keys

	switch {
	case m.filter.editing:
		keyMap = editingFilterKeys{}
	case m.filter.menu.Open():
		keyMap = openMenuKeys{}
	case m.filter.selected && m.filter.custom:
		keyMap = selectedFilterKeys{part: m.filter.part}
	case m.filter.selected:
		keyMap = selectedMenuKeys{}
	}

	footer := styles.Footer.Render(m.help.View(keyMap))

	if m.err != nil {
		// Long errors wrap to the terminal's width.
		message := styles.Footer
		if m.width > 0 {
			message = message.Width(m.width)
		}

		footer = lipgloss.JoinVertical(lipgloss.Left,
			message.Render(styles.Error.Render(m.err.Error())),
			footer,
		)
	}

	return footer
}

func (m Model) View() tea.View {
	if m.issue != nil {
		v := m.issue.View()

		if m.conflict != nil {
			v.Content = m.overlayConflict(v.Content)
		}

		return v
	}

	body := m.table.View()

	if len(m.issues) == 0 {
		body = styles.Byline.Render("No issues in " + m.dir)

		if m.unfiltered {
			body = styles.Byline.Render("No issues match the filter")
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeading(),
		styles.Body.Render(body),
		m.renderFooter(),
	)

	// The open filter menu's panel is drawn over the list.
	content = m.filter.menu.ViewWithOverlay(content, m.width, m.height)

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
