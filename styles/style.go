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

// BadgeCorner and BadgeName style the ish badge in the issue list's upper
// left corner.
// Green ("2") also looks nice...
var BadgeCorner = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6"))

var BadgeName = lipgloss.NewStyle().
	Foreground(lipgloss.Color("5"))

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
	BorderForeground(lipgloss.Color("11")).
	Padding(0, 1)

var FilterBoxSelected = FilterBox.
	Border(lipgloss.ThickBorder(), true).
	BorderForeground(lipgloss.Color("4"))

// FilterTrigger is the filter menu's trigger, which keeps its thin, rounded
// border when it's selected, to tell it apart from the custom filter field.
var FilterTrigger = FilterBox.
	Foreground(lipgloss.Color("7"))

var FilterTriggerSelected = FilterTrigger.
	BorderForeground(lipgloss.Color("4")).
	Foreground(lipgloss.Color("15"))

// Button is a button beside the custom filter field, and ButtonSelected the
// selected button.
var Button = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7")).
	Background(lipgloss.Color("8"))

var ButtonCancelSelected = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("10")).
	Bold(true)

var ButtonSelected = lipgloss.NewStyle().
	Foreground(lipgloss.Color("15")).
	Background(lipgloss.Color("5")).
	Bold(true)

// TagDraft sets apart the values of a new tag while its key is edited
// again, in the color of keys that aren't being edited.
var TagDraft = lipgloss.NewStyle().
	Foreground(Tag.GetForeground())

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
