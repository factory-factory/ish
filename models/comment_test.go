package models

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

var cKey = tea.KeyPressMsg{Code: 'c', Text: "c"}

// stubNow fixes the time for the duration of a test.
func stubNow(t *testing.T, at time.Time) {
	t.Helper()

	old := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = old })
}

var commentTime = time.Date(2026, 10, 11, 10, 30, 0, 0, time.FixedZone("", -7*60*60))

// commentingIssue returns an issue with the given cards whose new comments are
// by alice, saved with s if it isn't nil.
func commentingIssue(t *testing.T, s *saver, contents ...string) Issue {
	t.Helper()
	stubNow(t, commentTime)

	ish := newTestIssue(contents...)
	ish.SetAuthor("alice@example.com")

	if s != nil {
		ish.SetSave(s.save)
	}

	return resize(t, ish, 20)
}

func TestCommentKeyAddsAndEditsComment(t *testing.T) {
	ish := commentingIssue(t, nil, "First", "Second")
	m, cmd := ish.Update(cKey)
	ish = m.(Issue)

	if len(ish.Conversation) != 3 {
		t.Fatalf("Conversation has %d cards, want 3", len(ish.Conversation))
	}

	card := ish.Conversation[2]

	if card.Author != "alice@example.com" || card.DateCreated != "2026-10-11T10:30:00-07:00" {
		t.Errorf("new card byline = %q %q, want alice@example.com 2026-10-11T10:30:00-07:00", card.Author, card.DateCreated)
	}
	if card.Content == nil || *card.Content != "" {
		t.Errorf("new card content = %v, want empty", card.Content)
	}

	// The new card is selected and being edited.
	if ish.cardIndex() != 2 || !ish.editing || !card.editing || !card.selected {
		t.Errorf("cardIndex, editing, card editing, selected = %d, %v, %v, %v, want 2, true, true, true",
			ish.cardIndex(), ish.editing, card.editing, card.selected)
	}
	if ish.Conversation[0].selected || ish.Conversation[1].selected {
		t.Error("other cards are still selected")
	}
	if cmd == nil {
		t.Error("c returned no command, want the editor's cursor to start")
	}
	if footer := footerText(ish); !strings.Contains(footer, "ctrl+s save") {
		t.Errorf("footer %q, want the comment editing keys", footer)
	}
}

func TestNewCommentIsSaved(t *testing.T) {
	s := &saver{}
	ish := commentingIssue(t, s, "First")

	ish = update(t, ish, cKey)
	ish = typeText(t, ish, "A new comment")
	ish = update(t, ish, ctrlS)

	if ish.editing || len(ish.Conversation) != 2 || *ish.Conversation[1].Content != "A new comment" {
		t.Fatalf("editing, cards = %v, %d, want the saved comment as the second card", ish.editing, len(ish.Conversation))
	}
	if ish.cardIndex() != 1 {
		t.Errorf("cardIndex() = %d, want the new comment still selected", ish.cardIndex())
	}

	_, msg := exitWith(t, ish, q)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q sent %T, want tea.QuitMsg", msg)
	}
	if len(s.saved) != 1 || len(s.saved[0].Conversation) != 2 || *s.saved[0].Conversation[1].Content != "A new comment" {
		t.Errorf("saved %v, want one save with the new comment", s.saved)
	}
}

func TestAbandonedNewCommentIsRemoved(t *testing.T) {
	for _, tt := range []struct {
		name string
		msgs []tea.Msg
	}{
		{"cancelled", []tea.Msg{tea.KeyPressMsg{Code: 'x', Text: "x"}, esc}},
		{"saved empty", []tea.Msg{ctrlS}},
		{"saved blank", []tea.Msg{enter, tea.KeyPressMsg{Code: ' ', Text: " "}, ctrlS}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &saver{}
			ish := commentingIssue(t, s, "First", "Second")

			ish = update(t, ish, cKey)
			ish = update(t, ish, tt.msgs...)

			if ish.editing || len(ish.Conversation) != 2 {
				t.Fatalf("editing, cards = %v, %d, want the new comment removed", ish.editing, len(ish.Conversation))
			}

			// The previous last comment is selected instead.
			if ish.cardIndex() != 1 || !ish.Conversation[1].selected {
				t.Errorf("cardIndex() = %d, want the last remaining comment selected", ish.cardIndex())
			}

			exitWith(t, ish, q)
			if len(s.saved) != 0 {
				t.Errorf("saved %d times, want 0", len(s.saved))
			}
		})
	}
}

func TestAbandonedCommentOnIssueWithoutComments(t *testing.T) {
	ish := commentingIssue(t, nil)

	ish = update(t, ish, cKey)

	if len(ish.Conversation) != 1 || ish.cardIndex() != 0 || !ish.editing {
		t.Fatalf("cards, cardIndex, editing = %d, %d, %v, want a new comment being edited",
			len(ish.Conversation), ish.cardIndex(), ish.editing)
	}

	ish = update(t, ish, esc)

	if len(ish.Conversation) != 0 || ish.cursor != 0 {
		t.Errorf("cards, cursor = %d, %d after cancelling, want no cards and nothing selected",
			len(ish.Conversation), ish.cursor)
	}
}

func TestEditingExistingCommentIsNotRemoved(t *testing.T) {
	ish := commentingIssue(t, nil, "First")

	// Adding a comment and then cancelling an edit of an existing comment
	// leaves the existing comment alone.
	ish = update(t, ish, cKey)
	ish = typeText(t, ish, "New")
	ish = update(t, ish, ctrlS)
	ish = editAt(t, ish, ish.firstCard())
	ish = update(t, ish, ctrlU, ctrlS)

	if len(ish.Conversation) != 2 {
		t.Errorf("Conversation has %d cards, want 2", len(ish.Conversation))
	}
}

func TestCommentKeyWithoutAuthor(t *testing.T) {
	ish := resize(t, newTestIssue("First"), 20)
	ish = update(t, ish, cKey)

	if len(ish.Conversation) != 1 || ish.editing {
		t.Errorf("cards, editing = %d, %v, want c to do nothing without an author", len(ish.Conversation), ish.editing)
	}
	if footer := footerText(ish); strings.Contains(footer, "c comment") {
		t.Errorf("footer %q lists the comment key without an author", footer)
	}
}

func TestCommentKeyInOtherModes(t *testing.T) {
	// While editing a field, c is typed.
	ish := commentingIssue(t, nil, "First")
	ish = editAt(t, ish, 1)
	ish = update(t, ish, cKey)

	if got := ish.titleInput.Value(); got != "Titlec" || len(ish.Conversation) != 1 {
		t.Errorf("title input, cards = %q, %d, want c typed into the title", got, len(ish.Conversation))
	}

	// While selecting todo items, c adds a comment.
	ish = commentingIssue(t, nil, "- [ ] item")
	ish.setCursor(ish.firstCard())
	ish = update(t, ish, right, cKey)

	if len(ish.Conversation) != 2 || ish.cardIndex() != 1 || ish.Conversation[0].selecting {
		t.Errorf("cards, cardIndex, selecting todos = %d, %d, %v, want the new comment selected",
			len(ish.Conversation), ish.cardIndex(), ish.Conversation[0].selecting)
	}

	if footer := footerText(commentingIssue(t, nil, "First")); !strings.Contains(footer, "c comment") {
		t.Errorf("footer %q doesn't list the comment key", footer)
	}
}

func TestNewCommentScrollsIntoView(t *testing.T) {
	ish := commentingIssue(t, nil, lines(10), lines(10), lines(10))
	ish = update(t, ish, cKey)

	s := ish.spans[3]
	top, area := ish.viewport.YOffset(), ish.viewport.Height()

	if s.top < top || s.bottom() > top+area {
		t.Errorf("new comment %+v not in view at offset %d with area %d", s, top, area)
	}
}
