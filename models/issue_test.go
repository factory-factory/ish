package models

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
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

// resize sends a window size message giving the viewport the given height.
func resize(t *testing.T, ish Issue, area int) Issue {
	t.Helper()

	ish.initComponents()
	header := lipgloss.Height(ish.renderHeader())
	footer := lipgloss.Height(ish.renderFooter())
	ish = update(t, ish, tea.WindowSizeMsg{Width: 120, Height: header + area + footer})

	if got := ish.viewport.Height(); got != area {
		t.Fatalf("viewport height = %d, want %d", got, area)
	}

	return ish
}

// offset returns the viewport's scroll position.
func offset(ish Issue) int {
	return ish.viewport.YOffset()
}

var (
	down     = tea.KeyPressMsg{Code: tea.KeyDown}
	up       = tea.KeyPressMsg{Code: tea.KeyUp}
	tab      = tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	pageDown = tea.KeyPressMsg{Code: tea.KeyPgDown}
	question = tea.KeyPressMsg{Code: '?', Text: "?"}
	esc      = tea.KeyPressMsg{Code: tea.KeyEscape}
	w        = tea.KeyPressMsg{Code: 'w', Text: "w"}
)

func TestViewPinsHeaderAndFooter(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3), lines(3)), 6)
	ish.viewport.SetYOffset(4)

	header := strings.Split(ish.renderHeader(), "\n")
	footer := strings.Split(ish.renderFooter(), "\n")
	view := strings.Split(ish.View().Content, "\n")

	if len(view) != ish.height {
		t.Fatalf("View() has %d lines, want %d", len(view), ish.height)
	}

	trim := func(s string) string { return strings.TrimRight(s, " ") }

	for i, want := range header {
		if got := trim(view[i]); got != trim(want) {
			t.Errorf("View() line %d = %q, want header line %q", i, got, trim(want))
		}
	}

	for i, want := range footer {
		j := len(view) - len(footer) + i
		if got := trim(view[j]); got != trim(want) {
			t.Errorf("View() line %d = %q, want footer line %q", j, got, trim(want))
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
	if offset(ish) != 0 {
		t.Errorf("offset = %d, want 0", offset(ish))
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
	spans, area := ish.spans, ish.viewport.Height()
	hidden := spans[0].bottom() - area

	if hidden <= 0 {
		t.Fatalf("first card has height %d, want taller than area %d", spans[0].height, area)
	}

	ish = update(t, ish, down)

	for i := 1; i <= hidden; i++ {
		ish = update(t, ish, down)

		if offset(ish) != i {
			t.Fatalf("offset after %d scrolls = %d, want %d", i, offset(ish), i)
		}
		if got := ish.cardIndex(); got != 0 {
			t.Fatalf("cardIndex() after %d scrolls = %d, want 0", i, got)
		}
	}

	ish = update(t, ish, down)

	if got := ish.cardIndex(); got != 1 {
		t.Errorf("cardIndex() = %d, want 1", got)
	}
	if s := spans[1]; s.bottom() > offset(ish)+area || s.top < offset(ish) {
		t.Errorf("second card %+v not in view at offset %d with area %d", s, offset(ish), area)
	}
}

func TestUpRevealsBottomOfTallCard(t *testing.T) {
	ish := resize(t, newTestIssue(lines(10), lines(1)), 6)
	spans, area := ish.spans, ish.viewport.Height()

	ish.setCursor(ish.firstCard() + 1)
	ish.viewport.SetYOffset(spans[1].bottom() - area)
	ish = update(t, ish, up)

	if got := ish.cardIndex(); got != 0 {
		t.Fatalf("cardIndex() = %d, want 0", got)
	}
	if s := spans[0]; s.bottom() <= offset(ish) || s.bottom() > offset(ish)+area {
		t.Errorf("bottom of card %+v not in view at offset %d with area %d", s, offset(ish), area)
	}

	for offset(ish) > 0 {
		before := offset(ish)
		ish = update(t, ish, up)

		if offset(ish) != before-1 {
			t.Fatalf("offset = %d, want %d", offset(ish), before-1)
		}
	}

	// Up on the first card, fully scrolled to its top, does nothing.
	ish = update(t, ish, up)
	if got := ish.cardIndex(); got != 0 || offset(ish) != 0 {
		t.Errorf("cardIndex(), offset = %d, %d, want 0, 0", got, offset(ish))
	}
}

func TestTabRevealsSelectedCard(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3), lines(3)), 6)
	spans, area := ish.spans, ish.viewport.Height()

	ish.setCursor(ish.firstCard() + 1)
	ish = update(t, ish, tab)

	if got := ish.cardIndex(); got != 2 {
		t.Fatalf("cardIndex() = %d, want 2", got)
	}
	if s := spans[2]; s.top < offset(ish) || s.top >= offset(ish)+area {
		t.Errorf("top of card %+v not in view at offset %d with area %d", s, offset(ish), area)
	}
}

func TestTabWrapsAround(t *testing.T) {
	ish := newTestIssue(lines(1))
	total := len(ish.Metadata) + len(ish.Conversation) + 2

	ish = update(t, ish, shiftTab)
	if ish.cursor != total-1 {
		t.Errorf("cursor after shift+tab = %d, want %d", ish.cursor, total-1)
	}

	ish = update(t, ish, tab)
	if ish.cursor != 0 {
		t.Errorf("cursor after tab = %d, want 0", ish.cursor)
	}
}

func TestPageDownScrollsViewport(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3), lines(3)), 6)
	ish = update(t, ish, pageDown)

	if offset(ish) != 6 {
		t.Errorf("offset = %d, want 6", offset(ish))
	}
}

func TestHelpToggleResizesViewport(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3)), 10)
	short := lipgloss.Height(ish.renderFooter())

	ish = update(t, ish, question)

	if !ish.help.ShowAll {
		t.Fatal("full help not shown after ?")
	}

	full := lipgloss.Height(ish.renderFooter())
	if want := 10 - (full - short); ish.viewport.Height() != want {
		t.Errorf("viewport height = %d, want %d", ish.viewport.Height(), want)
	}
}

func TestResizeClampsOffset(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3)), 4)
	ish.viewport.SetYOffset(1000)

	if want := ish.spans[len(ish.spans)-1].bottom() - 4; offset(ish) != want {
		t.Errorf("offset = %d, want %d", offset(ish), want)
	}

	ish = resize(t, ish, 1000)
	if offset(ish) != 0 {
		t.Errorf("offset after growing = %d, want 0", offset(ish))
	}
}

func TestHeaderRuleSeparatesCards(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)

	view := strings.Split(ish.View().Content, "\n")
	headerHeight := lipgloss.Height(ish.renderHeader())

	rule := -1
	for i, line := range view[:headerHeight] {
		if strings.Contains(line, "─") {
			rule = i
		}
	}

	if rule < 0 {
		t.Fatal("no horizontal rule in the header")
	}

	got := strings.Count(view[rule], "─")
	if want := ish.width - styles.Rule.GetHorizontalMargins(); got != want {
		t.Errorf("rule is %d cells wide, want %d", got, want)
	}

	// The first card begins below the rule, at the top of the viewport.
	cards, _ := ish.renderCards()
	first := strings.TrimRight(strings.Split(cards, "\n")[0], " ")
	if got := strings.TrimRight(view[headerHeight], " "); got != first {
		t.Errorf("first viewport line = %q, want first card line %q", got, first)
	}
	if rule >= headerHeight {
		t.Errorf("rule on line %d, want above the viewport at line %d", rule, headerHeight)
	}
}

func TestState(t *testing.T) {
	ish := newTestIssue()

	if got := ish.State(); got != "open" {
		t.Errorf("State() = %q, want %q", got, "open")
	}

	ish.Metadata = []Tag{
		{Key: "assignees", Values: []string{"alice"}},
		{Key: "state", Values: []string{"in progress", "blocked"}},
	}
	if got := ish.State(); got != "in progress, blocked" {
		t.Errorf("State() = %q, want %q", got, "in progress, blocked")
	}

	ish.Metadata = nil
	if got := ish.State(); got != "" {
		t.Errorf("State() with no state tag = %q, want empty", got)
	}
}

func TestBackKeyDisabledByDefault(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)

	for _, msg := range []tea.KeyPressMsg{esc, w} {
		if _, cmd := ish.Update(msg); cmd != nil {
			t.Errorf("%s on an issue that can't be closed returned a command", msg)
		}
	}

	if footer := ish.renderFooter(); strings.Contains(footer, "back") {
		t.Errorf("footer %q lists the back key on an issue that can't be closed", footer)
	}
}

func TestBackKeyClosesClosableIssue(t *testing.T) {
	ish := newTestIssue(lines(1))
	ish.SetClosable(true)
	ish = resize(t, ish, 10)

	for _, msg := range []tea.KeyPressMsg{esc, w} {
		_, cmd := ish.Update(msg)

		if cmd == nil {
			t.Fatalf("%s returned no command, want one sending CloseMsg", msg)
		}
		if got, ok := cmd().(CloseMsg); !ok {
			t.Errorf("%s sent %T, want CloseMsg", msg, got)
		}
	}

	if footer := ish.renderFooter(); !strings.Contains(footer, "esc/w") {
		t.Errorf("footer %q doesn't list the back key", footer)
	}
}

func TestNewIssue(t *testing.T) {
	created := time.Date(2026, 10, 10, 23, 7, 0, 0, time.UTC)
	ish := NewIssue("ABCDEFGHJK", "alice@example.com", created)

	if ish.Id != "ABCDEFGHJK" || ish.Title != "" || ish.Author != "alice@example.com" {
		t.Errorf("Id, Title, Author = %q, %q, %q, want ABCDEFGHJK, empty, alice@example.com",
			ish.Id, ish.Title, ish.Author)
	}
	if ish.DateCreated != "2026-10-10T23:07:00Z" {
		t.Errorf("DateCreated = %q, want %q", ish.DateCreated, "2026-10-10T23:07:00Z")
	}
	if len(ish.Metadata) != 0 {
		t.Errorf("Metadata = %v, want none", ish.Metadata)
	}
	if len(ish.Conversation) != 1 {
		t.Fatalf("Conversation has %d cards, want 1", len(ish.Conversation))
	}

	card := ish.Conversation[0]

	if card.Author != ish.Author || card.DateCreated != ish.DateCreated || card.Content == nil || *card.Content != "" {
		t.Errorf("card = %+v, want an empty comment with the issue's byline", card)
	}

	// The new issue renders.
	ish = resize(t, ish, 10)
	if view := ish.View().Content; !strings.Contains(view, "[ABCDEFGHJK]") {
		t.Errorf("View() = %q, want the issue ID", view)
	}
}

func TestAssignees(t *testing.T) {
	tests := []struct {
		name     string
		metadata []Tag
		want     string
	}{
		{"assignees", []Tag{{Key: "assignees", Values: []string{"alice", "bob"}}}, "alice, bob"},
		{"among other tags", []Tag{
			{Key: "state", Values: []string{"open"}},
			{Key: "assignees", Values: []string{"bob"}},
		}, "bob"},
		// The key is standardized on assignees.
		{"singular key", []Tag{{Key: "assignee", Values: []string{"alice"}}}, ""},
		{"none", []Tag{{Key: "state", Values: []string{"open"}}}, ""},
		{"no metadata", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ish := Issue{Metadata: tt.metadata}

			if got := ish.Assignees(); got != tt.want {
				t.Errorf("Assignees() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	want := time.Date(2023, 5, 28, 15, 55, 23, 0, time.UTC)

	for _, s := range []string{
		"2023-05-28T11:55:23-04:00", // RFC 3339
		"2023-05-28 11:55:23-0400",  // ISO8601A
		"2023-05-28T11:55:23-0400",  // ISO8601B
		"2023-05-28 11:55:23 -0400", // Timestamp
	} {
		got, err := ParseDate(s)

		if err != nil {
			t.Errorf("ParseDate(%q) error = %v", s, err)
			continue
		}
		if !got.Equal(want) {
			t.Errorf("ParseDate(%q) = %v, want %v", s, got, want)
		}
	}

	if _, err := ParseDate("yesterday"); err == nil {
		t.Error(`ParseDate("yesterday") error = nil, want an error`)
	}
}

func TestOpened(t *testing.T) {
	ish := Issue{DateCreated: "2026-10-11T08:59:23-07:00"}

	got, err := ish.Opened()

	if err != nil {
		t.Fatalf("Opened() error = %v", err)
	}

	// The byline's time zone is kept.
	if got.Format(time.RFC3339) != "2026-10-11T08:59:23-07:00" {
		t.Errorf("Opened() = %v, want 2026-10-11T08:59:23-07:00", got)
	}
}
