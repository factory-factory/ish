package models

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var (
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	ctrlS = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	ctrlU = tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
)

// typeText sends a key press for each character in s.
func typeText(t *testing.T, ish Issue, s string) Issue {
	t.Helper()

	for _, r := range s {
		ish = update(t, ish, tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	return ish
}

// plain returns the issue's view with ANSI escape sequences removed.
func plain(ish Issue) string {
	return escapes.ReplaceAllString(ish.View().Content, "")
}

// editAt moves the cursor to c and starts editing the field there.
func editAt(t *testing.T, ish Issue, c int) Issue {
	t.Helper()

	ish.setCursor(c)
	ish = update(t, ish, enter)

	if !ish.editing {
		t.Fatalf("not editing after enter at cursor %d", c)
	}

	return ish
}

func TestEnterWithNothingSelected(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)
	ish = update(t, ish, enter)

	if ish.editing {
		t.Error("editing after enter with nothing selected")
	}
	if footer := ish.renderFooter(); strings.Contains(footer, "edit") {
		t.Errorf("footer %q lists the edit key with nothing selected", footer)
	}
}

func TestEditTitle(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)
	ish = editAt(t, ish, 1)

	// Only the title text is editable. The ID is shown beside it.
	if got := ish.titleInput.Value(); got != "Title" {
		t.Errorf("title input = %q, want %q", got, "Title")
	}
	if view := plain(ish); !strings.Contains(view, "[00000]") {
		t.Errorf("View() = %q, want the issue ID beside the title input", view)
	}

	ish = typeText(t, ish, " edited")
	ish = update(t, ish, enter)

	if ish.editing {
		t.Error("still editing after enter")
	}
	if ish.Title != "Title edited" || ish.Id != "00000" {
		t.Errorf("Id, Title = %q, %q, want %q, %q", ish.Id, ish.Title, "00000", "Title edited")
	}
	if view := plain(ish); !strings.Contains(view, "[00000] Title edited") {
		t.Errorf("View() = %q, want the edited title", view)
	}
}

func TestEditTitleCancelAndBlank(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, " edited")
	ish = update(t, ish, esc)

	if ish.editing || ish.Title != "Title" {
		t.Errorf("editing, Title = %v, %q after esc, want false, %q", ish.editing, ish.Title, "Title")
	}

	ish = editAt(t, ish, 1)
	ish = update(t, ish, ctrlU, enter)

	if ish.Title != "Title" {
		t.Errorf("Title = %q after saving a blank title, want it unchanged", ish.Title)
	}
}

func TestEditTagValues(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)
	ish = editAt(t, ish, 2)

	tag := ish.Metadata[0]

	// Only the values are editable. The key is shown beside them.
	if got := tag.input.Value(); got != "open" {
		t.Errorf("tag input = %q, want %q", got, "open")
	}
	if view := plain(ish); !strings.Contains(view, "• state: open") {
		t.Errorf("View() = %q, want the key beside the values input", view)
	}

	ish = update(t, ish, ctrlU)
	ish = typeText(t, ish, "closed,  blocked , ,")
	ish = update(t, ish, enter)

	tag = ish.Metadata[0]

	if ish.editing || tag.editing {
		t.Error("still editing after enter")
	}
	if tag.Key != "state" || strings.Join(tag.Values, "|") != "closed|blocked" {
		t.Errorf("tag = %q %q, want state [closed blocked]", tag.Key, tag.Values)
	}
}

func TestEditTagCancel(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)
	ish = editAt(t, ish, 2)
	ish = typeText(t, ish, ", blocked")
	ish = update(t, ish, esc)

	if tag := ish.Metadata[0]; tag.editing || strings.Join(tag.Values, "|") != "open" {
		t.Errorf("tag editing, values = %v, %q after esc, want false, [open]", tag.editing, tag.Values)
	}
}

func TestEditCardAsMarkdown(t *testing.T) {
	ish := resize(t, newTestIssue("Some **bold** text"), 20)
	ish = editAt(t, ish, ish.firstCard())

	card := ish.Conversation[0]

	if !card.editing {
		t.Fatal("card is not editing")
	}

	// The raw markdown is shown in place of the rendered markdown, inside the
	// highlighted border.
	view := plain(ish)

	if !strings.Contains(view, "Some **bold** text") {
		t.Errorf("View() = %q, want the raw markdown", view)
	}
	if !strings.Contains(view, "┏") {
		t.Errorf("View() = %q, want the selected card's border", view)
	}

	// Enter inserts a new line, and ctrl+s saves.
	ish = update(t, ish, tea.KeyPressMsg{Code: tea.KeyEnd})
	ish = update(t, ish, enter)
	ish = typeText(t, ish, "More")

	if !ish.editing {
		t.Fatal("enter stopped editing the card")
	}

	ish = update(t, ish, ctrlS)
	card = ish.Conversation[0]

	if ish.editing || card.editing {
		t.Error("still editing after ctrl+s")
	}
	if got, want := *card.Content, "Some **bold** text\nMore"; got != want {
		t.Errorf("Content = %q, want %q", got, want)
	}
	if view := plain(ish); strings.Contains(view, "**bold**") || !strings.Contains(view, "More") {
		t.Errorf("View() = %q, want the edited markdown rendered", view)
	}
}

func TestEditCardCancel(t *testing.T) {
	ish := resize(t, newTestIssue("Original"), 20)
	ish = editAt(t, ish, ish.firstCard())
	ish = typeText(t, ish, "Changed ")
	ish = update(t, ish, esc)

	if card := ish.Conversation[0]; card.editing || *card.Content != "Original" {
		t.Errorf("card editing, content = %v, %q after esc, want false, %q", card.editing, *card.Content, "Original")
	}
}

func TestEditCardFitsViewport(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1), lines(40)), 12)
	ish = editAt(t, ish, ish.firstCard()+1)

	s := ish.spans[1]
	top, area := ish.viewport.YOffset(), ish.viewport.Height()

	if s.height > area {
		t.Errorf("editing card is %d lines, want at most the viewport height %d", s.height, area)
	}
	if s.top < top || s.bottom() > top+area {
		t.Errorf("editing card %+v not in view at offset %d with area %d", s, top, area)
	}
}

func TestEditingCapturesKeys(t *testing.T) {
	ish := newTestIssue("Content")
	ish.SetClosable(true)
	ish = resize(t, ish, 20)
	ish = editAt(t, ish, 1)

	// Keys bound in the issue view are typed into the editor instead.
	m, cmd := ish.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	ish = m.(Issue)

	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("q quit while editing")
		}
	}
	if got := ish.titleInput.Value(); got != "Titleq" {
		t.Errorf("title input = %q, want %q", got, "Titleq")
	}

	// Esc cancels the edit rather than closing the issue.
	m, cmd = ish.Update(esc)
	ish = m.(Issue)

	if cmd != nil {
		t.Error("esc while editing returned a command, want it to only cancel the edit")
	}
	if ish.editing {
		t.Error("still editing after esc")
	}

	// Ctrl+c still quits.
	ish = editAt(t, ish, 1)
	_, cmd = ish.Update(ctrlC)

	if cmd == nil {
		t.Fatal("ctrl+c while editing returned no command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c while editing didn't quit")
	}
}

func TestEditingFooter(t *testing.T) {
	ish := resize(t, newTestIssue("Content"), 20)

	ish.setCursor(1)
	if footer := ish.renderFooter(); !strings.Contains(footer, "edit") {
		t.Errorf("footer %q doesn't list the edit key", footer)
	}

	ish = editAt(t, ish, 1)
	if footer := escapes.ReplaceAllString(ish.renderFooter(), ""); !strings.Contains(footer, "enter save") || !strings.Contains(footer, "esc cancel") {
		t.Errorf("footer %q while editing the title, want enter to save and esc to cancel", footer)
	}

	ish = update(t, ish, esc)
	ish = editAt(t, ish, ish.firstCard())
	if footer := escapes.ReplaceAllString(ish.renderFooter(), ""); !strings.Contains(footer, "ctrl+s save") {
		t.Errorf("footer %q while editing a card, want ctrl+s to save", footer)
	}
}

func TestBlankIssueOpensEditingTitle(t *testing.T) {
	ish := newTestIssue(lines(1))
	ish.Title = "  "

	ish.initComponents()
	header := len(strings.Split(ish.renderHeader(), "\n"))

	m, cmd := ish.Update(tea.WindowSizeMsg{Width: 100, Height: header + 20})
	ish = m.(Issue)

	if !ish.editing || ish.cursor != 1 {
		t.Fatalf("editing, cursor = %v, %d on first show, want true, 1", ish.editing, ish.cursor)
	}
	if cmd == nil {
		t.Error("first show returned no command, want the title input's cursor to start")
	}

	// Cancelling leaves the title blank, and later resizes don't start
	// editing again.
	ish = update(t, ish, esc, tea.WindowSizeMsg{Width: 80, Height: header + 10})

	if ish.editing {
		t.Error("editing again after a resize")
	}

	ish = editAt(t, ish, 1)
	ish = typeText(t, ish, "Named")
	ish = update(t, ish, enter)

	if ish.Title != "Named" || !ish.dirty {
		t.Errorf("Title, dirty = %q, %v, want %q, true", ish.Title, ish.dirty, "Named")
	}
}

func TestTitledIssueDoesNotOpenEditing(t *testing.T) {
	ish := resize(t, newTestIssue(lines(1)), 10)

	if ish.editing || ish.cursor != 0 {
		t.Errorf("editing, cursor = %v, %d on first show, want false, 0", ish.editing, ish.cursor)
	}
}

func TestBackgroundColorRestylesEditor(t *testing.T) {
	ish := resize(t, newTestIssue("Content"), 10)
	ish = editAt(t, ish, ish.firstCard())

	// Editing started with dark styles. Once the terminal reports a light
	// background, the editor switches to light styles.
	before := ish.Conversation[0].editor.View()
	ish = update(t, ish, tea.BackgroundColorMsg{Color: color.White})

	if !ish.light {
		t.Fatal("light background not recorded")
	}
	if after := ish.Conversation[0].editor.View(); after == before {
		t.Error("comment editor unchanged after a light background was reported, want light styles")
	}
}
