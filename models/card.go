package models

import (
	"log"
	"strings"
	"time"
	"unicode/utf8"

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

func (c Card) View() tea.View {
	t, err := time.Parse(time.RFC3339, c.DateCreated)

	if err != nil {
		t, err = time.Parse(ISO8601A, c.DateCreated)
	}

	if err != nil {
		t, err = time.Parse(ISO8601B, c.DateCreated)
	}

	if err != nil {
		t, err = time.Parse(Timestamp, c.DateCreated)
	}

	if err != nil {
		log.Fatal(err)
	}

	byline := styles.BylineBlock.Render(
		styles.BylineEmphasis.Render(c.Author) +
			styles.Byline.Render(" commented on "+t.Format(DateTime)),
	)

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
		if c.Content == nil {
			content, err = r.Render("")
		} else {
			content, err = r.Render(*c.Content)
		}
	}

	var comment string

	if err == nil {
		content = strings.TrimRight(content, "\n\t ")

		if utf8.RuneCountInString(content) == 0 {
			content = strings.Repeat(" ", 80)
		}

		comment = c.getStyle().Render(content)
	} else {
		comment = c.getStyle().Render(*c.Content)
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, byline, comment))
}

func (c Card) getStyle() lipgloss.Style {
	switch c.selected {
	case true:
		return styles.CardSelected
	default:
		return styles.Card
	}
}
