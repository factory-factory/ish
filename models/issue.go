package models

import (
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iridescent-software/ish/styles"
)

type Issue struct {
	id          string
	dateCreated string
	author      string
	title       string

	metadata []Tag

	conversation []Card

	selectedComment int
	cursor          int
}

const (
	ISO8601  = "2006-01-02T15:04:05Z0700"
	DateTime = "Jan 2, 2006" // 15:04:05 MST"
)

// type Cursor struct {
// }

// func (c Cursor) Next() {
// }

func InitialModel() Issue {
	return Issue{
		//
		id:          "00000",
		dateCreated: "2023-05-26T23:35:36-0400",
		author:      "brendanberg",
		title:       "What is ish?",

		metadata: []Tag{
			{
				key:    "state",
				values: []string{"open"},
			},
			{
				key:    "assignees",
				values: []string{"brendanberg", "jhc"},
			},
		},
		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		//conversation: make([]card, 2),
		conversation: []Card{
			{
				id:          "001",
				dateCreated: "2023-05-27T13:41:50-0400",
				author:      "brendanberg",
				content: `**ish** is a file format convention and a set of utilities for distributed, file-based issue tracking.

**ish** consists of a library and a command line utility.
The library provides a directory scanner, parser, and database API,
while the command line app provides a terminal UI to interact with issues.`,
			},
			{
				id:          "002",
				dateCreated: "2023-05-27T23:37:18-0400",
				author:      "brendanberg",
				content: `The **ish** command line tools rely on go modules from [charmbracelet](https://github.com/charmbracelet):

- [Bubbletea](https://github.com/charmbracelet/bubbletea), a terminal UI framework
- [Glamour](https://github.com/charmbracelet/glamour), a markdown formatting tool
- [Lipgloss](https://github.com/charmbracelet/lipgloss), a terminal style library

## TODO:

- [x] Learn the basics of the charmbracelet tools
- [ ] Write parser
- [ ] Build ish DB
- [ ] Create ish commands
  - [ ] ish init
  - [ ] ish new [TITLE]
  - [ ] ish list [QUERY]
  - [ ] ish edit [HASH]`,
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

		case "tab":
			metaLen := len(ish.metadata)
			convLen := len(ish.conversation)
			c := ish.cursor

			switch {
			case c >= 2 && c < metaLen+2:
				meta, _ := ish.metadata[c-2].Update(SelectMsg(false))
				ish.metadata[c-2] = meta.(Tag)
			case c >= metaLen+2 && c < metaLen+convLen+2:
				comment, _ := ish.conversation[c-metaLen-2].Update(SelectMsg(false))
				ish.conversation[c-metaLen-2] = comment.(Card)
			}

			c += 1

			switch {
			case c >= 2 && c < metaLen+2:
				meta, _ := ish.metadata[c-2].Update(SelectMsg(true))
				ish.metadata[c-2] = meta.(Tag)
			case c >= metaLen+2 && c < metaLen+convLen+2:
				comment, _ := ish.conversation[c-metaLen-2].Update(SelectMsg(true))
				ish.conversation[c-metaLen-2] = comment.(Card)
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
	t, err := time.Parse(ISO8601, ish.dateCreated)

	if err != nil {
		log.Fatal(err)
	}

	c := make([]string, 0)

	c = append(c, ish.getTitleStyle().Render(fmt.Sprintf("[%s] %s", ish.id, ish.title)))
	c = append(c, styles.BylineBlock.Copy().MarginBottom(1).Render(
		styles.BylineEmphasis.Render(ish.author)+
			styles.Byline.Render(" opened this issue on "+t.Format(DateTime)),
	))

	for _, tag := range ish.metadata {
		c = append(c, tag.View())
	}

	for _, card := range ish.conversation {
		c = append(c, card.View())
	}

	return styles.Block.Render(lipgloss.JoinVertical(lipgloss.Left, c...))
}

func (ish Issue) getTitleStyle() lipgloss.Style {
	switch ish.cursor {
	case 1:
		return styles.TitleSelected
	default:
		return styles.Title
	}
}
