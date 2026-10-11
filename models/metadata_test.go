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
