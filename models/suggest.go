package models

import (
	"slices"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/styles"
)

// MetadataValues maps each metadata key used in the issues to the distinct
// values used with it, to suggest as tags are edited. See
// Issue.SetMetadataValues.
type MetadataValues map[string][]string

// References holds what's suggested for a value that starts with # or @
// instead of the values used with the tag's key: the IDs of every issue, and
// the names of the repository's users. See Issue.SetReferences.
type References struct {
	IssueIDs  []string
	UserNames []string
}

// minSuggestion is the number of characters of a key or value that must be
// typed before completions are suggested.
const minSuggestion = 3

// acceptKey accepts the suggested completion of the key or value being
// typed. See Issue.updateEditing.
var acceptKey = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "accept suggestion"),
)

// tagInputStyles returns the styles for a tag's key and value inputs, with
// suggested completions in their own style.
func tagInputStyles(isDark bool) textinput.Styles {
	s := textinput.DefaultStyles(isDark)
	s.Focused.Suggestion = styles.Suggestion

	return s
}

// useSuggestions sets up input to show the first suggested completion after
// the cursor. Tab and enter accept it, which the issue view handles, so the
// input's own keys for accepting and cycling through suggestions are turned
// off.
func useSuggestions(input *textinput.Model) {
	input.ShowSuggestions = true
	input.KeyMap.AcceptSuggestion = key.NewBinding(key.WithDisabled())
}

// activeInput returns the input being edited: the key's or the values'.
func (t *Tag) activeInput() *textinput.Model {
	if t.editingKey {
		return &t.keyInput
	}

	return &t.input
}

// updateInput passes msg to the input being edited, then updates the
// suggested completions for what's in it. With a completion shown, backspace
// dismisses it, rather than deleting a character. Nothing more is suggested
// until characters are added.
func (t *Tag) updateInput(msg tea.Msg) tea.Cmd {
	input := t.activeInput()

	if msg, ok := msg.(tea.KeyPressMsg); ok && msg.String() == "backspace" && t.suggesting() {
		t.dismissed = true
		t.refreshSuggestions()
		return nil
	}

	before := input.Value()

	var cmd tea.Cmd
	*input, cmd = input.Update(msg)

	if utf8.RuneCountInString(input.Value()) > utf8.RuneCountInString(before) {
		t.dismissed = false
	}

	t.refreshSuggestions()

	return cmd
}

// refreshSuggestions sets the completions the input being edited can suggest
// for what's been typed. Keys are suggested from the keys used in the
// issues, leaving out those the issue already has. Values are suggested for
// the value after the last comma, from the values used with the tag's key,
// leaving out those already in the tag. Nothing is suggested until
// minSuggestion characters are typed, or after a suggestion is dismissed.
func (t *Tag) refreshSuggestions() {
	if !t.editingKey && !t.editing {
		return
	}

	input := t.activeInput()
	text := input.Value()

	// The value being typed is the one after the last comma, and the text
	// before it stays as it is.
	before, typed := "", text
	if !t.editingKey {
		i := strings.LastIndex(text, ",") + 1
		typed = strings.TrimLeft(text[i:], " ")
		before = text[:len(text)-len(typed)]
	}

	if t.dismissed || utf8.RuneCountInString(typed) < minSuggestion {
		input.SetSuggestions(nil)
		return
	}

	var candidates []string

	if t.editingKey {
		for k := range t.known {
			if !t.takenKeys[k] {
				candidates = append(candidates, k)
			}
		}
	} else {
		used := map[string]bool{}
		for _, v := range splitValues(before) {
			used[v] = true
		}

		for _, v := range t.valueCandidates(typed) {
			if !used[v] {
				candidates = append(candidates, before+v)
			}
		}
	}

	slices.SortFunc(candidates, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	input.SetSuggestions(candidates)
}

// valueCandidates returns the values that can be suggested for typed, the
// value being typed. A value starting with # is a reference to another
// issue, so every other issue's ID is suggested. One starting with @ refers
// to a user, so their names are suggested, except any with a comma, which
// would be split into two values. Otherwise, the values used with the tag's
// key are suggested.
func (t *Tag) valueCandidates(typed string) []string {
	var values []string

	switch {
	case strings.HasPrefix(typed, "#"):
		for _, id := range t.refs.IssueIDs {
			if id != t.issueID {
				values = append(values, "#"+id)
			}
		}

	case strings.HasPrefix(typed, "@"):
		for _, name := range t.refs.UserNames {
			if !strings.Contains(name, ",") {
				values = append(values, "@"+name)
			}
		}

	default:
		values = t.known[t.Key]
	}

	return values
}

// suggesting reports whether a completion is shown after the cursor in the
// input being edited.
func (t *Tag) suggesting() bool {
	if !t.editingKey && !t.editing {
		return false
	}

	input := t.activeInput()
	value := []rune(input.Value())

	return input.Position() == len(value) &&
		utf8.RuneCountInString(input.CurrentSuggestion()) > len(value)
}

// acceptSuggestion completes the input being edited with the suggestion shown
// after the cursor, reporting whether there was one.
func (t *Tag) acceptSuggestion() bool {
	if !t.suggesting() {
		return false
	}

	input := t.activeInput()
	input.SetValue(input.CurrentSuggestion())
	input.CursorEnd()
	t.refreshSuggestions()
	t.sizeEditor(t.width)

	return true
}
