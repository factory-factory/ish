package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/akrennmair/slice"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:              "ish",
		Short:            "A file-based distributed issue tracker",
		SilenceErrors:    false,
		SilenceUsage:     false,
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			p := tea.NewProgram(initialModel())
			if _, err := p.Run(); err != nil {
				fmt.Printf("Alas, there's been an error: %v", err)
				os.Exit(1)
			}
		},
	}
)

type issue struct {
	id          string
	dateCreated string
	author      string
	title       string

	metadata []tag

	conversation []card
}

type tag struct {
	key    string
	values []string
}

type card struct {
	id          string
	dateCreated string
	author      string
	content     string
}

func initialModel() issue {
	return issue{
		//
		id:          "00000",
		dateCreated: "2023-05-26T23:35:36-0400",
		author:      "brendanberg",
		title:       "What is ish?",

		metadata: []tag{
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
		conversation: []card{
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
	}
}

func (i issue) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (ish issue) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return ish, tea.Quit
		}
	}

	return ish, nil
}

const (
	ISO8601  = "2006-01-02T15:04:05Z0700"
	DateTime = "Jan 2, 2006" // 15:04:05 MST"
)

var titleStyle = lipgloss.NewStyle().
	Margin(0, 0, 0, 2).
	Padding(0, 1).
	Bold(true).
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("5"))

var bylineBlockStyle = lipgloss.NewStyle().
	Margin(1, 0, 0, 2).
	Padding(0)

var bylineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11"))

var bylineEmphasisStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11")).
	Bold(true)

var blockStyle = lipgloss.NewStyle().
	// Border(lipgloss.RoundedBorder(), true).
	Margin(0, 0, 1).
	Padding(1) //.Background(lipgloss.Color("2"))

var cardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder(), true).
	BorderForeground(lipgloss.Color("11")).
	Foreground(lipgloss.Color("15")).
	Padding(0, 1).
	MarginBottom(0)

var tagStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11"))

var valueStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))

var greenStyle = valueStyle.Copy().Background(lipgloss.Color("2")).Padding(0, 1)
var yellowStyle = valueStyle.Copy().Background(lipgloss.Color("3")).Padding(0, 1)
var blueStyle = valueStyle.Copy().Background(lipgloss.Color("4")).Padding(0, 1)
var magentaStyle = valueStyle.Copy().Background(lipgloss.Color("5")).Padding(0, 1)
var aquaStyle = valueStyle.Copy().Background(lipgloss.Color("6")).Padding(0, 1)
var greyStyle = valueStyle.Copy().Background(lipgloss.Color("10")).Padding(0, 1)

// var selectedCardStyle = cardStyle.Copy().BorderForeground(lipgloss.Color("15"))

func (ish issue) View() string {
	t, err := time.Parse(ISO8601, ish.dateCreated)

	if err != nil {
		log.Fatal(err)
	}

	c := make([]string, 0)

	c = append(c, titleStyle.Render(fmt.Sprintf("[%s] %s", ish.id, ish.title)))
	c = append(c, bylineBlockStyle.Copy().MarginBottom(1).Render(
		bylineEmphasisStyle.Render(ish.author)+
			bylineStyle.Render(" opened this issue on "+t.Format(DateTime)),
	))

	for _, tag := range ish.metadata {
		vals := slice.Map(tag.values, func(s string) string {
			switch s {
			case "open":
				return greyStyle.Render(s)
			case "in progress":
				return blueStyle.Render(s)
			case "in review":
				return blueStyle.Render(s)
			case "closed":
				return greenStyle.Render(s)
			default:
				return valueStyle.Render(s)
			}
		})

		s := strings.Join(vals, ", ")
		c = append(c, "• "+tagStyle.Render(tag.key+": ")+s)
	}

	for _, card := range ish.conversation {
		t, err := time.Parse(ISO8601, card.dateCreated)

		if err != nil {
			log.Fatal(err)
		}

		c = append(c, bylineBlockStyle.Render(
			bylineEmphasisStyle.Render(card.author)+
				bylineStyle.Render(" commented on "+t.Format(DateTime)),
		))

		content, err := glamour.Render(card.content, "default.json")

		if err == nil {
			c = append(c, cardStyle.Render(strings.TrimRight(content, "\n")))
		} else {
			c = append(c, cardStyle.Render(card.content))
		}

	}

	return blockStyle.Render(lipgloss.JoinVertical(lipgloss.Left, c...))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
