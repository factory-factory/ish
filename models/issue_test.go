package models

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// newTestIssue returns an issue with one card per content string.
func newTestIssue(contents ...string) Issue {
	ish := Issue{
		Id:          "00000",
		DateCreated: "2023-05-26T23:35:36-0400",
		Author:      "author",
		Title:       "Title",
		Metadata: []Tag{
			{Key: "state", Values: []string{"open"}},
		},
	}

	for _, content := range contents {
		ish.Conversation = append(ish.Conversation, Card{
			DateCreated: "2023-05-27T13:41:50-0400",
			Author:      "author",
			Content:     &content,
		})
	}

	return ish
}

// lines returns markdown content made of n separate paragraphs.
func lines(n int) string {
	paragraphs := make([]string, n)
	for i := range paragraphs {
		paragraphs[i] = "line"
	}
	return strings.Join(paragraphs, "\n\n")
}

func update(t *testing.T, ish Issue, msgs ...tea.Msg) Issue {
	t.Helper()

	for _, msg := range msgs {
		m, _ := ish.Update(msg)
		ish = m.(Issue)
	}

	return ish
}

// resize sends a window size message giving the card area the given height.
func resize(t *testing.T, ish Issue, area int) Issue {
	t.Helper()

	header := lipgloss.Height(ish.renderHeader())
	return update(t, ish, tea.WindowSizeMsg{Width: 120, Height: header + area})
}

var (
	down = tea.KeyPressMsg{Code: tea.KeyDown}
	up   = tea.KeyPressMsg{Code: tea.KeyUp}
	tab  = tea.KeyPressMsg{Code: tea.KeyTab}
)

func TestViewPinsHeader(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3), lines(3)), 6)
	ish.offset = 4

	header := strings.Split(ish.renderHeader(), "\n")
	view := strings.Split(ish.View().Content, "\n")

	if len(view) > ish.height {
		t.Errorf("View() has %d lines, want at most %d", len(view), ish.height)
	}

	for i, want := range header {
		got := strings.TrimRight(view[i], " ")
		if want = strings.TrimRight(want, " "); got != want {
			t.Errorf("View() line %d = %q, want header line %q", i, got, want)
		}
	}
}

func TestViewShowsScrolledCards(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3)), 5)
	ish.offset = 2

	cards, _ := ish.renderCards()
	want := strings.Split(cards, "\n")[2:7]

	view := strings.Split(ish.View().Content, "\n")
	got := view[len(view)-5:]

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("card area line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDownSelectsFirstCard(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1), lines(1)), 40)
	ish = update(t, ish, down)

	if got := ish.cardIndex(); got != 0 {
		t.Errorf("cardIndex() = %d, want 0", got)
	}
	if !ish.Conversation[0].selected {
		t.Error("first card is not selected")
	}
}

func TestDownMovesBetweenCardsThatFit(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1), lines(1), lines(1)), 40)
	ish = update(t, ish, down, down, down)

	if got := ish.cardIndex(); got != 2 {
		t.Errorf("cardIndex() = %d, want 2", got)
	}
	if ish.offset != 0 {
		t.Errorf("offset = %d, want 0", ish.offset)
	}
	if ish.Conversation[0].selected || ish.Conversation[1].selected {
		t.Error("previously selected cards are still selected")
	}

	// Down on the last card, fully in view, does nothing.
	ish = update(t, ish, down)
	if got := ish.cardIndex(); got != 2 {
		t.Errorf("cardIndex() after extra down = %d, want 2", got)
	}
}

func TestDownScrollsThroughTallCard(t *testing.T) {
	ish := resize(t, newTestIssue(lines(10), lines(1)), 6)
	spans, area := ish.layout()
	hidden := spans[0].bottom() - area

	if hidden <= 0 {
		t.Fatalf("first card has height %d, want taller than area %d", spans[0].height, area)
	}

	ish = update(t, ish, down)

	for i := 1; i <= hidden; i++ {
		ish = update(t, ish, down)

		if ish.offset != i {
			t.Fatalf("offset after %d scrolls = %d, want %d", i, ish.offset, i)
		}
		if got := ish.cardIndex(); got != 0 {
			t.Fatalf("cardIndex() after %d scrolls = %d, want 0", i, got)
		}
	}

	ish = update(t, ish, down)

	if got := ish.cardIndex(); got != 1 {
		t.Errorf("cardIndex() = %d, want 1", got)
	}
	if s := spans[1]; s.bottom() > ish.offset+area || s.top < ish.offset {
		t.Errorf("second card %+v not in view at offset %d with area %d", s, ish.offset, area)
	}
}

func TestUpRevealsBottomOfTallCard(t *testing.T) {
	ish := resize(t, newTestIssue(lines(10), lines(1)), 6)
	spans, area := ish.layout()

	ish.setCursor(ish.firstCard() + 1)
	ish.offset = spans[1].bottom() - area
	ish = update(t, ish, up)

	if got := ish.cardIndex(); got != 0 {
		t.Fatalf("cardIndex() = %d, want 0", got)
	}

	if s := spans[0]; s.bottom() <= ish.offset || s.bottom() > ish.offset+area {
		t.Errorf("bottom of card %+v not in view at offset %d with area %d", s, ish.offset, area)
	}

	for ish.offset > 0 {
		before := ish.offset
		ish = update(t, ish, up)

		if ish.offset != before-1 {
			t.Fatalf("offset = %d, want %d", ish.offset, before-1)
		}
	}

	// Up on the first card, fully scrolled to its top, does nothing.
	ish = update(t, ish, up)
	if got := ish.cardIndex(); got != 0 || ish.offset != 0 {
		t.Errorf("cardIndex(), offset = %d, %d, want 0, 0", got, ish.offset)
	}
}

func TestTabRevealsSelectedCard(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3), lines(3)), 6)
	spans, area := ish.layout()

	ish.setCursor(ish.firstCard() + 1)
	ish = update(t, ish, tab)

	if got := ish.cardIndex(); got != 2 {
		t.Fatalf("cardIndex() = %d, want 2", got)
	}
	if s := spans[2]; s.top < ish.offset || s.top >= ish.offset+area {
		t.Errorf("top of card %+v not in view at offset %d with area %d", s, ish.offset, area)
	}
}

func TestTabWrapsAround(t *testing.T) {
	ish := newTestIssue(lines(1))
	total := len(ish.Metadata) + len(ish.Conversation) + 2

	ish = update(t, ish, tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if ish.cursor != total-1 {
		t.Errorf("cursor after shift+tab = %d, want %d", ish.cursor, total-1)
	}

	ish = update(t, ish, tab)
	if ish.cursor != 0 {
		t.Errorf("cursor after tab = %d, want 0", ish.cursor)
	}
}

func TestResizeClampsOffset(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3)), 4)
	ish.offset = 1000
	ish = resize(t, ish, 4)

	spans, area := ish.layout()
	if want := spans[len(spans)-1].bottom() - area; ish.offset != want {
		t.Errorf("offset = %d, want %d", ish.offset, want)
	}

	ish = resize(t, ish, 1000)
	if ish.offset != 0 {
		t.Errorf("offset after growing = %d, want 0", ish.offset)
	}
}
