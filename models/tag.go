package models

import (
	"strings"

	"github.com/akrennmair/slice"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/iridescent-software/ish/styles"
)

type Tag struct {
	key    string
	values []string

	selected bool
}

func (t Tag) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (t Tag) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SelectMsg:
		t.selected = bool(msg)
	}

	return t, nil
}

func (t Tag) View() string {
	vals := slice.Map(t.values, func(s string) string {
		switch s {
		case "open":
			return styles.Grey.Render(s)
		case "in progress":
			return styles.Blue.Render(s)
		case "in review":
			return styles.Blue.Render(s)
		case "closed":
			return styles.Green.Render(s)
		default:
			return styles.Value.Render(s)
		}
	})

	s := strings.Join(vals, ", ")

	switch t.selected {
	case true:
		return "• " + styles.TagSelected.Render(t.key+": ") + s
	default:
		return "• " + styles.Tag.Render(t.key+": ") + s
	}
}
