package models

import (
	"log"
	"math"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	glamstyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
	"github.com/muesli/termenv"
)

type Card struct {
	Id          string
	DateCreated string
	Author      string
	Content     *string

	selected bool

	// editing reports whether the content is being edited as markdown in
	// editor.
	editing bool
	editor  textarea.Model

	// selectingTodo reports whether the card's todo items are being
	// selected, and todo is the index of the selected item.
	selectingTodo bool
	todo          int
}

type SelectMsg bool

func (c Card) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (c Card) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SelectMsg:
		c.selected = bool(msg)

		if !c.selected {
			c.stopTodos()
		}

		return c, nil
	}

	if c.editing {
		var cmd tea.Cmd
		c.editor, cmd = c.editor.Update(msg)
		return c, cmd
	}

	return c, nil
}

// maxEditorWidth matches the width glamour wraps rendered markdown to.
const maxEditorWidth = 80

// startEdit begins editing the card's content as markdown in a text area,
// sized so the card fits within width and height.
func (c *Card) startEdit(width, height int, isDark bool) tea.Cmd {
	c.editor = textarea.New()
	c.editor.Prompt = ""
	c.editor.ShowLineNumbers = false
	c.editor.DynamicHeight = true
	// Allow any amount of content. MaxHeight only limits how much is shown.
	c.editor.MaxContentHeight = math.MaxInt
	c.editor.SetStyles(textarea.DefaultStyles(isDark))

	if c.Content != nil {
		c.editor.SetValue(*c.Content)
	}

	c.sizeEditor(width, height)
	c.editor.MoveToBegin()
	c.editing = true

	return c.editor.Focus()
}

// sizeEditor fits the card, with its byline and border, within width and
// height.
func (c *Card) sizeEditor(width, height int) {
	frame := c.getStyle().GetHorizontalFrameSize()
	chrome := lipgloss.Height(c.renderByline()) + c.getStyle().GetVerticalFrameSize()

	c.editor.MaxHeight = max(height-chrome, 1)
	c.editor.SetWidth(min(max(width-frame, 1), maxEditorWidth))
}

// commitEdit replaces the card's content with the edited markdown,
// reporting whether it changed.
func (c *Card) commitEdit() bool {
	content := c.editor.Value()
	c.editing = false

	if c.Content != nil && *c.Content == content || c.Content == nil && content == "" {
		return false
	}

	c.Content = &content
	return true
}

// cancelEdit stops editing, leaving the content unchanged.
func (c *Card) cancelEdit() {
	c.editing = false
}

// renderByline renders the comment's author and date.
func (c Card) renderByline() string {
	t, err := ParseDate(c.DateCreated)

	if err != nil {
		log.Fatal(err)
	}

	return styles.BylineBlock.Render(
		styles.BylineEmphasis.Render(c.Author) +
			styles.Byline.Render(" commented on "+t.Format(DateTime)),
	)
}

func (c Card) View() tea.View {
	byline := c.renderByline()

	if c.editing {
		editor := c.getStyle().Render(c.editor.View())
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, byline, editor))
	}

	content, _ := c.renderContent()
	comment := c.getStyle().Render(content)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, byline, comment))
}

// renderContent renders the card's markdown, highlighting the selected todo
// item if todo items are being selected. It returns the rendered content and
// the index of the highlighted line, or -1 if there isn't one.
func (c Card) renderContent() (string, int) {
	source := ""
	if c.Content != nil {
		source = *c.Content
	}

	if todos := c.todos(); c.selectingTodo && c.todo < len(todos) {
		source = markTodo(source, todos[c.todo])
	}

	var style ansi.StyleConfig

	if termenv.HasDarkBackground() {
		style = glamstyles.DarkStyleConfig
	} else {
		style = glamstyles.LightStyleConfig
	}

	margin := uint(0)
	style.Document.Margin = &margin
	style.Document.BlockPrefix = ""
	style.Document.BlockSuffix = ""

	var content string

	r, err := glamour.NewTermRenderer(glamour.WithStyles(style))

	if err == nil {
		content, err = r.Render(source)
	}

	if err != nil {
		content = source
	}

	content = strings.TrimRight(content, "\n\t ")

	if utf8.RuneCountInString(content) == 0 {
		content = strings.Repeat(" ", 80)
	}

	return highlightTodo(content)
}

// todoRow returns the line of the card's view showing the selected todo item,
// or -1 if no todo item is selected.
func (c Card) todoRow() int {
	_, row := c.renderContent()

	if row < 0 {
		return -1
	}

	return lipgloss.Height(c.renderByline()) + c.getStyle().GetBorderTopSize() + row
}

func (c Card) getStyle() lipgloss.Style {
	switch c.selected {
	case true:
		return styles.CardSelected
	default:
		return styles.Card
	}
}
