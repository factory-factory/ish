package models

import (
	"fmt"
	"log"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

type Issue struct {
	Id          string
	DateCreated string
	Author      string
	Title       string

	Metadata []Tag

	Conversation []Card

	selectedComment int
	cursor          int

	// width and height are the terminal dimensions.
	width  int
	height int

	// viewport scrolls the cards below the pinned header, and spans records
	// where each card sits in the viewport content.
	viewport viewport.Model
	spans    []span

	help  help.Model
	ready bool

	// closable reports whether the back key closes the issue. See
	// SetClosable.
	closable bool
}

// CloseMsg is sent when the user closes an issue with the back key.
type CloseMsg struct{}

// SetClosable sets whether the back key closes the issue by sending a
// CloseMsg, for when the issue is shown by a model that can return to a
// previous view.
func (ish *Issue) SetClosable(closable bool) {
	ish.closable = closable
}

// keyMap returns the issue's key bindings, with the back key enabled if the
// issue can be closed.
func (ish Issue) keyMap() keyMap {
	k := keys
	k.Back.SetEnabled(ish.closable)
	return k
}

const (
	ISO8601A  = "2006-01-02 15:04:05Z0700"
	ISO8601B  = "2006-01-02T15:04:05Z0700"
	Timestamp = "2006-01-02 15:04:05 -0700" // Local time :sweat-smile:
	DateTime  = "Jan 2, 2006"               // 15:04:05 MST"
)

// type Cursor struct {
// }

// func (c Cursor) Next() {
// }

func InitialModel() Issue {
	return Issue{
		//
		Id:          "00000",
		DateCreated: "2023-05-26T23:35:36-0400",
		Author:      "brendanberg",
		Title:       "What is ish?",

		Metadata: []Tag{
			{
				Key:    "state",
				Values: []string{"in progress"},
			},
			{
				Key:    "assignees",
				Values: []string{"brendanberg", "jhc"},
			},
		},
		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		//conversation: make([]card, 2),
		Conversation: []Card{
			{
				Id:          "001",
				DateCreated: "2023-05-27T13:41:50-0400",
				Author:      "brendanberg",
				Content:     nil,
			},
			{
				Id:          "002",
				DateCreated: "2023-05-27T23:37:18-0400",
				Author:      "brendanberg",
				Content:     nil,
			},
		},

		selectedComment: 0,
		cursor:          0,
	}
}

func (ish Issue) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (ish Issue) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ish.initComponents()

	switch msg := msg.(type) {

	case tea.BackgroundColorMsg:
		ish.help.Styles = help.DefaultStyles(msg.IsDark())
		return ish, nil

	case tea.WindowSizeMsg:
		ish.width = msg.Width
		ish.height = msg.Height
		ish.resize()

		if i := ish.cardIndex(); i >= 0 {
			ish.reveal(ish.spans[i], true)
		}

		return ish, nil

	case tea.KeyPressMsg:
		keys := ish.keyMap()

		switch {
		case key.Matches(msg, keys.Quit):
			return ish, tea.Quit

		case key.Matches(msg, keys.Back):
			return ish, func() tea.Msg { return CloseMsg{} }

		case key.Matches(msg, keys.Help):
			ish.help.ShowAll = !ish.help.ShowAll
			ish.resize()
			return ish, nil

		case key.Matches(msg, keys.Next, keys.Prev):
			total := len(ish.Metadata) + len(ish.Conversation) + 2
			next := key.Matches(msg, keys.Next)
			c := ish.cursor

			if next {
				c = (c + 1) % total
			} else {
				c = (c - 1 + total) % total
			}

			ish.setCursor(c)

			if i := ish.cardIndex(); i >= 0 {
				ish.reveal(ish.spans[i], next)
			}

			return ish, nil

		case key.Matches(msg, keys.Down):
			ish.scrollDown()
			return ish, nil

		case key.Matches(msg, keys.Up):
			ish.scrollUp()
			return ish, nil
		}
	}

	// Paging keys and the mouse wheel scroll the viewport directly.
	var cmd tea.Cmd
	ish.viewport, cmd = ish.viewport.Update(msg)

	return ish, cmd
}

// initComponents sets up the viewport and help footer the first time the
// issue receives a message, since issues are built as struct literals.
func (ish *Issue) initComponents() {
	if ish.ready {
		return
	}

	ish.viewport = viewport.New()
	// Arrow keys move between cards, so the viewport mustn't handle them too.
	ish.viewport.KeyMap.Up.Unbind()
	ish.viewport.KeyMap.Down.Unbind()

	ish.help = help.New()
	ish.ready = true

	ish.syncViewport()
}

// resize fits the viewport into the space left between the pinned header and
// the help footer.
func (ish *Issue) resize() {
	ish.help.SetWidth(ish.width)

	header := lipgloss.Height(ish.renderHeader())
	footer := lipgloss.Height(ish.renderFooter())

	ish.viewport.SetWidth(ish.width)
	ish.viewport.SetHeight(max(ish.height-header-footer, 0))

	ish.syncViewport()
}

// syncViewport re-renders the cards into the viewport, recording where each
// card sits so the selection can be scrolled into view.
func (ish *Issue) syncViewport() {
	content, spans := ish.renderCards()

	ish.spans = spans
	ish.viewport.SetContent(content)
	// Content may have shrunk, so clamp the offset to the new bounds.
	ish.viewport.SetYOffset(ish.viewport.YOffset())
}

// span is the vertical extent, in lines, of a rendered card within the
// viewport content.
type span struct {
	top    int
	height int
}

func (s span) bottom() int {
	return s.top + s.height
}

// firstCard is the cursor position of the first card in the conversation.
// Positions before it belong to the title, the byline, and the metadata tags.
func (ish Issue) firstCard() int {
	return len(ish.Metadata) + 2
}

// cardIndex returns the index of the selected card, or -1 if the cursor is
// not on a card.
func (ish Issue) cardIndex() int {
	i := ish.cursor - ish.firstCard()
	if i < 0 || i >= len(ish.Conversation) {
		return -1
	}
	return i
}

// setCursor moves the cursor to c, updating the selection state of the
// metadata tag or card under the old and new positions.
func (ish *Issue) setCursor(c int) {
	ish.selectAt(ish.cursor, false)
	ish.cursor = c
	ish.selectAt(ish.cursor, true)

	ish.syncViewport()
}

func (ish *Issue) selectAt(c int, selected bool) {
	first := ish.firstCard()

	switch {
	case c >= 2 && c < first:
		meta, _ := ish.Metadata[c-2].Update(SelectMsg(selected))
		ish.Metadata[c-2] = meta.(Tag)
	case c >= first && c < first+len(ish.Conversation):
		comment, _ := ish.Conversation[c-first].Update(SelectMsg(selected))
		ish.Conversation[c-first] = comment.(Card)
	}
}

// scrollDown scrolls the rest of the selected card into view one line at a
// time, then moves the selection to the next card once the selected card's
// last line is visible.
func (ish *Issue) scrollDown() {
	i := ish.cardIndex()
	bottom := ish.viewport.YOffset() + ish.viewport.Height()

	switch {
	case len(ish.spans) == 0:
		return
	case i < 0:
		ish.setCursor(ish.firstCard())
		ish.reveal(ish.spans[0], true)
	case ish.spans[i].bottom() > bottom:
		ish.viewport.ScrollDown(1)
	case i+1 < len(ish.spans):
		ish.setCursor(ish.cursor + 1)
		ish.reveal(ish.spans[i+1], true)
	}
}

// scrollUp scrolls the rest of the selected card into view one line at a
// time, then moves the selection to the previous card once the selected
// card's first line is visible.
func (ish *Issue) scrollUp() {
	i := ish.cardIndex()
	last := len(ish.spans) - 1

	switch {
	case len(ish.spans) == 0:
		return
	case i < 0:
		ish.setCursor(ish.firstCard() + last)
		ish.reveal(ish.spans[last], false)
	case ish.spans[i].top < ish.viewport.YOffset():
		ish.viewport.ScrollUp(1)
	case i > 0:
		ish.setCursor(ish.cursor - 1)
		ish.reveal(ish.spans[i-1], false)
	}
}

// reveal scrolls the viewport so that s is visible. A card taller than the
// viewport is revealed from its top edge when moving down and from its bottom
// edge when moving up.
func (ish *Issue) reveal(s span, down bool) {
	top := ish.viewport.YOffset()
	area := ish.viewport.Height()

	if s.height > area {
		if down && (s.top < top || s.top >= top+area) {
			ish.viewport.SetYOffset(s.top)
		} else if !down && (s.bottom() <= top || s.bottom() > top+area) {
			ish.viewport.SetYOffset(s.bottom() - area)
		}
		return
	}

	if s.top < top {
		ish.viewport.SetYOffset(s.top)
	} else if s.bottom() > top+area {
		ish.viewport.SetYOffset(s.bottom() - area)
	}
}

// renderHeader renders the title, byline, metadata tags, and the rule beneath
// them, all of which stay pinned to the top of the screen.
func (ish Issue) renderHeader() string {
	t, err := time.Parse(time.RFC3339, ish.DateCreated)

	if err != nil {
		t, err = time.Parse(ISO8601A, ish.DateCreated)
	}

	if err != nil {
		t, err = time.Parse(ISO8601B, ish.DateCreated)
	}

	if err != nil {
		t, err = time.Parse(Timestamp, ish.DateCreated)
	}

	if err != nil {
		log.Fatal(err)
	}

	c := make([]string, 0)

	c = append(c, ish.getTitleStyle().Render(fmt.Sprintf("[%s] %s", ish.Id, ish.Title)))
	c = append(c, styles.BylineBlock.Copy().MarginBottom(1).Render(
		styles.BylineEmphasis.Render(ish.Author)+
			styles.Byline.Render(" opened this issue on "+t.Format(DateTime)),
	))

	for _, tag := range ish.Metadata {
		c = append(c, tag.View().Content)
	}

	header := styles.Header.Render(lipgloss.JoinVertical(lipgloss.Left, c...))

	// Separate the header from the cards with a rule spanning the screen, or
	// the header itself before the terminal size is known.
	width := ish.width
	if width <= 0 {
		width = lipgloss.Width(header)
	}

	rule := styles.Rule.Render(
		strings.Repeat("─", max(width-styles.Rule.GetHorizontalMargins(), 0)),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, rule)
}

// renderCards renders every card in the conversation, along with the position
// of each card in the result.
func (ish Issue) renderCards() (string, []span) {
	c := make([]string, 0, len(ish.Conversation))
	spans := make([]span, 0, len(ish.Conversation))
	top := 0

	for _, card := range ish.Conversation {
		s := card.View().Content
		h := lipgloss.Height(s)

		c = append(c, s)
		spans = append(spans, span{top: top, height: h})
		top += h
	}

	return styles.Body.Render(lipgloss.JoinVertical(lipgloss.Left, c...)), spans
}

// renderFooter renders the key binding help shown below the viewport.
func (ish Issue) renderFooter() string {
	return styles.Footer.Render(ish.help.View(ish.keyMap()))
}

func (ish Issue) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left,
		ish.renderHeader(),
		ish.viewport.View(),
		ish.renderFooter(),
	)

	if ish.width > 0 && ish.height > 0 {
		content = lipgloss.NewStyle().
			MaxWidth(ish.width).
			MaxHeight(ish.height).
			Render(content)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
}

// State returns the values of the issue's state tag, or an empty string if
// it has none.
func (ish Issue) State() string {
	for _, tag := range ish.Metadata {
		if tag.Key == "state" {
			return strings.Join(tag.Values, ", ")
		}
	}

	return ""
}

func (ish Issue) Print() {
	fmt.Printf("Title: %s\n", ish.Title)
	fmt.Printf("Author: %s\n", ish.Author)
	fmt.Printf("Date: %s\n", ish.DateCreated)

	if len(ish.Metadata) > 0 {
		fmt.Print("Metadata:\n")
		for _, elt := range ish.Metadata {
			fmt.Printf("  Key('%s'):", elt.Key)
			for _, val := range elt.Values {
				fmt.Printf(" Value('%s')", val)
			}
			fmt.Print("\n")
		}
	}

	for _, card := range ish.Conversation {
		fmt.Print("-----\n")
		fmt.Printf("Author: %s\n", card.Author)
		fmt.Printf("Date: %s\n", card.DateCreated)
		fmt.Print("Content:\n  ")
		fmt.Print(strings.ReplaceAll(*card.Content, "\n", "\n  "))
	}

}

func (ish Issue) getTitleStyle() lipgloss.Style {
	switch ish.cursor {
	case 1:
		return styles.TitleSelected
	default:
		return styles.Title
	}
}
