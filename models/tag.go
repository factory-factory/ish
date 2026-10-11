package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/akrennmair/slice"
	"github.com/factory-factory/ish/styles"
)

type Tag struct {
	Key    string
	Values []string

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

func (t Tag) View() tea.View {
	vals := slice.Map(t.Values, func(s string) string {
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
		return tea.NewView(styles.TagSelected.Render("• "+t.Key+": ") + s)
	default:
		return tea.NewView(styles.Tag.Render("• "+t.Key+": ") + s)
	}
}
