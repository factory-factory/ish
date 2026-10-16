package models

import (
	"slices"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

type Tag struct {
	Key    string
	Values []string

	selected bool

	// editing reports whether the values are being edited in input.
	editing bool
	input   textinput.Model

	// editingKey reports whether the key of a new tag is being edited in
	// keyInput.
	editingKey bool
	keyInput   textinput.Model

	// draft holds the values typed for a new tag while its key is edited
	// again, to be restored when its values are. They're shown dimmed after
	// the key.
	draft string

	// width is the width the tag's editor fits within.
	width int

	// deleting reports whether the tag is highlighted, waiting for the user
	// to confirm that it should be deleted.
	deleting bool

	// selecting reports whether the links in the tag's values are being
	// selected, and link is the index of the selected one.
	selecting bool
	link      int

	// issues reports which issues exist, to link references to them.
	issues IssueLookup
}

func (t Tag) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (t Tag) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SelectMsg:
		t.selected = bool(msg)

		if !t.selected {
			t.deleting = false
			t.stopLinks()
		}

		return t, nil
	}

	if t.editingKey {
		// Keys are single words, so spaces aren't typed.
		if msg, ok := msg.(tea.KeyPressMsg); ok && strings.ContainsFunc(msg.Text, unicode.IsSpace) {
			return t, nil
		}

		cmd := t.updateKey(msg)
		return t, cmd
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

// keyPrefix is shown before a new tag's key while it's being edited.
const keyPrefix = "• "

// startKeyEdit begins editing the key of a new tag in an input that fits
// within width.
func (t *Tag) startKeyEdit(width int, isDark bool) tea.Cmd {
	t.keyInput = textinput.New()
	t.keyInput.Prompt = ""
	t.keyInput.SetStyles(textinput.DefaultStyles(isDark))
	t.sizeEditor(width)
	t.editingKey = true

	return t.keyInput.Focus()
}

// editedKey returns the key being edited, without any whitespace, which a key
// can't contain.
func (t Tag) editedKey() string {
	return strings.Join(strings.Fields(t.keyInput.Value()), "")
}

// commitKey sets the tag's key to the edited key, and stops editing it.
func (t *Tag) commitKey() {
	t.Key = t.editedKey()
	t.editingKey = false
}

// startEdit begins editing the tag's values as comma-separated text in an
// input that fits, with the key, within width.
func (t *Tag) startEdit(width int, isDark bool) tea.Cmd {
	t.input = textinput.New()
	t.input.Prompt = ""
	t.input.SetStyles(textinput.DefaultStyles(isDark))
	t.input.SetValue(strings.Join(t.Values, ", "))

	if t.draft != "" {
		t.input.SetValue(t.draft)
		t.draft = ""
	}

	t.input.CursorEnd()
	t.sizeEditor(width)
	t.editing = true

	return t.input.Focus()
}

// editKeyAgain stops editing a new tag's values and returns to editing its
// key, keeping the values typed so far for when its values are edited again.
func (t *Tag) editKeyAgain() tea.Cmd {
	t.draft = t.input.Value()
	t.input.Blur()
	t.editing = false
	t.editingKey = true
	t.sizeEditor(t.width)

	return t.keyInput.Focus()
}

// updateKey passes msg to the key input. While it's only as wide as the key,
// it would scroll as the key grows, so it's given the full width while it's
// updated.
func (t *Tag) updateKey(msg tea.Msg) tea.Cmd {
	t.keyInput.SetWidth(max(t.width-lipgloss.Width(keyPrefix), 1))

	var cmd tea.Cmd
	t.keyInput, cmd = t.keyInput.Update(msg)
	t.sizeEditor(t.width)

	return cmd
}

// draftSeparator separates the key being edited from the dimmed values
// after it.
const draftSeparator = ": "

// pasteKey handles text pasted while a new tag's key is being edited. The
// text before the first colon is pasted into the key, and the text after it,
// without surrounding space, is returned as values, reporting whether there
// was a colon.
func (t *Tag) pasteKey(text string) (values string, split bool) {
	key, values, split := strings.Cut(text, ":")

	t.updateKey(tea.PasteMsg{Content: key})

	return strings.TrimSpace(values), split
}

// sizeEditor fits the input, with the key, within width.
func (t *Tag) sizeEditor(width int) {
	t.width = width

	if t.editingKey {
		available := width - lipgloss.Width(keyPrefix)

		// The input is padded to its width, so with values shown after it,
		// it's only as wide as the key. The cursor at the end is drawn
		// beyond it, where the colon is shown. An empty key has no width,
		// so it isn't padded at all.
		if t.draft != "" {
			available = min(available-lipgloss.Width(draftSeparator+t.draft), lipgloss.Width(t.keyInput.Value()))
			t.keyInput.SetWidth(max(available, 0))
			return
		}

		t.keyInput.SetWidth(max(available, 1))
		return
	}

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
	if t.editingKey {
		key := t.keyInput.View()

		if t.draft == "" {
			return tea.NewView(styles.TagSelected.Render(keyPrefix) + key)
		}

		// The input ends with a blank cell: the cursor, or the padding
		// that stands in for it when the cursor is within the key. The
		// colon goes in that cell, so it follows the key directly, and the
		// cursor at the end is drawn on it.
		if i := strings.LastIndex(key, " "); i >= 0 {
			key = key[:i] + ":" + key[i+1:]
		}

		return tea.NewView(styles.TagSelected.Render(keyPrefix) + key +
			styles.TagDraft.Render(draftSeparator[1:]+t.draft))
	}

	if t.deleting {
		return tea.NewView(styles.TagDeleting.Render(
			"• " + t.Key + ": " + strings.Join(t.Values, ", "),
		))
	}

	if t.editing {
		return tea.NewView(t.label() + t.input.View())
	}

	vals := make([]string, len(t.Values))

	for i, s := range t.Values {
		vals[i] = t.renderValue(i, s)
	}

	return tea.NewView(t.label() + strings.Join(vals, ", "))
}

// renderValue renders the value at index i, s, with any links in it, or in
// the color of its state.
func (t Tag) renderValue(i int, s string) string {
	if refs := findReferences(s, t.issues); len(refs) > 0 {
		return t.renderLinkedValue(i, refs)
	}

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
}

// tagLink is a link in one of a tag's values.
type tagLink struct {
	value int
	ref   reference
}

// links returns the links in the tag's values, in order.
func (t Tag) links() []tagLink {
	var links []tagLink

	for i, v := range t.Values {
		for _, ref := range findReferences(v, t.issues) {
			links = append(links, tagLink{value: i, ref: ref})
		}
	}

	return links
}

// startLinks begins selecting the tag's links, starting with the first,
// reporting whether it has any.
func (t *Tag) startLinks() bool {
	if len(t.links()) == 0 {
		return false
	}

	t.selecting = true
	t.link = 0

	return true
}

// stopLinks stops selecting the tag's links.
func (t *Tag) stopLinks() {
	t.selecting = false
}

// moveLink moves the selection by delta links, stopping at the first and
// last links.
func (t *Tag) moveLink(delta int) {
	t.link = min(max(t.link+delta, 0), len(t.links())-1)
}

// selectedLink returns the ID of the issue the selected link refers to, if
// links are being selected.
func (t Tag) selectedLink() (string, bool) {
	links := t.links()

	if !t.selecting || t.link < 0 || t.link >= len(links) {
		return "", false
	}

	return links[t.link].ref.id, true
}

// renderLinkedValue renders the value at index i, whose references are refs,
// with its links styled and the selected link highlighted.
func (t Tag) renderLinkedValue(i int, refs []reference) string {
	var selected *reference
	links := t.links()

	if t.selecting && t.link >= 0 && t.link < len(links) && links[t.link].value == i {
		selected = &links[t.link].ref
	}

	marked := t.Values[i]

	for i := len(refs) - 1; i >= 0; i-- {
		marker := linkMarker
		if selected != nil && *selected == refs[i] {
			marker = selectedLinkMarker
		}

		marked = marked[:refs[i].start] + marker + marked[refs[i].start:refs[i].end] + linkEndMarker + marked[refs[i].end:]
	}

	return renderLinks(marked, styles.Value, styles.Link, styles.LinkSelected)
}
