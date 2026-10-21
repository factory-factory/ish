package models

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/factory-factory/ish/styles"
)

// people are the users tests show by name. Sam's name is two users', so it's
// ambiguous, and Alice has two names, so the first is shown.
var people = NewDirectory([]User{
	{"Alice", "alice@example.com"},
	{"Alice Lee", "alice.lee@example.com"},
	{"Alicia", "alice@example.com"},
	{"Bob", "bob@example.com"},
	{"Sam", "sam@one.example"},
	{"Sam", "sam@two.example"},
})

func TestDirectory(t *testing.T) {
	for _, tt := range []struct {
		email, name string
		ok          bool
	}{
		{"alice@example.com", "Alice", true},
		{"bob@example.com", "Bob", true},
		{"nobody@example.com", "", false},
	} {
		if name, ok := people.Name(tt.email); name != tt.name || ok != tt.ok {
			t.Errorf("Name(%q) = %q, %v, want %q, %v", tt.email, name, ok, tt.name, tt.ok)
		}
	}

	for _, tt := range []struct {
		name, email string
		ok          bool
	}{
		{"Bob", "bob@example.com", true},
		{"Alicia", "alice@example.com", true},
		{"Sam", "", false}, // Ambiguous.
		{"Nobody", "", false},
	} {
		if email, ok := people.Email(tt.name); email != tt.email || ok != tt.ok {
			t.Errorf("Email(%q) = %q, %v, want %q, %v", tt.name, email, ok, tt.email, tt.ok)
		}
	}

	var none *Directory
	if _, ok := none.Name("alice@example.com"); ok {
		t.Error("a nil directory has a name")
	}

	for value, want := range map[string]string{
		"<alice@example.com>":  "@Alice",
		"<nobody@example.com>": "nobody@example.com",
		"alice@example.com":    "alice@example.com",
		"open":                 "open",
	} {
		if got := people.Show(value); got != want {
			t.Errorf("Show(%q) = %q, want %q", value, got, want)
		}
	}
}

// peopleIssue returns an issue that shows people's names, with the given
// metadata and comments.
func peopleIssue(t *testing.T, tags []Tag, comments ...string) Issue {
	t.Helper()

	ish := newTestIssue(comments...)
	ish.Author = "alice@example.com"
	ish.Metadata = append(ish.Metadata, tags...)
	ish.SetDirectory(people)

	return resize(t, ish, 30)
}

func TestTagShowsNames(t *testing.T) {
	ish := peopleIssue(t, []Tag{{
		Key:    "assignees",
		Values: Tokens("<bob@example.com>", "<nobody@example.com>", "alice@example.com"),
	}})

	tag := ish.Metadata[1]

	if want := []Token{
		{"Bob", "<bob@example.com>"},
		{"", "<nobody@example.com>"},
		{"", "alice@example.com"},
	}; !slices.Equal(tag.Values, want) {
		t.Errorf("values = %q, want %q", tag.Values, want)
	}

	// A user's email in angle brackets is shown as their name, in aqua. An
	// unknown one is shown as an ordinary email, and one without brackets
	// as it's written.
	view := tag.View().Content

	if !strings.Contains(view, styles.Mention.Render("@Bob")) {
		t.Errorf("tag %q, want Bob's name", view)
	}
	if plain := escapes.ReplaceAllString(view, ""); !strings.Contains(plain, "@Bob, nobody@example.com, alice@example.com") {
		t.Errorf("tag %q, want each value shown", plain)
	}

	// The file and the database get the canonical values.
	if got := ish.tagValues("assignees"); got != "<bob@example.com>, <nobody@example.com>, alice@example.com" {
		t.Errorf("tagValues() = %q, want the emails", got)
	}
}

func TestTagEditsNames(t *testing.T) {
	ish := peopleIssue(t, []Tag{{Key: "assignees", Values: Tokens("<bob@example.com>", "carol")}})
	ish.setCursor(3)
	ish = update(t, ish, enter)

	// Names are edited in place of emails.
	if got := ish.Metadata[1].input.Value(); got != "@Bob, carol" {
		t.Fatalf("editing %q, want the name in place of the email", got)
	}

	ish = update(t, ish, ctrlU)
	ish = typeText(t, ish, "@Alicia, @Sam, @Nobody, <bob@example.com>, @Bob")
	ish = update(t, ish, enter)

	// A single user's name is saved as their email. An ambiguous or unknown
	// one is kept as it's typed.
	want := []Token{
		{"Alice", "<alice@example.com>"},
		{"", "@Sam"},
		{"", "@Nobody"},
		{"Bob", "<bob@example.com>"},
		{"Bob", "<bob@example.com>"},
	}

	if got := ish.Metadata[1].Values; !slices.Equal(got, want) {
		t.Errorf("values = %q, want %q", got, want)
	}
}

func TestEditingWithoutChangeKeepsTag(t *testing.T) {
	ish := peopleIssue(t, []Tag{{Key: "assignees", Values: Tokens("<bob@example.com>")}})
	ish.setCursor(3)
	ish = update(t, ish, enter, enter)

	if ish.dirty {
		t.Error("saving a name unchanged made the issue dirty")
	}
}

func TestBylinesShowNames(t *testing.T) {
	ish := peopleIssue(t, nil, "Hello")
	ish.Conversation[0].Author = "bob@example.com"
	ish.Conversation = append(ish.Conversation, Card{
		DateCreated: "2023-05-27T13:41:50-0400",
		Author:      "Claude",
		Content:     ptr("Hi"),
		dir:         people,
	})

	if header := ish.renderHeader(); !strings.Contains(header, styles.BylineEmphasis.Render("@Alice")) {
		t.Errorf("header %q, want the issue's author by name", header)
	}
	if byline := ish.Conversation[0].renderByline(); !strings.Contains(byline, styles.BylineEmphasis.Render("@Bob")) {
		t.Errorf("byline %q, want the comment's author by name", byline)
	}
	if byline := ish.Conversation[1].renderByline(); !strings.Contains(byline, styles.BylineEmphasis.Render("Claude")) {
		t.Errorf("byline %q, want an author who isn't a user as they are", byline)
	}
}

func TestBylineAuthors(t *testing.T) {
	for author, want := range map[string]string{
		"bob@example.com":      "@Bob",
		"<bob@example.com>":    "@Bob",
		"<nobody@example.com>": "nobody@example.com",
		"nobody@example.com":   "nobody@example.com",
		"Claude":               "Claude",
	} {
		if got := people.author(author); got != want {
			t.Errorf("author(%q) = %q, want %q", author, got, want)
		}
	}

	// The issue's and comments' bylines alike.
	ish := peopleIssue(t, nil, "Hello")
	ish.Author = "<alice@example.com>"
	ish.Conversation[0].Author = "<bob@example.com>"

	if header := ish.renderHeader(); !strings.Contains(header, styles.BylineEmphasis.Render("@Alice")) {
		t.Errorf("header %q, want the issue's bracketed author by name", header)
	}
	if byline := ish.Conversation[0].renderByline(); !strings.Contains(byline, styles.BylineEmphasis.Render("@Bob")) {
		t.Errorf("byline %q, want the comment's bracketed author by name", byline)
	}
}

func ptr(s string) *string { return &s }

func TestCommentShowsNames(t *testing.T) {
	content := "Ping <bob@example.com>, not `<bob@example.com>` or <nobody@example.com>.\n\n```\n<bob@example.com>\n```"
	card := Card{DateCreated: "2023-05-27T13:41:50-0400", Author: "a", Content: &content, dir: people}

	rendered, _ := card.renderContent()
	plain := escapes.ReplaceAllString(rendered, "")

	if !strings.Contains(rendered, styles.Mention.Render("@Bob")) {
		t.Errorf("rendered %q, want Bob's name in aqua", rendered)
	}
	if strings.Count(plain, "@Bob") != 1 || strings.Count(plain, "bob@example.com") != 2 {
		t.Errorf("rendered %q, want only the email outside code replaced", plain)
	}
	if !strings.Contains(plain, "nobody@example.com") {
		t.Errorf("rendered %q, want an unknown email as it is", plain)
	}

	for _, marker := range []string{mentionMarker, mentionEndMarker} {
		if strings.Contains(rendered, marker) {
			t.Errorf("rendered content contains marker %q", marker)
		}
	}
}

func TestShowAndSaveNames(t *testing.T) {
	// Content as it's written, and as it's edited.
	for _, tt := range []struct {
		saved, edited string
	}{
		{"Ping <bob@example.com>.", "Ping @Bob."},
		{"<alice@example.com> and <alice.lee@example.com>", "@Alice and @Alice Lee"},
		{"`<bob@example.com>` stays", "`<bob@example.com>` stays"},
		{"<nobody@example.com> stays", "<nobody@example.com> stays"},
		{"<sam@one.example> stays, as Sam is ambiguous", "<sam@one.example> stays, as Sam is ambiguous"},
	} {
		if got := people.showNames(tt.saved); got != tt.edited {
			t.Errorf("showNames(%q) = %q, want %q", tt.saved, got, tt.edited)
		}
		if got := people.saveEmails(tt.edited); got != tt.saved {
			t.Errorf("saveEmails(%q) = %q, want %q", tt.edited, got, tt.saved)
		}
	}

	// Typed names that aren't a single user's, or aren't names, are kept.
	for _, text := range []string{
		"@Sam is ambiguous",
		"@Nobody",
		"email bob@Bob.example",
		"@Bobby isn't Bob",
		"`@Bob` is code",
		"@",
	} {
		if got := people.saveEmails(text); got != text {
			t.Errorf("saveEmails(%q) = %q, want it unchanged", text, got)
		}
	}
}

func TestCardEditsNames(t *testing.T) {
	ish := peopleIssue(t, nil, "Ping <bob@example.com>")
	ish.setCursor(ish.firstCard())
	ish = update(t, ish, enter)

	if got := ish.Conversation[0].editor.Value(); got != "Ping @Bob" {
		t.Fatalf("editing %q, want Bob's name", got)
	}

	// Saving without a change leaves the comment as it was.
	ish = update(t, ish, ctrlS)

	if ish.dirty || *ish.Conversation[0].Content != "Ping <bob@example.com>" {
		t.Errorf("dirty, content = %v, %q, want the comment unchanged", ish.dirty, *ish.Conversation[0].Content)
	}

	// A typed name is saved as the email.
	ish = update(t, ish, enter, tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl})
	ish = typeText(t, ish, " and @Alice Lee")
	ish = update(t, ish, ctrlS)

	if got := *ish.Conversation[0].Content; got != "Ping <bob@example.com> and <alice.lee@example.com>" {
		t.Errorf("content = %q, want the names saved as emails", got)
	}
}

func TestNewCommentsAndTagsGetDirectory(t *testing.T) {
	ish := peopleIssue(t, nil)
	ish.SetAuthor("bob@example.com")
	ish = update(t, ish, cKey)

	if ish.Conversation[len(ish.Conversation)-1].dir != people {
		t.Error("a new comment doesn't have the directory")
	}

	ish = update(t, ish, ctrlS, mKey)

	if ish.Metadata[len(ish.Metadata)-1].dir != people {
		t.Error("a new tag doesn't have the directory")
	}
}
