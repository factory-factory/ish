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

// maxOffset returns the furthest the issue's view can scroll, which puts
// the bottom of the last card at the bottom of the view.
func maxOffset(ish Issue) int {
	return max(ish.viewport.TotalLineCount()-ish.viewport.Height(), 0)
}

var k = tea.KeyPressMsg{Code: 'k', Text: "k"}

func TestArrowsScrollWithoutChangingSelection(t *testing.T) {
	ish := resize(t, newTestIssue(lines(10), lines(1)), 6)

	for _, cursor := range []int{0, 1, ish.firstCard() + 1} {
		ish.setCursor(cursor)
		ish.viewport.SetYOffset(0)

		for i := 1; i <= 3; i++ {
			ish = update(t, ish, down)

			if offset(ish) != i || ish.cursor != cursor {
				t.Fatalf("cursor %d: offset, cursor after %d downs = %d, %d, want %d, %d",
					cursor, i, offset(ish), ish.cursor, i, cursor)
			}
		}

		ish = update(t, ish, up)

		if offset(ish) != 2 || ish.cursor != cursor {
			t.Errorf("cursor %d: offset, cursor after up = %d, %d, want 2, %d", cursor, offset(ish), ish.cursor, cursor)
		}
	}

	// The view stops at either end.
	ish = update(t, ish, up, up, up)
	if offset(ish) != 0 {
		t.Errorf("offset after scrolling past the top = %d, want 0", offset(ish))
	}

	for range maxOffset(ish) + 2 {
		ish = update(t, ish, down)
	}
	if offset(ish) != maxOffset(ish) {
		t.Errorf("offset after scrolling past the end = %d, want %d", offset(ish), maxOffset(ish))
	}
}

// endIssue returns an issue whose last comment can't be scrolled to the top
// of the view, since the view stops at the end of it.
func endIssue(t *testing.T) Issue {
	t.Helper()

	ish := resize(t, newTestIssue(lines(3), lines(3), lines(1)), 6)

	if spans := ish.spans; maxOffset(ish) <= spans[1].top || maxOffset(ish) >= spans[2].top {
		t.Fatalf("max offset %d, want only the last card unable to reach the top (spans %+v)", maxOffset(ish), spans)
	}

	return ish
}

func TestJSelectsNextCommentAtTop(t *testing.T) {
	ish := endIssue(t)
	spans := ish.spans

	// With nothing selected, j selects the first comment.
	for i, want := range []int{spans[0].top, spans[1].top, maxOffset(ish)} {
		ish = update(t, ish, j)

		if got := ish.cardIndex(); got != i {
			t.Fatalf("after %d j: cardIndex() = %d, want %d", i+1, got, i)
		}
		if offset(ish) != want {
			t.Errorf("after %d j: offset = %d, want %d", i+1, offset(ish), want)
		}
	}

	// On the last comment, j does nothing.
	if ish = update(t, ish, j); ish.cardIndex() != 2 || offset(ish) != maxOffset(ish) {
		t.Errorf("j on the last comment: cardIndex(), offset = %d, %d, want 2, %d", ish.cardIndex(), offset(ish), maxOffset(ish))
	}

	// With the last comment already in view, j only moves the selection.
	ish.setCursor(ish.firstCard() + 1)
	ish.viewport.SetYOffset(maxOffset(ish))
	ish = update(t, ish, j)

	if ish.cardIndex() != 2 || offset(ish) != maxOffset(ish) {
		t.Errorf("j with the last comment in view: cardIndex(), offset = %d, %d, want 2, %d", ish.cardIndex(), offset(ish), maxOffset(ish))
	}
}

func TestKSelectsPreviousCommentAtTop(t *testing.T) {
	ish := endIssue(t)
	spans := ish.spans

	// From the title, k selects the last comment.
	ish.setCursor(1)
	ish = update(t, ish, k)

	if ish.cardIndex() != 2 || offset(ish) != maxOffset(ish) {
		t.Fatalf("k from the title: cardIndex(), offset = %d, %d, want 2, %d", ish.cardIndex(), offset(ish), maxOffset(ish))
	}

	for i := 1; i >= 0; i-- {
		ish = update(t, ish, k)

		if ish.cardIndex() != i || offset(ish) != min(spans[i].top, maxOffset(ish)) {
			t.Errorf("k to comment %d: cardIndex(), offset = %d, %d, want %d, %d",
				i, ish.cardIndex(), offset(ish), i, min(spans[i].top, maxOffset(ish)))
		}
	}

	// On the first comment, k does nothing.
	if ish = update(t, ish, k); ish.cardIndex() != 0 || offset(ish) != 0 {
		t.Errorf("k on the first comment: cardIndex(), offset = %d, %d, want 0, 0", ish.cardIndex(), offset(ish))
	}
}

func TestJFromFieldSelectsFirstComment(t *testing.T) {
	ish := resize(t, newTestIssue(lines(3), lines(3)), 6)
	ish.setCursor(2)
	ish = update(t, ish, j)

	if ish.cardIndex() != 0 || offset(ish) != 0 {
		t.Errorf("j from a tag: cardIndex(), offset = %d, %d, want 0, 0", ish.cardIndex(), offset(ish))
	}
}

func TestJKWithoutComments(t *testing.T) {
	ish := resize(t, newTestIssue(), 6)
	ish.setCursor(1)
	ish = update(t, ish, j, k)

	if ish.cursor != 1 {
		t.Errorf("cursor = %d after j and k without comments, want the title still selected", ish.cursor)
	}
}

func TestTabScrollsCommentToTop(t *testing.T) {
	ish := endIssue(t)
	spans := ish.spans

	ish.setCursor(ish.firstCard())
	ish = update(t, ish, tab)

	if ish.cardIndex() != 1 || offset(ish) != spans[1].top {
		t.Errorf("tab: cardIndex(), offset = %d, %d, want 1, %d", ish.cardIndex(), offset(ish), spans[1].top)
	}

	// Near the end, the last comment's bottom stays at the bottom of the
	// view.
	ish = update(t, ish, tab)

	if ish.cardIndex() != 2 || offset(ish) != maxOffset(ish) {
		t.Errorf("tab: cardIndex(), offset = %d, %d, want 2, %d", ish.cardIndex(), offset(ish), maxOffset(ish))
	}

	ish = update(t, ish, shiftTab, shiftTab)

	if ish.cardIndex() != 0 || offset(ish) != spans[0].top {
		t.Errorf("shift+tab: cardIndex(), offset = %d, %d, want 0, %d", ish.cardIndex(), offset(ish), spans[0].top)
	}
}

func TestNavigationHelp(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 6)

	if footer := footerText(ish); !strings.Contains(footer, "j next comment") || !strings.Contains(footer, "k previous comment") {
		t.Errorf("footer %q, want j and k", footer)
	}

	ish = update(t, ish, question)

	if footer := footerText(ish); !strings.Contains(footer, "scroll down") || !strings.Contains(footer, "scroll up") {
		t.Errorf("full help %q, want the arrows' scrolling", footer)
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
