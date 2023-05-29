package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var Title = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("5")).
	Margin(0, 0, 0, 2).
	Padding(0, 1).
	Bold(true)

var TitleSelected = Title.Copy().
	Background(lipgloss.Color("4"))

var BylineBlock = lipgloss.NewStyle().
	Margin(1, 0, 0, 2).
	Padding(0)

var Byline = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11"))

var BylineEmphasis = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11")).
	Bold(true)

var Block = lipgloss.NewStyle().
	Margin(0, 0, 1).
	Padding(1)

var Card = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder(), true).
	BorderForeground(lipgloss.Color("11")).
	Foreground(lipgloss.Color("15")).
	MarginBottom(0).
	Padding(0, 1)

var CardSelected = Card.Copy().BorderForeground(lipgloss.Color("4"))

var Tag = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11"))

var TagSelected = Tag.Copy().Foreground(lipgloss.Color("4")).Bold(true)

var Value = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))

var Green = Value.Copy().Background(lipgloss.Color("2")).Padding(0, 1)
var Yellow = Value.Copy().Background(lipgloss.Color("3")).Padding(0, 1)
var Blue = Value.Copy().Background(lipgloss.Color("4")).Padding(0, 1)
var Magenta = Value.Copy().Background(lipgloss.Color("5")).Padding(0, 1)
var Aqua = Value.Copy().Background(lipgloss.Color("6")).Padding(0, 1)
var Grey = Value.Copy().Background(lipgloss.Color("10")).Padding(0, 1)
