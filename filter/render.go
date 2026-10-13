package filter

import (
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

// printer styles each kind of token when an expression is formatted as text.
type printer struct {
	brace     func(string) string
	attribute func(string) string
	operator  func(string) string
	paren     func(string) string
	keyword   func(string) string
	value     func(Value) string
}

func unstyled(s string) string { return s }

// plain formats an expression as plain text, for String.
var plain = printer{
	brace:     unstyled,
	attribute: unstyled,
	operator:  unstyled,
	paren:     unstyled,
	keyword:   unstyled,
	value:     Value.String,
}

// highlight formats an expression with each kind of token colored, for
// Render.
var highlight = printer{
	brace:     with(styles.FilterBrace),
	attribute: with(styles.FilterAttribute),
	operator:  with(styles.FilterOperator),
	paren:     with(styles.FilterParenthesis),
	keyword:   with(styles.FilterKeyword),
	value:     renderValue,
}

// with returns a function that renders text in style.
func with(style lipgloss.Style) func(string) string {
	return func(s string) string { return style.Render(s) }
}

// renderValue colors a value by its kind: strings, numbers, and regular
// expressions are green, true and false are bold green, and null is grey.
func renderValue(v Value) string {
	switch v.Data.(type) {
	case nil:
		return styles.FilterNull.Render(v.String())
	case bool:
		return styles.FilterBoolean.Render(v.String())
	default:
		return styles.FilterValue.Render(v.String())
	}
}
