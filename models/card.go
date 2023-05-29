package models

import (
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/iridescent-software/ish/styles"
)

type Card struct {
	id          string
	dateCreated string
	author      string
	content     string

	selected bool
	// editing  bool
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
	}

	return c, nil
}

func (c Card) View() string {
	t, err := time.Parse(ISO8601, c.dateCreated)

	if err != nil {
		log.Fatal(err)
	}

	byline := styles.BylineBlock.Render(
		styles.BylineEmphasis.Render(c.author) +
			styles.Byline.Render(" commented on "+t.Format(DateTime)),
	)

	content, err := glamour.Render(c.content, "default.json")

	var comment string

	if err == nil {
		comment = c.getStyle().Render(strings.TrimRight(content, "\n"))
	} else {
		comment = c.getStyle().Render(c.content)
	}

	return lipgloss.JoinVertical(lipgloss.Left, byline, comment)
}

func (c Card) getStyle() lipgloss.Style {
	switch c.selected {
	case true:
		return styles.CardSelected
	default:
		return styles.Card
	}
}
