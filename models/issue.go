package models

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iridescent-software/ish/styles"
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return ish, tea.Quit

		case "tab", "shift+tab":
			metaLen := len(ish.Metadata)
			convLen := len(ish.Conversation)
			c := ish.cursor

			switch {
			case c >= 2 && c < metaLen+2:
				meta, _ := ish.Metadata[c-2].Update(SelectMsg(false))
				ish.Metadata[c-2] = meta.(Tag)
			case c >= metaLen+2 && c < metaLen+convLen+2:
				comment, _ := ish.Conversation[c-metaLen-2].Update(SelectMsg(false))
				ish.Conversation[c-metaLen-2] = comment.(Card)
			}

			switch msg.String() {
			case "tab":
				c += 1
			case "shift+tab":
				c -= 1
				if c < 0 {
					c = metaLen + convLen + 1
				}
			}

			switch {
			case c >= 2 && c < metaLen+2:
				meta, _ := ish.Metadata[c-2].Update(SelectMsg(true))
				ish.Metadata[c-2] = meta.(Tag)
			case c >= metaLen+2 && c < metaLen+convLen+2:
				comment, _ := ish.Conversation[c-metaLen-2].Update(SelectMsg(true))
				ish.Conversation[c-metaLen-2] = comment.(Card)
			case c >= metaLen+convLen+2:
				c = 0
			}

			ish.cursor = c

			return ish, nil
		}
	}

	return ish, nil
}

func (ish Issue) View() string {
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
		c = append(c, tag.View())
	}

	for _, card := range ish.Conversation {
		c = append(c, card.View())
	}

	return styles.Block.Render(lipgloss.JoinVertical(lipgloss.Left, c...))
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
