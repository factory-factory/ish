package models

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/styles"
)

const linkContent = "Blocked by #AAAAA-11111.\n\n- [ ] first\n- [ ] after #BBBBB-22222\n\nNot `#AAAAA-11111` or #CCCCC-33333."

func linkCard(content string) Card {
	return Card{
		DateCreated: "2023-05-27T13:41:50-0400",
		Author:      "author",
		Content:     &content,
		selected:    true,
		issues:      knownIssues,
	}
}

func TestCardItemsInReadingOrder(t *testing.T) {
	card := linkCard(linkContent)

	var got []string

	for _, it := range card.items() {
		if it.todo >= 0 {
			got = append(got, "todo "+strings.Split(*card.Content, "\n")[it.todo])
		} else {
			got = append(got, "link "+it.link.id)
		}
	}

	want := []string{"link AAAAA-11111", "todo - [ ] first", "todo - [ ] after #BBBBB-22222", "link BBBBB-22222"}

	if strings.Join(got, "; ") != strings.Join(want, "; ") {
		t.Errorf("items() = %q, want %q", got, want)
	}
}

func TestRenderContentStylesLinks(t *testing.T) {
	card := linkCard(linkContent)
	rendered, row := card.renderContent()

	if row != -1 {
		t.Errorf("selected row = %d without selection, want -1", row)
	}

	for _, link := range []string{"#AAAAA-11111", "#BBBBB-22222"} {
		if !strings.Contains(rendered, styles.Link.Render(link)) {
			t.Errorf("rendered content doesn't style %s as a link:\n%q", link, rendered)
		}
	}

	// Code and unknown issues aren't links.
	for _, text := range []string{"#CCCCC-33333"} {
		if strings.Contains(rendered, styles.Link.Render(text)) {
			t.Errorf("%s is styled as a link", text)
		}
	}
	if strings.Count(rendered, styles.Link.Render("#AAAAA-11111")) != 1 {
		t.Error("the reference in a code span is styled as a link")
	}

	for _, marker := range []string{linkMarker, selectedLinkMarker, linkEndMarker} {
		if strings.Contains(rendered, marker) {
			t.Errorf("rendered content contains marker %q", marker)
		}
	}

	// Removing the styling leaves the text as glamour rendered it.
	if text := escapes.ReplaceAllString(rendered, ""); !strings.Contains(text, "Blocked by #AAAAA-11111.") {
		t.Errorf("rendered text = %q, want the reference in its sentence", text)
	}
}

func TestRenderContentHighlightsSelectedLink(t *testing.T) {
	card := linkCard(linkContent)
	card.startItems()
	card.moveItem(3)

	rendered, row := card.renderContent()
	lines := strings.Split(rendered, "\n")

	if row < 0 || !strings.Contains(lines[row], styles.LinkSelected.Render("#BBBBB-22222")) {
		t.Fatalf("row %d of %q, want the selected link", row, rendered)
	}
	if !strings.Contains(rendered, styles.Link.Render("#AAAAA-11111")) {
		t.Error("the other link isn't styled as a link")
	}

	view := strings.Split(card.View().Content, "\n")
	if got := escapes.ReplaceAllString(view[card.itemRow()], ""); !strings.Contains(got, "#BBBBB-22222") {
		t.Errorf("view line %d = %q, want the selected link", card.itemRow(), got)
	}
}

func TestSelectedTodoKeepsLinkVisible(t *testing.T) {
	card := linkCard(linkContent)
	card.startItems()
	card.moveItem(2)

	rendered, row := card.renderContent()
	line := strings.Split(rendered, "\n")[row]

	if !strings.Contains(line, todoLinkStyle.Render("#BBBBB-22222")) {
		t.Errorf("selected todo line %q, want its link underlined in the todo's colors", line)
	}
	if !strings.Contains(line, styles.TodoSelected.Render("[ ] after ")) {
		t.Errorf("selected todo line %q, want its text highlighted", line)
	}
}

func TestStyleLinksRestoresSurroundingStyle(t *testing.T) {
	bold := "\x1b[1m"
	rendered := bold + "see " + linkMarker + "#ABC" + linkEndMarker + " here\x1b[0m"

	got, _ := styleLinks(rendered)
	want := bold + "see " + styles.Link.Render("#ABC") + bold + " here\x1b[0m"

	if got != want {
		t.Errorf("styleLinks() = %q, want %q", got, want)
	}

	if got := activeStyle("\x1b[1m\x1b[0m\x1b[3mx\x1b[4m"); got != "\x1b[3m\x1b[4m" {
		t.Errorf("activeStyle() = %q, want the styles since the reset", got)
	}
}

func TestMarkItemsAtSameOffset(t *testing.T) {
	card := linkCard("- [ ] #ABC")
	card.startItems()

	if got := card.markItems(); got != "- [ ] "+todoMarker+linkMarker+"#ABC"+linkEndMarker {
		t.Errorf("markItems() = %q, want the todo marker before the link", got)
	}

	card.moveItem(1)

	if got := card.markItems(); got != "- [ ] "+selectedLinkMarker+"#ABC"+linkEndMarker {
		t.Errorf("markItems() = %q, want the link selected", got)
	}
}

// linkIssue returns an issue whose metadata and second card link to other
// issues, with lookup set after the cards are created.
func linkIssue(t *testing.T, s *saver) Issue {
	t.Helper()

	ish := newTestIssue("No links here", linkContent)
	ish.Metadata = append(ish.Metadata,
		Tag{Key: "relation.blocked-by", Values: Tokens("#CCCCC-33333", "#AAAAA-11111", "see #BBBBB-22222")},
	)
	ish.SetClosable(true)
	ish.SetIssueLookup(knownIssues)

	if s != nil {
		ish.SetSave(s.save)
	}

	return resize(t, ish, 30)
}

// openedId sends msgs, and returns the ID in the OpenMsg produced by the
// last one's command, or "" if it produced none.
func openedId(t *testing.T, ish Issue, msgs ...tea.Msg) (Issue, string) {
	t.Helper()

	ish = update(t, ish, msgs[:len(msgs)-1]...)
	ish, msg := exitWith(t, ish, msgs[len(msgs)-1])

	if open, ok := msg.(OpenMsg); ok {
		return ish, open.Id
	}

	return ish, ""
}

func TestEnterOnCardLinkOpensIssue(t *testing.T) {
	ish := linkIssue(t, nil)
	ish.setCursor(ish.firstCard() + 1)

	if footer := footerText(ish); !strings.Contains(footer, "→/l select") {
		t.Errorf("footer %q doesn't list the select key on a card with links", footer)
	}

	if _, id := openedId(t, ish, right, enter); id != "AAAAA-11111" {
		t.Errorf("enter on the first link opened %q, want AAAAA-11111", id)
	}
	if _, id := openedId(t, ish, right, j, j, j, enter); id != "BBBBB-22222" {
		t.Errorf("enter on the last link opened %q, want BBBBB-22222", id)
	}

	// Enter on a todo item still toggles it.
	ish, id := openedId(t, ish, right, j, enter)

	if id != "" || !strings.Contains(*ish.Conversation[1].Content, "- [x] first") {
		t.Errorf("enter on a todo item opened %q, content %q, want it checked", id, *ish.Conversation[1].Content)
	}
}

func TestCardWithOnlyLinksIsSelectable(t *testing.T) {
	ish := newTestIssue("See #AAAAA-11111")
	ish.SetIssueLookup(knownIssues)
	ish = resize(t, ish, 20)
	ish.setCursor(ish.firstCard())

	if _, id := openedId(t, ish, l, enter); id != "AAAAA-11111" {
		t.Errorf("opened %q, want AAAAA-11111", id)
	}
}

func TestOpeningLinkSavesChanges(t *testing.T) {
	s := &saver{}
	ish := linkIssue(t, s)
	ish.setCursor(ish.firstCard() + 1)

	// Check a todo item, then open a link.
	_, id := openedId(t, ish, right, j, enter, up, enter)

	if id != "AAAAA-11111" {
		t.Fatalf("opened %q, want AAAAA-11111", id)
	}
	if len(s.saved) != 1 || !strings.Contains(*s.saved[0].Conversation[1].Content, "- [x] first") {
		t.Errorf("saved %d times, want the checked item saved before opening the link", len(s.saved))
	}

	// If the changes can't be saved, the issue stays open.
	s = &saver{err: errors.New("disk full")}
	ish = linkIssue(t, s)
	ish.setCursor(ish.firstCard() + 1)

	if ish, id = openedId(t, ish, right, j, enter, up, enter); id != "" || ish.saveErr == nil {
		t.Errorf("opened %q with a save error %v, want the issue kept open with the error", id, ish.saveErr)
	}
}

func TestTagLinks(t *testing.T) {
	ish := linkIssue(t, nil)
	ish.setCursor(3)

	if footer := footerText(ish); !strings.Contains(footer, "→/l select") {
		t.Errorf("footer %q doesn't list the select key on a tag with links", footer)
	}

	header := ish.renderHeader()

	if !strings.Contains(header, styles.Link.Render("#AAAAA-11111")) || !strings.Contains(header, styles.Link.Render("#BBBBB-22222")) {
		t.Errorf("header %q doesn't style the tag's links", header)
	}
	if strings.Contains(header, styles.Link.Render("#CCCCC-33333")) {
		t.Error("a reference to an unknown issue is styled as a link")
	}

	ish = update(t, ish, right)

	if !ish.selectingItems() || !strings.Contains(ish.renderHeader(), styles.LinkSelected.Render("#AAAAA-11111")) {
		t.Errorf("after →, header %q, want the first link selected", ish.renderHeader())
	}
	if footer := footerText(ish); !strings.Contains(footer, "check/open") {
		t.Errorf("footer %q while selecting links, want the item keys", footer)
	}

	if _, id := openedId(t, ish, enter); id != "AAAAA-11111" {
		t.Errorf("enter opened %q, want AAAAA-11111", id)
	}
	if _, id := openedId(t, ish, j, j, enter); id != "BBBBB-22222" {
		t.Errorf("enter on the last link opened %q, want BBBBB-22222", id)
	}

	// Done, and moving away, stop selecting.
	if ish := update(t, ish, left); ish.selectingItems() {
		t.Error("still selecting links after ←")
	}
	if ish := update(t, ish, tab); ish.Metadata[1].selecting {
		t.Error("tag still selecting links after tab moved the cursor away")
	}
}

func TestTagWithoutLinks(t *testing.T) {
	ish := linkIssue(t, nil)
	ish.setCursor(2)
	ish = update(t, ish, right)

	if ish.selectingItems() {
		t.Error("selecting links on a tag without links")
	}
	if footer := footerText(ish); strings.Contains(footer, "→/l select") {
		t.Errorf("footer %q lists the select key on a tag without links", footer)
	}
}

func TestNoLinksWithoutLookup(t *testing.T) {
	ish := newTestIssue("See #AAAAA-11111")
	ish.Metadata = append(ish.Metadata, Tag{Key: "relation.parent", Values: Tokens("#AAAAA-11111")})
	ish = resize(t, ish, 20)

	if strings.Contains(ish.View().Content, styles.Link.Render("#AAAAA-11111")) {
		t.Error("references are linked without an issue lookup")
	}
}

func TestNewCommentsAndTagsLink(t *testing.T) {
	ish := linkIssue(t, nil)
	ish.SetAuthor("me")
	ish = update(t, ish, tea.KeyPressMsg{Code: 'c', Text: "c"})

	if !ish.Conversation[len(ish.Conversation)-1].issues("AAAAA-11111") {
		t.Error("a new comment doesn't have the issue lookup")
	}

	ish = update(t, ish, ctrlS, tea.KeyPressMsg{Code: 'm', Text: "m"})

	if tag := ish.Metadata[len(ish.Metadata)-1]; tag.issues == nil {
		t.Error("a new tag doesn't have the issue lookup")
	}
}
