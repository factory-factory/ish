package models

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/styles"
)

var (
	mKey      = tea.KeyPressMsg{Code: 'm', Text: "m"}
	backspace = tea.KeyPressMsg{Code: tea.KeyBackspace}
	deleteKey = tea.KeyPressMsg{Code: tea.KeyDelete}
)

// taggingIssue returns an issue with a state tag and one comment, saved with
// s if it isn't nil.
func taggingIssue(t *testing.T, s *saver) Issue {
	t.Helper()

	ish := newTestIssue("Content")

	if s != nil {
		ish.SetSave(s.save)
	}

	return resize(t, ish, 20)
}

func TestAddTagEditsKey(t *testing.T) {
	ish := taggingIssue(t, nil)
	area := ish.viewport.Height()

	m, cmd := ish.Update(mKey)
	ish = m.(Issue)

	if len(ish.Metadata) != 2 {
		t.Fatalf("Metadata has %d tags, want 2", len(ish.Metadata))
	}

	tag := ish.Metadata[1]

	if tag.Key != "" || len(tag.Values) != 0 {
		t.Errorf("new tag = %q %q, want blank", tag.Key, tag.Values)
	}
	if ish.tagIndex() != 1 || !tag.selected || !tag.editingKey || !ish.editing {
		t.Errorf("tagIndex, selected, editing key, editing = %d, %v, %v, %v, want 1, true, true, true",
			ish.tagIndex(), tag.selected, tag.editingKey, ish.editing)
	}
	if cmd == nil {
		t.Error("m returned no command, want the key input's cursor to start")
	}

	// The header has another line, so the viewport is shorter.
	if got := ish.viewport.Height(); got != area-1 {
		t.Errorf("viewport height = %d, want %d", got, area-1)
	}
}

func TestAddTagKeyThenValue(t *testing.T) {
	ish := taggingIssue(t, nil)

	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "priority")
	ish = update(t, ish, enter)

	tag := ish.Metadata[1]

	// Setting the key moves on to editing the values.
	if tag.Key != "priority" || tag.editingKey || !tag.editing || !ish.editing {
		t.Fatalf("key, editing key, editing values = %q, %v, %v, want priority, false, true",
			tag.Key, tag.editingKey, tag.editing)
	}
	if view := plain(ish); !strings.Contains(view, "• priority: ") {
		t.Errorf("View() = %q, want the key beside the values input", view)
	}

	ish = typeText(t, ish, "high, urgent")
	ish = update(t, ish, enter)
	tag = ish.Metadata[1]

	// With both set, the tag stops editing but stays selected.
	if ish.editing || tag.editing || !tag.selected || ish.tagIndex() != 1 {
		t.Errorf("editing, tag editing, selected, tagIndex = %v, %v, %v, %d, want false, false, true, 1",
			ish.editing, tag.editing, tag.selected, ish.tagIndex())
	}
	if !slices.Equal(tag.Values, []string{"high", "urgent"}) || !ish.dirty {
		t.Errorf("values, dirty = %q, %v, want [high urgent], true", tag.Values, ish.dirty)
	}
	if view := plain(ish); !strings.Contains(view, "• priority: high, urgent") {
		t.Errorf("View() = %q, want the new tag", view)
	}
}

func TestNewTagKeyIgnoresSpaces(t *testing.T) {
	ish := taggingIssue(t, nil)

	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "due date")
	ish = update(t, ish, enter)

	if got := ish.Metadata[1].Key; got != "duedate" {
		t.Errorf("Key = %q, want %q", got, "duedate")
	}
}

func TestEmptyValuesKeepEditing(t *testing.T) {
	ish := taggingIssue(t, nil)

	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "priority")
	ish = update(t, ish, enter, enter)
	ish = typeText(t, ish, " , ")
	ish = update(t, ish, enter)

	if tag := ish.Metadata[1]; !ish.editing || !tag.editing {
		t.Errorf("editing, tag editing = %v, %v after saving no values, want still editing", ish.editing, tag.editing)
	}
}

func TestAbandonedNewTagIsRemoved(t *testing.T) {
	for _, tt := range []struct {
		name string
		msgs []tea.Msg
	}{
		{"empty key", []tea.Msg{enter}},
		{"blank key", []tea.Msg{tea.KeyPressMsg{Code: ' ', Text: " "}, enter}},
		{"cancelled key", []tea.Msg{tea.KeyPressMsg{Code: 'x', Text: "x"}, esc}},
		{"cancelled values", []tea.Msg{tea.KeyPressMsg{Code: 'x', Text: "x"}, enter, tea.KeyPressMsg{Code: 'y', Text: "y"}, esc}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &saver{}
			ish := taggingIssue(t, s)
			area := ish.viewport.Height()

			ish = update(t, ish, mKey)
			ish = update(t, ish, tt.msgs...)

			if ish.editing || len(ish.Metadata) != 1 {
				t.Fatalf("editing, tags = %v, %d, want the new tag removed", ish.editing, len(ish.Metadata))
			}

			// The previous last tag is selected instead.
			if ish.tagIndex() != 0 || !ish.Metadata[0].selected {
				t.Errorf("tagIndex() = %d, want the remaining tag selected", ish.tagIndex())
			}
			if got := ish.viewport.Height(); got != area {
				t.Errorf("viewport height = %d, want %d", got, area)
			}

			exitWith(t, ish, q)
			if len(s.saved) != 0 {
				t.Errorf("saved %d times, want 0", len(s.saved))
			}
		})
	}
}

func TestQuitWhileAddingTag(t *testing.T) {
	s := &saver{}
	ish := taggingIssue(t, s)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter, mKey)
	ish = typeText(t, ish, "priority")

	_, msg := exitWith(t, ish, ctrlC)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c sent %T, want tea.QuitMsg", msg)
	}
	if len(s.saved) != 1 || len(s.saved[0].Metadata) != 1 || s.saved[0].Title != "Title edited" {
		t.Errorf("saved %v, want the title edit without the unfinished tag", s.saved)
	}
}

func TestNewTagIsSaved(t *testing.T) {
	s := &saver{}
	ish := taggingIssue(t, s)

	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "priority")
	ish = update(t, ish, enter)
	ish = typeText(t, ish, "high")
	ish = update(t, ish, enter)

	exitWith(t, ish, q)

	if len(s.saved) != 1 {
		t.Fatalf("saved %d times, want 1", len(s.saved))
	}
	if tags := s.saved[0].Metadata; len(tags) != 2 || tags[1].Key != "priority" || !slices.Equal(tags[1].Values, []string{"high"}) {
		t.Errorf("saved metadata = %v, want the new priority tag", tags)
	}
}

func TestDeleteTag(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{backspace, deleteKey} {
		ish := taggingIssue(t, nil)
		ish.Metadata = append(ish.Metadata, Tag{Key: "priority", Values: []string{"high"}})
		ish.resize()
		area := ish.viewport.Height()
		ish.setCursor(2)

		// The first press highlights the tag.
		ish = update(t, ish, key)

		if !ish.confirmingDelete() || len(ish.Metadata) != 2 {
			t.Fatalf("after one %s: confirming, tags = %v, %d, want true, 2", key, ish.confirmingDelete(), len(ish.Metadata))
		}
		if view := ish.View().Content; !strings.Contains(view, styles.TagDeleting.Render("• state: open")) {
			t.Errorf("View() doesn't highlight the tag being deleted:\n%s", view)
		}
		if footer := footerText(ish); !strings.Contains(footer, "delete field") || !strings.Contains(footer, "any other key cancel") {
			t.Errorf("footer %q, want the delete confirmation keys", footer)
		}

		// The second press deletes it, and selects the tag that took its
		// place.
		ish = update(t, ish, key)

		if len(ish.Metadata) != 1 || ish.Metadata[0].Key != "priority" || !ish.dirty {
			t.Fatalf("after two %s: tags, dirty = %v, %v, want only priority, true", key, ish.Metadata, ish.dirty)
		}
		if ish.tagIndex() != 0 || !ish.Metadata[0].selected || ish.confirmingDelete() {
			t.Errorf("tagIndex, selected, confirming = %d, %v, %v, want 0, true, false",
				ish.tagIndex(), ish.Metadata[0].selected, ish.confirmingDelete())
		}
		if got := ish.viewport.Height(); got != area+1 {
			t.Errorf("viewport height = %d, want %d", got, area+1)
		}
	}
}

func TestDeleteLastTags(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish.Metadata = append(ish.Metadata, Tag{Key: "priority", Values: []string{"high"}})
	ish.resize()

	// Deleting the last tag selects the one before it.
	ish.setCursor(3)
	ish = update(t, ish, backspace, backspace)

	if len(ish.Metadata) != 1 || ish.tagIndex() != 0 {
		t.Fatalf("tags, tagIndex = %d, %d, want 1, 0", len(ish.Metadata), ish.tagIndex())
	}

	// Deleting the only tag selects the title.
	ish = update(t, ish, backspace, backspace)

	if len(ish.Metadata) != 0 || ish.cursor != 1 {
		t.Errorf("tags, cursor = %d, %d, want 0, 1", len(ish.Metadata), ish.cursor)
	}
}

func TestOtherKeyCancelsDelete(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish.setCursor(2)

	ish = update(t, ish, backspace, j)

	// The key only cancels; it doesn't also move the selection.
	if ish.confirmingDelete() || len(ish.Metadata) != 1 || ish.tagIndex() != 0 || ish.dirty {
		t.Errorf("confirming, tags, tagIndex, dirty = %v, %d, %d, %v, want false, 1, 0, false",
			ish.confirmingDelete(), len(ish.Metadata), ish.tagIndex(), ish.dirty)
	}

	// Moving away from a highlighted tag also cancels.
	ish = update(t, ish, backspace)
	ish.setCursor(1)

	if ish.Metadata[0].deleting {
		t.Error("tag still highlighted for deletion after the cursor moved away")
	}
}

func TestDeleteKeyElsewhere(t *testing.T) {
	ish := taggingIssue(t, nil)

	// Backspace on the title or a comment does nothing.
	for _, c := range []int{1, ish.firstCard()} {
		ish.setCursor(c)
		ish = update(t, ish, backspace, backspace)

		if len(ish.Metadata) != 1 {
			t.Errorf("tags = %d after backspace at cursor %d, want 1", len(ish.Metadata), c)
		}
	}

	if footer := footerText(ish); strings.Contains(footer, "delete field") {
		t.Errorf("footer %q lists deleting a field with a comment selected", footer)
	}

	// While editing a tag's values, backspace deletes a character.
	ish = editAt(t, ish, 2)
	ish = update(t, ish, backspace, enter)

	if len(ish.Metadata) != 1 || !slices.Equal(ish.Metadata[0].Values, []string{"ope"}) {
		t.Errorf("tags = %v, want state with one character deleted", ish.Metadata)
	}
}

// newTagLine returns the line of the issue's view showing the tag being
// added, without ANSI escape sequences.
func newTagLine(t *testing.T, ish Issue) string {
	t.Helper()

	lines := strings.Split(plain(ish), "\n")

	for i, line := range lines {
		if strings.Contains(line, "• state:") {
			return strings.TrimRight(lines[i+1], " ")
		}
	}

	t.Fatalf("no tag line in view:\n%s", plain(ish))
	return ""
}

func TestNewTagKeyStartsEmpty(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish = update(t, ish, mKey)

	// Only the bullet and the cursor are shown, with no stray characters
	// such as a placeholder under the cursor.
	if got := newTagLine(t, ish); got != " •" {
		t.Errorf("new tag line = %q, want only the bullet before anything is typed", got)
	}

	ish = typeText(t, ish, "priority")

	if got := newTagLine(t, ish); !strings.HasSuffix(got, "• priority") {
		t.Errorf("new tag line = %q, want the typed key", got)
	}
}

func TestTabAndColonMoveToValues(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{tab, {Code: ':', Text: ":"}} {
		ish := taggingIssue(t, nil)
		ish = update(t, ish, mKey)

		if footer := footerText(ish); !strings.Contains(footer, "tab/: values") || strings.Contains(footer, "shift+tab") {
			t.Errorf("footer %q while editing a new key, want tab/: and not shift+tab", footer)
		}

		ish = typeText(t, ish, "priority")
		ish = update(t, ish, key)
		tag := ish.Metadata[1]

		if tag.Key != "priority" || tag.editingKey || !tag.editing {
			t.Fatalf("after %s: key, editing key, editing values = %q, %v, %v, want priority, false, true",
				key, tag.Key, tag.editingKey, tag.editing)
		}
		if tag.input.Value() != "" {
			t.Errorf("after %s: values input = %q, want it empty", key, tag.input.Value())
		}
		if footer := footerText(ish); !strings.Contains(footer, "shift+tab key") || strings.Contains(footer, "tab/:") {
			t.Errorf("footer %q while editing a new tag's values, want shift+tab and not tab/:", footer)
		}

		// A colon in the values is just text.
		ish = typeText(t, ish, "a:b")
		ish = update(t, ish, enter)

		if got := ish.Metadata[1].Values; !slices.Equal(got, []string{"a:b"}) {
			t.Errorf("after %s: values = %q, want [a:b]", key, got)
		}
	}
}

func TestTabAndColonWithEmptyKey(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{tab, {Code: ':', Text: ":"}} {
		ish := taggingIssue(t, nil)
		ish = update(t, ish, mKey, key)
		tag := ish.Metadata[1]

		if len(ish.Metadata) != 2 || !tag.editingKey || tag.keyInput.Value() != "" {
			t.Errorf("after %s on an empty key: tags, editing key, key = %d, %v, %q, want the key still being edited",
				key, len(ish.Metadata), tag.editingKey, tag.keyInput.Value())
		}
	}
}

func TestShiftTabReturnsToKey(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "prio")
	ish = update(t, ish, tab)
	ish = typeText(t, ish, "high")
	ish = update(t, ish, shiftTab)
	tag := ish.Metadata[1]

	if !tag.editingKey || tag.editing || tag.keyInput.Value() != "prio" {
		t.Fatalf("editing key, editing values, key = %v, %v, %q, want the key being edited again",
			tag.editingKey, tag.editing, tag.keyInput.Value())
	}

	// Shift+tab does nothing more while the key is edited.
	ish = update(t, ish, shiftTab)
	ish = typeText(t, ish, "rity")
	ish = update(t, ish, tab)
	tag = ish.Metadata[1]

	if tag.Key != "priority" || !tag.editing || tag.input.Value() != "high" {
		t.Fatalf("key, editing values, values = %q, %v, %q, want priority with the values kept",
			tag.Key, tag.editing, tag.input.Value())
	}

	ish = update(t, ish, enter)

	if tag := ish.Metadata[1]; tag.Key != "priority" || !slices.Equal(tag.Values, []string{"high"}) {
		t.Errorf("saved tag %q: %q, want priority: [high]", tag.Key, tag.Values)
	}
}

func TestShiftTabOnExistingTag(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish.setCursor(2)
	ish = update(t, ish, enter)

	if footer := footerText(ish); strings.Contains(footer, "shift+tab") {
		t.Errorf("footer %q while editing an existing tag, want no shift+tab", footer)
	}

	ish = update(t, ish, shiftTab)

	if tag := ish.Metadata[0]; tag.editingKey || !tag.editing {
		t.Errorf("editing key, editing values = %v, %v after shift+tab on an existing tag, want its values still edited",
			tag.editingKey, tag.editing)
	}
}

func TestPasteKeyAndValues(t *testing.T) {
	tests := []struct {
		paste      string
		key        string
		editingKey bool
		values     string
	}{
		{"relation.parent: #3CSJ4-PEZ1N", "relation.parent", false, "#3CSJ4-PEZ1N"},
		{"priority:high, urgent  ", "priority", false, "high, urgent"},
		{"url: http://example.com", "url", false, "http://example.com"},
		{"priority:", "priority", false, ""},

		// Without a colon, the paste is the key.
		{"priority", "priority", true, ""},

		// With an empty key, a colon does nothing, so the values are dropped.
		{": high", "", true, ""},
	}

	for _, tt := range tests {
		ish := taggingIssue(t, nil)
		ish = update(t, ish, mKey, tea.PasteMsg{Content: tt.paste})
		tag := ish.Metadata[1]

		key := tag.Key
		if tag.editingKey {
			key = tag.editedKey()
		}

		if key != tt.key || tag.editingKey != tt.editingKey || tag.input.Value() != tt.values {
			t.Errorf("paste %q: key, editing key, values = %q, %v, %q, want %q, %v, %q",
				tt.paste, key, tag.editingKey, tag.input.Value(), tt.key, tt.editingKey, tt.values)
		}
	}
}

func TestPasteReplacesKeptValues(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "x")
	ish = update(t, ish, tab)
	ish = typeText(t, ish, "kept")
	ish = update(t, ish, shiftTab, ctrlU, tea.PasteMsg{Content: "state: done"})

	if tag := ish.Metadata[1]; tag.Key != "state" || tag.input.Value() != "done" {
		t.Errorf("key, values = %q, %q, want state, done", tag.Key, tag.input.Value())
	}
}

func TestValuesDimmedWhileKeyEditedAgain(t *testing.T) {
	ish := taggingIssue(t, nil)
	ish = update(t, ish, mKey)
	ish = typeText(t, ish, "prio")
	ish = update(t, ish, tab)
	ish = typeText(t, ish, "high, low")
	ish = update(t, ish, shiftTab)

	if header := ish.renderHeader(); !strings.Contains(header, styles.TagDraft.Render(" high, low")) {
		t.Errorf("header %q, want the values dimmed after the key", header)
	}

	// The colon, under the cursor, and the values follow the key without
	// a gap, as the key grows, and with the cursor within the key.
	if got := newTagLine(t, ish); !strings.Contains(got, "• prio: high, low") {
		t.Errorf("new tag line %q, want the values right after the key", got)
	}
	if header := ish.renderHeader(); strings.Contains(header, "prio\x1b[m ") {
		t.Errorf("header %q, want the cursor on the colon", header)
	}

	ish = typeText(t, ish, "rity")

	if got := newTagLine(t, ish); !strings.Contains(got, "• priority: high, low") {
		t.Errorf("new tag line %q after typing, want the values right after the key", got)
	}

	ish = update(t, ish, left, left)

	if got := newTagLine(t, ish); !strings.Contains(got, "• priority: high, low") {
		t.Errorf("new tag line %q with the cursor in the key, want the values right after it", got)
	}

	// With the key deleted, the colon still comes first.
	ish = update(t, ish, right, right, ctrlU)

	if got := newTagLine(t, ish); !strings.Contains(got, "• : high, low") {
		t.Errorf("new tag line %q with an empty key, want the colon and values", got)
	}

	ish = typeText(t, ish, "priority")

	// Moving on to the values shows them in the input again, undimmed.
	ish = update(t, ish, tab)

	if header := ish.renderHeader(); strings.Contains(header, styles.TagDraft.Render(" high, low")) {
		t.Errorf("header %q, want the values in their input", header)
	}
	if got := newTagLine(t, ish); !strings.Contains(got, "• priority: high, low") {
		t.Errorf("new tag line %q, want the values in their input", got)
	}
}
