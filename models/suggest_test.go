package models

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/styles"
)

// knownValues are the metadata used in other issues, for suggestions.
var knownValues = MetadataValues{
	"assignees":           {"alice", "Bob", "zaphine"},
	"priority":            {"high", "low"},
	"relation.blocked-by": {"#AAAAA-11111"},
	"relation.parent":     {"#BBBBB-22222"},
	"state":               {"closed", "open"},
}

// suggestingIssue returns an issue with a state, that suggests knownValues.
func suggestingIssue(t *testing.T) Issue {
	t.Helper()

	ish := taggingIssue(t, nil)
	ish.SetMetadataValues(knownValues)

	return ish
}

// suggestion returns the suggestion shown for the input of the tag being
// edited, or "" if there isn't one.
func suggestion(ish Issue) string {
	tag := ish.Metadata[ish.tagIndex()]

	if !tag.suggesting() {
		return ""
	}

	return tag.activeInput().CurrentSuggestion()
}

func TestKeySuggestions(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)

	// Nothing is suggested until 3 letters are typed.
	if ish = typeText(t, ish, "re"); suggestion(ish) != "" {
		t.Errorf("suggestion after 2 letters = %q, want none", suggestion(ish))
	}

	ish = typeText(t, ish, "l")

	if got := suggestion(ish); got != "relation.blocked-by" {
		t.Fatalf("suggestion = %q, want the first match in order", got)
	}

	// The cursor stays in place, on the suggestion's next letter, and the
	// rest follows in its own style.
	if line := ish.renderHeader(); !strings.Contains(line, styles.Suggestion.Render("tion.blocked-by")) {
		t.Errorf("header %q, want the rest of the suggestion after the cursor", line)
	}
	if ish.Metadata[1].keyInput.Value() != "rel" {
		t.Errorf("key = %q, want only what was typed", ish.Metadata[1].keyInput.Value())
	}

	// Tab and shift+tab cycle through the matches.
	for _, tt := range []struct {
		key  tea.KeyPressMsg
		want string
	}{
		{tab, "relation.parent"},
		{shiftTab, "relation.blocked-by"},
	} {
		if ish = update(t, ish, tt.key); suggestion(ish) != tt.want {
			t.Errorf("after %s: suggestion = %q, want %q", tt.key, suggestion(ish), tt.want)
		}
	}

	if footer := footerText(ish); !strings.Contains(footer, "tab/shift+tab suggestions") {
		t.Errorf("footer %q with several suggestions, want the keys to cycle them", footer)
	}

	// Enter accepts the suggestion and moves on to the values.
	ish = update(t, ish, tab, enter)
	tag := ish.Metadata[1]

	if tag.Key != "relation.parent" || tag.editingKey || !tag.editing {
		t.Errorf("after enter: key, editing key, editing values = %q, %v, %v, want relation.parent, false, true",
			tag.Key, tag.editingKey, tag.editing)
	}
}

func TestKeySuggestionsIgnoreCaseAndTakenKeys(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)

	if ish = typeText(t, ish, "PRI"); suggestion(ish) != "priority" {
		t.Errorf("suggestion for PRI = %q, want priority", suggestion(ish))
	}

	// The issue already has a state, so it isn't suggested.
	ish = suggestingIssue(t)
	ish = update(t, ish, mKey)

	if ish = typeText(t, ish, "sta"); suggestion(ish) != "" {
		t.Errorf("suggestion for sta = %q, want none, as the issue has a state", suggestion(ish))
	}
}

func TestColonKeepsTypedKey(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "pri")
	ish = update(t, ish, colon)

	if tag := ish.Metadata[1]; tag.Key != "pri" || !tag.editing {
		t.Errorf("key, editing values = %q, %v after a colon, want pri, true", tag.Key, tag.editing)
	}
}

func TestBackspaceDismissesSuggestion(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "pri")
	ish = update(t, ish, backspace)

	if key := ish.Metadata[1].keyInput.Value(); key != "pri" || suggestion(ish) != "" {
		t.Fatalf("after backspace: key, suggestion = %q, %q, want pri and no suggestion", key, suggestion(ish))
	}

	// Then backspace deletes, and typing suggests again.
	ish = update(t, ish, backspace)

	if key := ish.Metadata[1].keyInput.Value(); key != "pr" {
		t.Errorf("after a second backspace: key = %q, want pr", key)
	}

	if ish = typeText(t, ish, "i"); suggestion(ish) != "priority" {
		t.Errorf("after typing again: suggestion = %q, want priority", suggestion(ish))
	}

	// With the suggestion dismissed, enter saves only what was typed.
	ish = update(t, ish, backspace, enter)

	if tag := ish.Metadata[1]; tag.Key != "pri" {
		t.Errorf("key = %q after enter with the suggestion dismissed, want pri", tag.Key)
	}
}

func TestValueSuggestions(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "assignees")
	ish = update(t, ish, colon)

	if ish = typeText(t, ish, "al"); suggestion(ish) != "" {
		t.Errorf("suggestion after 2 letters = %q, want none", suggestion(ish))
	}

	// Each comma-separated value is suggested separately, regardless of
	// case.
	ish = update(t, ish, ctrlU)
	ish = typeText(t, ish, "zaphine, ALI")

	if got := suggestion(ish); got != "zaphine, alice" {
		t.Fatalf("suggestion = %q, want alice after zaphine", got)
	}

	// Enter accepts it, then saves the tag.
	ish = update(t, ish, enter)
	tag := ish.Metadata[1]

	if !tag.editing || tag.input.Value() != "zaphine, alice" {
		t.Fatalf("after enter: editing, values = %v, %q, want the suggestion accepted", tag.editing, tag.input.Value())
	}

	ish = update(t, ish, enter)

	if tag := ish.Metadata[1]; tag.editing || !slices.Equal(tag.Values, []string{"zaphine", "alice"}) {
		t.Errorf("after a second enter: editing, values = %v, %q, want the tag saved", tag.editing, tag.Values)
	}
}

func TestValueSuggestionsLeaveOutTagsValues(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "assignees")
	ish = update(t, ish, colon)
	ish = typeText(t, ish, "alice, ali")

	if got := suggestion(ish); got != "" {
		t.Errorf("suggestion = %q, want none, as the tag has alice", got)
	}
}

func TestExistingTagValueSuggestions(t *testing.T) {
	ish := suggestingIssue(t)
	ish.setCursor(2)
	ish = update(t, ish, enter, ctrlU)
	ish = typeText(t, ish, "clo")

	if got := suggestion(ish); got != "closed" {
		t.Errorf("suggestion = %q, want closed", got)
	}
}

func TestNoSuggestionsWithoutMetadataValues(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish = update(t, ish, mKey)

	if ish = typeText(t, ish, "pri"); suggestion(ish) != "" {
		t.Errorf("suggestion = %q, want none without metadata values", suggestion(ish))
	}
}

func TestSuggestionStyle(t *testing.T) {
	if got := tagInputStyles(true).Focused.Suggestion.GetForeground(); got != styles.Suggestion.GetForeground() {
		t.Errorf("suggestion color = %v, want %v", got, styles.Suggestion.GetForeground())
	}
}
