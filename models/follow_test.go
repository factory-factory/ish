package models

import (
	"errors"
	"strings"
	"testing"
)

func TestFollowKeepsSelectionAndScroll(t *testing.T) {
	old := resize(t, newTestIssue(lines(20), lines(20), lines(20)), 10)
	old.setCursor(old.firstCard() + 1)
	old.viewport.SetYOffset(25)

	fresh := newTestIssue(lines(20), lines(20), lines(20))
	fresh.Follow(old)

	if fresh.cursor != old.cursor {
		t.Errorf("cursor = %d, want %d", fresh.cursor, old.cursor)
	}

	if got := offset(fresh); got != 25 {
		t.Errorf("offset = %d, want 25", got)
	}

	if fresh.width != old.width || fresh.height != old.height || !fresh.shown {
		t.Errorf("size %dx%d, shown %v; want %dx%d, shown", fresh.width, fresh.height, fresh.shown, old.width, old.height)
	}
}

func TestFollowWithFewerCardsAndTags(t *testing.T) {
	old := resize(t, newTestIssue("a", "b", "c"), 10)
	old.setCursor(old.firstCard() + 2)

	fresh := newTestIssue("a")
	fresh.Follow(old)

	if got := fresh.cardIndex(); got != 0 {
		t.Errorf("card %d selected, want the last card left, 0", got)
	}

	// With the selected tag gone, the title is selected.
	old.setCursor(2)

	fresh = newTestIssue("a")
	fresh.Metadata = nil
	fresh.Follow(old)

	if fresh.cursor != 1 {
		t.Errorf("cursor = %d, want the title, 1", fresh.cursor)
	}
}

func TestUnsaved(t *testing.T) {
	ish := resize(t, newTestIssue("Content"), 10)

	if ish.Unsaved() {
		t.Error("Unsaved() = true for an issue just opened")
	}

	// A field being edited is unsaved, even before it's changed.
	ish = editAt(t, ish, 1)

	if !ish.Unsaved() {
		t.Error("Unsaved() = false while the title is being edited")
	}

	ish = typeText(t, ish, "!")
	ish = update(t, ish, enter)

	if !ish.Unsaved() {
		t.Error("Unsaved() = false after the title was changed")
	}
}

func TestSaveWritesNow(t *testing.T) {
	s := &saver{}
	ish := newSavingIssue(t, s)

	// Saving without changes writes the issue too, to overwrite its file.
	if err := ish.Save(); err != nil || len(s.saved) != 1 {
		t.Fatalf("Save() = %v, saved %d issues; want one", err, len(s.saved))
	}

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, "!")
	ish = update(t, ish, enter)

	if err := ish.Save(); err != nil || ish.Unsaved() {
		t.Errorf("Save() = %v, Unsaved() = %v; want the changes saved", err, ish.Unsaved())
	}

	s.err = errors.New("disk full")

	if err := ish.Save(); err == nil || !strings.Contains(plain(ish), "Couldn't save changes: disk full") {
		t.Errorf("Save() = %v, want the error, shown:\n%s", err, plain(ish))
	}
}

func TestSetNotice(t *testing.T) {
	ish := resize(t, newTestIssue("Content"), 10)
	ish.SetNotice("Something happened")

	if !strings.Contains(plain(ish), "Something happened") {
		t.Errorf("view doesn't show the notice:\n%s", plain(ish))
	}

	ish.SetNotice("")

	if strings.Contains(plain(ish), "Something happened") {
		t.Errorf("view shows the cleared notice:\n%s", plain(ish))
	}
}
