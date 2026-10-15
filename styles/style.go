package styles

import (
	"charm.land/lipgloss/v2"
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

var Header = lipgloss.NewStyle().
	Padding(1)

var Rule = lipgloss.NewStyle().
	Foreground(lipgloss.Color("8")).
	Margin(0, 1, 0)

var Body = lipgloss.NewStyle().
	Padding(0, 1)

var Footer = lipgloss.NewStyle().
	Padding(0, 2)

var Error = lipgloss.NewStyle().
	Foreground(lipgloss.Color("9"))

var cardBase = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	MarginBottom(0).
	Padding(0, 1)

var Card = cardBase.Copy().
	Border(lipgloss.RoundedBorder(), true).
	BorderForeground(lipgloss.Color("11"))

var CardSelected = cardBase.Copy().
	Border(lipgloss.ThickBorder(), true).
	BorderForeground(lipgloss.Color("4"))

var Tag = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11"))

var TagSelected = Tag.Copy().Foreground(lipgloss.Color("4")).Bold(true)

var TagDeleting = lipgloss.NewStyle().
	Foreground(lipgloss.Color("9")).
	Background(lipgloss.Color("8")).
	Bold(true)

var TodoSelected = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("4"))

// Link is the style of a link to another issue, and LinkSelected the style
// of the selected link.
var Link = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4")).
	Underline(true)

var LinkSelected = Link.Bold(true)

var Value = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))

var Green = Value.Copy().Background(lipgloss.Color("2")).Padding(0, 1)
var Yellow = Value.Copy().Background(lipgloss.Color("3")).Padding(0, 1)
var Blue = Value.Copy().Background(lipgloss.Color("4")).Padding(0, 1)
var Magenta = Value.Copy().Background(lipgloss.Color("5")).Padding(0, 1)
var Aqua = Value.Copy().Background(lipgloss.Color("6")).Padding(0, 1)
var Grey = Value.Copy().Background(lipgloss.Color("10")).Padding(0, 1)

var FilterLabel = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Padding(0, 1)

var FilterBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder(), true).
	BorderForeground(lipgloss.Color("8")).
	Padding(0, 1)

var FilterBoxSelected = FilterBox.
	Border(lipgloss.ThickBorder(), true).
	BorderForeground(lipgloss.Color("4"))

// TagDraft dims the values of a new tag while its key is edited again.
var TagDraft = lipgloss.NewStyle().
	Foreground(lipgloss.Color("8"))

var FilterPlaceholder = lipgloss.NewStyle().
	Foreground(lipgloss.Color("8"))

// Filter expression syntax highlighting.

var FilterBrace = lipgloss.NewStyle().
	Foreground(lipgloss.Color("4"))

var FilterAttribute = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5"))

var FilterOperator = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Bold(true)

var FilterValue = lipgloss.NewStyle().
	Foreground(lipgloss.Color("2"))

var FilterBoolean = FilterValue.
	Bold(true)

var FilterNull = lipgloss.NewStyle().
	Foreground(lipgloss.Color("11"))

var FilterParenthesis = lipgloss.NewStyle().
	Foreground(lipgloss.Color("3"))

var FilterKeyword = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))
