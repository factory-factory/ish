package models

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var (
	lightBackground = tea.BackgroundColorMsg{Color: color.White}
	darkBackground  = tea.BackgroundColorMsg{Color: color.Black}
)

// codeContent is markdown with a code block, whose background shows which
// background a card was rendered for.
const codeContent = "```\ncode\n```"

// renderedFor returns the background card was rendered for, light or dark,
// by its code block's background.
func renderedFor(t *testing.T, card Card) string {
	t.Helper()

	content, _ := card.renderContent()

	switch {
	case strings.Contains(content, codeBackground(false)):
		return "light"
	case strings.Contains(content, codeBackground(true)):
		return "dark"
	}

	t.Fatalf("no code block background in:\n%q", content)
	return ""
}

func TestCardRendersForItsBackground(t *testing.T) {
	content := codeContent
	card := Card{DateCreated: "2023-05-27T13:41:50-0400", Author: "author", Content: &content}

	// Until the terminal says otherwise, it's assumed to be dark.
	if got := renderedFor(t, card); got != "dark" {
		t.Errorf("card rendered for a %s background, want dark", got)
	}

	card.light = true

	if got := renderedFor(t, card); got != "light" {
		t.Errorf("card rendered for a %s background, want light", got)
	}
}

func TestIssuePassesBackgroundToCards(t *testing.T) {
	ish := resize(t, newTestIssue(codeContent, codeContent), 20)
	ish = update(t, ish, lightBackground)

	for i, card := range ish.Conversation {
		if got := renderedFor(t, card); got != "light" {
			t.Errorf("card %d rendered for a %s background, want light", i, got)
		}
	}

	// The viewport shows the cards as they're rendered now.
	if !strings.Contains(ish.viewport.View(), codeBackground(false)) {
		t.Error("the viewport doesn't show the cards rendered for a light background")
	}

	ish = update(t, ish, darkBackground)

	if got := renderedFor(t, ish.Conversation[0]); got != "dark" {
		t.Errorf("card rendered for a %s background after it changed, want dark", got)
	}
}

func TestNewCommentsUseBackground(t *testing.T) {
	ish := resize(t, newTestIssue("Content"), 20)
	ish.SetAuthor("author")
	ish = update(t, ish, lightBackground)
	ish.addComment()

	if card := ish.Conversation[len(ish.Conversation)-1]; !card.light {
		t.Error("a new comment isn't rendered for the light background")
	}
}

func TestFollowKeepsBackground(t *testing.T) {
	old := resize(t, newTestIssue(codeContent), 20)
	old = update(t, old, lightBackground)

	fresh := newTestIssue(codeContent)
	fresh.Follow(old)

	if got := renderedFor(t, fresh.Conversation[0]); got != "light" {
		t.Errorf("reloaded card rendered for a %s background, want light", got)
	}
}
