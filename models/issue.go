package models

import (
	"fmt"
	"log"
	"strings"
	"time"

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

	// width and height are the terminal dimensions, and offset is the number
	// of lines the card area is scrolled down.
	width  int
	height int
	offset int
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
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (ish Issue) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		ish.width = msg.Width
		ish.height = msg.Height

		spans, area := ish.layout()
		if i := ish.cardIndex(); i >= 0 {
			ish.reveal(spans[i], area, true)
		}
		ish.clampOffset(spans, area)

		return ish, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return ish, tea.Quit

		case "tab", "shift+tab":
			total := len(ish.Metadata) + len(ish.Conversation) + 2
			c := ish.cursor

			switch msg.String() {
			case "tab":
				c = (c + 1) % total
			case "shift+tab":
				c = (c - 1 + total) % total
			}

			ish.setCursor(c)

			spans, area := ish.layout()
			if i := ish.cardIndex(); i >= 0 {
				ish.reveal(spans[i], area, msg.String() == "tab")
			}
			ish.clampOffset(spans, area)

			return ish, nil

		case "down", "j":
			ish.scrollDown()
			return ish, nil

		case "up", "k":
			ish.scrollUp()
			return ish, nil
		}
	}

	return ish, nil
}

// span is the vertical extent, in lines, of a rendered card within the
// scrollable card area.
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
	spans, area := ish.layout()
	i := ish.cardIndex()

	switch {
	case len(spans) == 0:
		return
	case i < 0:
		ish.setCursor(ish.firstCard())
		ish.reveal(spans[0], area, true)
	case spans[i].bottom() > ish.offset+area:
		ish.offset++
	case i+1 < len(spans):
		ish.setCursor(ish.cursor + 1)
		ish.reveal(spans[i+1], area, true)
	}

	ish.clampOffset(spans, area)
}

// scrollUp scrolls the rest of the selected card into view one line at a
// time, then moves the selection to the previous card once the selected
// card's first line is visible.
func (ish *Issue) scrollUp() {
	spans, area := ish.layout()
	i := ish.cardIndex()

	switch {
	case len(spans) == 0:
		return
	case i < 0:
		ish.setCursor(ish.firstCard() + len(spans) - 1)
		ish.reveal(spans[len(spans)-1], area, false)
	case spans[i].top < ish.offset:
		ish.offset--
	case i > 0:
		ish.setCursor(ish.cursor - 1)
		ish.reveal(spans[i-1], area, false)
	}

	ish.clampOffset(spans, area)
}

// reveal adjusts the scroll offset so that s is visible in a card area of the
// given height. A card taller than the area is revealed from its top edge when
// moving down and from its bottom edge when moving up.
func (ish *Issue) reveal(s span, area int, down bool) {
	if s.height > area {
		if down && (s.top < ish.offset || s.top >= ish.offset+area) {
			ish.offset = s.top
		} else if !down && (s.bottom() <= ish.offset || s.bottom() > ish.offset+area) {
			ish.offset = s.bottom() - area
		}
		return
	}

	if s.top < ish.offset {
		ish.offset = s.top
	} else if s.bottom() > ish.offset+area {
		ish.offset = s.bottom() - area
	}
}

// clampOffset keeps the scroll offset within the bounds of the card content.
func (ish *Issue) clampOffset(spans []span, area int) {
	total := 0
	if len(spans) > 0 {
		total = spans[len(spans)-1].bottom()
	}

	ish.offset = min(ish.offset, max(total-area, 0))
	ish.offset = max(ish.offset, 0)
}

// layout returns the position of each rendered card and the height of the
// scrollable card area below the pinned header. Before the terminal size is
// known, the card area is tall enough to hold every card.
func (ish Issue) layout() ([]span, int) {
	_, spans := ish.renderCards()

	if ish.height <= 0 {
		total := 0
		if len(spans) > 0 {
			total = spans[len(spans)-1].bottom()
		}
		return spans, total
	}

	return spans, max(ish.height-lipgloss.Height(ish.renderHeader()), 0)
}

// renderHeader renders the title, byline, and metadata tags that stay pinned
// to the top of the screen.
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

	return styles.Header.Render(lipgloss.JoinVertical(lipgloss.Left, c...))
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

func (ish Issue) View() tea.View {
	header := ish.renderHeader()
	cards, _ := ish.renderCards()

	if ish.height > 0 {
		area := max(ish.height-lipgloss.Height(header), 0)
		lines := strings.Split(cards, "\n")
		end := min(ish.offset+area, len(lines))
		start := min(ish.offset, end)
		cards = strings.Join(lines[start:end], "\n")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, cards)

	if ish.width > 0 && ish.height > 0 {
		content = lipgloss.NewStyle().
			MaxWidth(ish.width).
			MaxHeight(ish.height).
			Render(content)
	}

	v := tea.NewView(content)
	v.AltScreen = true

	return v
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
