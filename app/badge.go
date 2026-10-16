package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

// Badge is the ish logo shown in the upper left corner of the issue list:
// the name between corner glyphs.
type Badge struct{}

// badgeGap is the space between the badge and the text of the filter label
// beside it.
const badgeGap = 5

func (b Badge) Init() tea.Cmd {
	return nil
}

func (b Badge) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return b, nil
}

func (b Badge) View() tea.View {
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		styles.BadgeCorner.Render("╭     ╮"),
		"  "+styles.BadgeName.Render("ish")+"  ",
		styles.BadgeCorner.Render("╰     ╯"),
	))
}

// width returns the width of the badge.
func (b Badge) width() int {
	return lipgloss.Width(b.View().Content)
}
