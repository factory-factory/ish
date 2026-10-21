package models

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// saver records the issues it's asked to save, failing with err if set.
type saver struct {
	saved []Issue
	err   error
}

func (s *saver) save(issue Issue) error {
	if s.err != nil {
		return s.err
	}

	s.saved = append(s.saved, issue)
	return nil
}

var q = tea.KeyPressMsg{Code: 'q', Text: "q"}

// newSavingIssue returns a closable issue that saves with s.
func newSavingIssue(t *testing.T, s *saver) Issue {
	t.Helper()

	ish := newTestIssue("Content")
	ish.SetClosable(true)
	ish.SetSave(s.save)

	return resize(t, ish, 20)
}

// exitWith sends msg and returns the issue and the message produced by the
// resulting command, if any.
func exitWith(t *testing.T, ish Issue, msg tea.Msg) (Issue, tea.Msg) {
	t.Helper()

	m, cmd := ish.Update(msg)

	if cmd == nil {
		return m.(Issue), nil
	}

	return m.(Issue), cmd()
}

func TestExitWithoutChangesDoesNotSave(t *testing.T) {
	s := &saver{}
	ish := newSavingIssue(t, s)

	// Saving a field without changing it isn't a change.
	ish = editAt(t, ish, 1)
	ish = update(t, ish, enter)
	ish = editAt(t, ish, 2)
	ish = update(t, ish, enter)
	ish = editAt(t, ish, ish.firstCard())
	ish = update(t, ish, ctrlS)

	_, msg := exitWith(t, ish, q)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q sent %T, want tea.QuitMsg", msg)
	}
	if len(s.saved) != 0 {
		t.Errorf("saved %d times, want 0", len(s.saved))
	}
}

func TestQuitSavesChanges(t *testing.T) {
	s := &saver{}
	ish := newSavingIssue(t, s)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter)

	if len(s.saved) != 0 {
		t.Fatal("saved before exiting")
	}

	_, msg := exitWith(t, ish, q)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q sent %T, want tea.QuitMsg", msg)
	}
	if len(s.saved) != 1 || s.saved[0].Title != "Title edited" {
		t.Errorf("saved %v, want one save with the edited title", s.saved)
	}
}

func TestBackSavesChanges(t *testing.T) {
	s := &saver{}
	ish := newSavingIssue(t, s)

	ish = editAt(t, ish, 2)
	ish = typeText(t, ish, ", blocked")
	ish = update(t, ish, enter)
	ish = editAt(t, ish, ish.firstCard())
	ish = typeText(t, ish, "New ")
	ish = update(t, ish, ctrlS)

	ish, msg := exitWith(t, ish, esc)

	if _, ok := msg.(CloseMsg); !ok {
		t.Errorf("esc sent %T, want CloseMsg", msg)
	}
	if len(s.saved) != 1 {
		t.Fatalf("saved %d times, want 1", len(s.saved))
	}

	saved := s.saved[0]

	if got := strings.Join(Canonical(saved.Metadata[0].Values), ","); got != "open,blocked" {
		t.Errorf("saved state = %q, want %q", got, "open,blocked")
	}
	if got := *saved.Conversation[0].Content; got != "New Content" {
		t.Errorf("saved content = %q, want %q", got, "New Content")
	}

	// Once saved, exiting again doesn't save again.
	exitWith(t, ish, esc)
	if len(s.saved) != 1 {
		t.Errorf("saved %d times after exiting again, want 1", len(s.saved))
	}
}

func TestSaveFailure(t *testing.T) {
	s := &saver{err: errors.New("disk full")}
	ish := newSavingIssue(t, s)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter)

	// The first exit fails to save, so the issue stays open with the error.
	ish, msg := exitWith(t, ish, q)

	if msg != nil {
		t.Errorf("q after a failed save sent %T, want nothing", msg)
	}
	if footer := plain(ish); !strings.Contains(footer, "disk full") {
		t.Errorf("View() = %q, want the save error", footer)
	}
	if got := len(strings.Split(ish.View().Content, "\n")); got != ish.height {
		t.Errorf("View() has %d lines, want %d", got, ish.height)
	}

	// Exiting again discards the changes.
	_, msg = exitWith(t, ish, q)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("second q sent %T, want tea.QuitMsg", msg)
	}
}

func TestEditAfterSaveFailureRetries(t *testing.T) {
	s := &saver{err: errors.New("disk full")}
	ish := newSavingIssue(t, s)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter)
	ish, _ = exitWith(t, ish, esc)

	// Another edit clears the error, so the next exit tries to save again.
	s.err = nil
	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " again")
	ish = update(t, ish, enter)

	if footer := plain(ish); strings.Contains(footer, "disk full") {
		t.Errorf("View() = %q, want the save error cleared", footer)
	}

	_, msg := exitWith(t, ish, esc)

	if _, ok := msg.(CloseMsg); !ok {
		t.Errorf("esc sent %T, want CloseMsg", msg)
	}
	if len(s.saved) != 1 || s.saved[0].Title != "Title edited again" {
		t.Errorf("saved %v, want one save with both edits", s.saved)
	}
}

func TestCtrlCWhileEditingSavesEarlierChanges(t *testing.T) {
	s := &saver{}
	ish := newSavingIssue(t, s)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter)

	// The field being edited when quitting is discarded.
	ish = editAt(t, ish, 2)
	ish = typeText(t, ish, ", blocked")

	_, msg := exitWith(t, ish, ctrlC)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c sent %T, want tea.QuitMsg", msg)
	}
	if len(s.saved) != 1 {
		t.Fatalf("saved %d times, want 1", len(s.saved))
	}
	if saved := s.saved[0]; saved.Title != "Title edited" || strings.Join(Canonical(saved.Metadata[0].Values), ",") != "open" {
		t.Errorf("saved title %q, state %q, want the saved title and unchanged state", saved.Title, saved.Metadata[0].Values)
	}
}

func TestEmptyTagValuesAreNotSaved(t *testing.T) {
	s := &saver{}
	ish := newSavingIssue(t, s)

	ish = editAt(t, ish, 2)
	ish = update(t, ish, ctrlU, enter)

	if got := strings.Join(Canonical(ish.Metadata[0].Values), ","); got != "open" {
		t.Errorf("state = %q after removing every value, want it unchanged", got)
	}

	exitWith(t, ish, q)
	if len(s.saved) != 0 {
		t.Errorf("saved %d times, want 0", len(s.saved))
	}
}

func TestExitWithoutSaveFunc(t *testing.T) {
	ish := resize(t, newTestIssue("Content"), 20)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter)

	if _, msg := exitWith(t, ish, q); msg == nil {
		t.Error("q with unsaved changes and no save function didn't quit")
	}
}
