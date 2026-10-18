package models

import (
	"slices"
	"strings"
	"testing"

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

	if footer := footerText(ish); !strings.Contains(footer, "tab accept suggestion") {
		t.Errorf("footer %q with a suggestion, want tab to accept it", footer)
	}

	// Tab accepts the suggestion, and a second tab moves on to the values.
	ish = update(t, ish, tab)
	tag := ish.Metadata[1]

	if !tag.editingKey || tag.keyInput.Value() != "relation.blocked-by" || suggestion(ish) != "" {
		t.Fatalf("after tab: editing key, key, suggestion = %v, %q, %q, want the suggestion accepted in the key",
			tag.editingKey, tag.keyInput.Value(), suggestion(ish))
	}

	ish = update(t, ish, tab)

	if tag := ish.Metadata[1]; tag.Key != "relation.blocked-by" || tag.editingKey || !tag.editing {
		t.Errorf("after a second tab: key, editing key, editing values = %q, %v, %v, want relation.blocked-by, false, true",
			tag.Key, tag.editingKey, tag.editing)
	}
}

func TestEnterAcceptsKeySuggestion(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "rel")
	ish = update(t, ish, enter)

	// Enter accepts the suggestion and moves on to the values at once.
	if tag := ish.Metadata[1]; tag.Key != "relation.blocked-by" || tag.editingKey || !tag.editing {
		t.Errorf("after enter: key, editing key, editing values = %q, %v, %v, want relation.blocked-by, false, true",
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

func TestNoSuggestionsWhileDeleting(t *testing.T) {
	ish := suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "prio")

	// Dismissed, the suggestion stays away while deleting, even with 3
	// letters left.
	ish = update(t, ish, backspace, backspace)

	if key := ish.Metadata[1].keyInput.Value(); key != "pri" || suggestion(ish) != "" {
		t.Fatalf("key, suggestion = %q, %q, want pri and no suggestion", key, suggestion(ish))
	}

	// Adding a character suggests again.
	if ish = typeText(t, ish, "o"); suggestion(ish) != "priority" {
		t.Errorf("suggestion after typing = %q, want priority", suggestion(ish))
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

	// Tab accepts it, and keeps editing the values.
	ish = update(t, ish, tab)
	tag := ish.Metadata[1]

	if !tag.editing || tag.input.Value() != "zaphine, alice" {
		t.Fatalf("after tab: editing, values = %v, %q, want the suggestion accepted", tag.editing, tag.input.Value())
	}

	// Enter accepts a suggestion and saves the tag at once.
	ish = suggestingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "assignees")
	ish = update(t, ish, colon)
	ish = typeText(t, ish, "ali")
	ish = update(t, ish, enter)

	if tag := ish.Metadata[1]; tag.editing || !slices.Equal(tag.Values, []string{"alice"}) {
		t.Errorf("after enter: editing, values = %v, %q, want the suggestion accepted and the tag saved", tag.editing, tag.Values)
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

// referencingIssue returns an issue that suggests knownValues, issue IDs,
// including its own, and user names.
func referencingIssue(t *testing.T) Issue {
	t.Helper()

	ish := suggestingIssue(t)
	ish.SetReferences(References{
		IssueIDs:  []string{"00000", "AAAAA-11111", "BBBBB-22222", "CCCCC-33333"},
		UserNames: []string{"Brendan Berg", "Carol, Jr.", "brenda"},
	})

	return ish
}

// valueSuggestion adds a tag with key and types text into its values, and
// returns what's suggested.
func valueSuggestion(t *testing.T, ish Issue, key, text string) string {
	t.Helper()

	ish = update(t, ish, mKey)
	ish = typeText(t, ish, key)
	ish = update(t, ish, colon)
	ish = typeText(t, ish, text)

	return suggestion(ish)
}

func TestIssueIDSuggestions(t *testing.T) {
	for _, tt := range []struct {
		key, text, want string
	}{
		// Every issue's ID is suggested, not only the values used with
		// the key.
		{"relation.parent", "#AA", "#AAAAA-11111"},
		{"relation.parent", "#cc", "#CCCCC-33333"},
		{"related", "#BB", "#BBBBB-22222"},

		// Not the issue's own ID, or one already in the tag.
		{"relation.parent", "#00", ""},
		{"relation.parent", "#AAAAA-11111, #AA", ""},
		{"relation.parent", "#AAAAA-11111, #BB", "#AAAAA-11111, #BBBBB-22222"},

		// Below the threshold, counting the #, nothing is suggested.
		{"relation.parent", "#A", ""},
	} {
		if got := valueSuggestion(t, referencingIssue(t), tt.key, tt.text); got != tt.want {
			t.Errorf("%s: %q suggests %q, want %q", tt.key, tt.text, got, tt.want)
		}
	}

	// Values used with the key that aren't issue IDs aren't suggested.
	ish := referencingIssue(t)
	ish.SetMetadataValues(MetadataValues{"relation.parent": {"#ZZZZZ-99999"}})

	if got := valueSuggestion(t, ish, "relation.parent", "#ZZ"); got != "" {
		t.Errorf("#ZZ suggests %q, want none, as it isn't an issue", got)
	}
}

func TestUserNameSuggestions(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		// User names are suggested, regardless of case, in order.
		{"@Bre", "@brenda"},
		{"@bren", "@brenda"},
		{"@Brend", "@brenda"},
		{"@Brendan", "@Brendan Berg"},
		{"alice, @Brendan", "alice, @Brendan Berg"},

		// Not a name with a comma, which would split into two values.
		{"@Car", ""},
	} {
		if got := valueSuggestion(t, referencingIssue(t), "assignees", tt.text); got != tt.want {
			t.Errorf("%q suggests %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestEnterAcceptsReference(t *testing.T) {
	ish := referencingIssue(t)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "relation.parent")
	ish = update(t, ish, colon)
	ish = typeText(t, ish, "#BB")
	ish = update(t, ish, enter)

	if tag := ish.Metadata[1]; tag.editing || !slices.Equal(tag.Values, []string{"#BBBBB-22222"}) {
		t.Errorf("editing, values = %v, %q, want #BBBBB-22222 saved", tag.editing, tag.Values)
	}
}

func TestExistingTagsGetReferences(t *testing.T) {
	ish := referencingIssue(t)
	ish.setCursor(2)
	ish = update(t, ish, enter, ctrlU)
	ish = typeText(t, ish, "@Brendan")

	if got := suggestion(ish); got != "@Brendan Berg" {
		t.Errorf("suggestion = %q, want @Brendan Berg", got)
	}
}
