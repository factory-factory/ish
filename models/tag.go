package models

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/akrennmair/slice"
	"github.com/factory-factory/ish/styles"
)

type Tag struct {
	Key    string
	Values []string

	selected bool

	// editing reports whether the values are being edited in input.
	editing bool
	input   textinput.Model
}

func (t Tag) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (t Tag) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SelectMsg:
		t.selected = bool(msg)
		return t, nil
	}

	if t.editing {
		var cmd tea.Cmd
		t.input, cmd = t.input.Update(msg)
		return t, cmd
	}

	return t, nil
}

// label renders the tag's key, which is never editable.
func (t Tag) label() string {
	if t.selected {
		return styles.TagSelected.Render("• " + t.Key + ": ")
	}
	return styles.Tag.Render("• " + t.Key + ": ")
}

// startEdit begins editing the tag's values as comma-separated text in an
// input that fits, with the key, within width.
func (t *Tag) startEdit(width int, isDark bool) tea.Cmd {
	t.input = textinput.New()
	t.input.Prompt = ""
	t.input.SetStyles(textinput.DefaultStyles(isDark))
	t.input.SetValue(strings.Join(t.Values, ", "))
	t.input.CursorEnd()
	t.sizeEditor(width)
	t.editing = true

	return t.input.Focus()
}

// sizeEditor fits the input, with the key, within width.
func (t *Tag) sizeEditor(width int) {
	t.input.SetWidth(max(width-lipgloss.Width(t.label()), 1))
}

// commitEdit replaces the tag's values with the edited ones, reporting
// whether they changed. A tag must have a value, so if every value was
// removed the values are left unchanged.
func (t *Tag) commitEdit() bool {
	values := splitValues(t.input.Value())
	t.editing = false

	if len(values) == 0 || slices.Equal(values, t.Values) {
		return false
	}

	t.Values = values
	return true
}

// cancelEdit stops editing, leaving the values unchanged.
func (t *Tag) cancelEdit() {
	t.editing = false
}

// splitValues splits comma-separated text into trimmed, non-empty values.
func splitValues(s string) []string {
	values := []string{}

	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}

	return values
}

func (t Tag) View() tea.View {
	if t.editing {
		return tea.NewView(t.label() + t.input.View())
	}

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

	return tea.NewView(t.label() + strings.Join(vals, ", "))
}
