package models

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/factory-factory/ish/styles"
)

const todoContent = "Intro\n\n- [ ] first\n- [x] second\n  - [X] nested\n- plain item"

var (
	right = tea.KeyPressMsg{Code: tea.KeyRight}
	left  = tea.KeyPressMsg{Code: tea.KeyLeft}
	l     = tea.KeyPressMsg{Code: 'l', Text: "l"}
	h     = tea.KeyPressMsg{Code: 'h', Text: "h"}
	j     = tea.KeyPressMsg{Code: 'j', Text: "j"}
)

func TestTodoLines(t *testing.T) {
	content := strings.Join([]string{
		"- [ ] unchecked",         // 0
		"- [x] checked",           // 1
		"- [X] checked uppercase", // 2
		"    - [ ] indented",      // 3
		"- [ ]",                   // 4: empty item
		"* [ ] other bullet",      // 5
		"- [] no space",           // 6
		"-[ ] no space",           // 7
		"- [ ]x no space after",   // 8
		"text - [ ] mid-line",     // 9
		"```",                     // 10
		"- [ ] in a code block",   // 11
		"```",                     // 12
		"~~~",                     // 13
		"- [x] in a code block",   // 14
		"~~~",                     // 15
		"- [x] after code",        // 16
	}, "\n")

	want := []int{0, 1, 2, 3, 4, 16}

	if got := todoLines(content); !slices.Equal(got, want) {
		t.Errorf("todoLines() = %v, want %v", got, want)
	}
}

func TestToggleTodo(t *testing.T) {
	content := todoContent
	card := Card{Content: &content}

	if !card.startTodos() {
		t.Fatal("startTodos() = false, want true")
	}

	card.toggleTodo()
	card.moveTodo(1)
	card.toggleTodo()
	card.moveTodo(1)
	card.toggleTodo()

	want := "Intro\n\n- [x] first\n- [ ] second\n  - [ ] nested\n- plain item"

	if *card.Content != want {
		t.Errorf("Content = %q, want %q", *card.Content, want)
	}
	if content != todoContent {
		t.Error("toggling changed the original content string")
	}
}

func TestMoveTodoStopsAtEnds(t *testing.T) {
	content := todoContent
	card := Card{Content: &content}
	card.startTodos()

	card.moveTodo(-1)
	if card.todo != 0 {
		t.Errorf("todo = %d after moving up from the first item, want 0", card.todo)
	}

	card.moveTodo(10)
	if card.todo != 2 {
		t.Errorf("todo = %d after moving past the last item, want 2", card.todo)
	}
}

func TestStartTodosWithoutTodos(t *testing.T) {
	content := "No todos\n\n- plain item"
	card := Card{Content: &content}

	if card.startTodos() || card.selectingTodo {
		t.Error("startTodos() on a card without todo items started selecting")
	}

	card = Card{}
	if card.startTodos() {
		t.Error("startTodos() on a card without content started selecting")
	}
}

func TestRenderContentHighlightsSelectedTodo(t *testing.T) {
	content := todoContent
	card := Card{
		DateCreated: "2023-05-27T13:41:50-0400",
		Author:      "author",
		Content:     &content,
		selected:    true,
	}

	card.startTodos()
	card.moveTodo(1)

	rendered, row := card.renderContent()
	lines := strings.Split(rendered, "\n")

	if row < 0 {
		t.Fatal("no highlighted row")
	}
	if strings.Contains(rendered, todoMarker) {
		t.Error("rendered content contains the todo marker")
	}
	if got := escapes.ReplaceAllString(lines[row], ""); !strings.Contains(got, "second") {
		t.Errorf("highlighted line = %q, want the second item", got)
	}
	if !strings.Contains(lines[row], styles.TodoSelected.Render(strings.TrimSpace(escapes.ReplaceAllString(lines[row], "")))) {
		t.Errorf("line %q isn't highlighted", lines[row])
	}

	// The row in the card's view accounts for the byline and border.
	view := strings.Split(card.View().Content, "\n")
	if got := escapes.ReplaceAllString(view[card.todoRow()], ""); !strings.Contains(got, "second") {
		t.Errorf("view line %d = %q, want the second item", card.todoRow(), got)
	}

	// Without todo selection, nothing is highlighted.
	card.stopTodos()
	if _, row := card.renderContent(); row != -1 {
		t.Errorf("highlighted row = %d without todo selection, want -1", row)
	}
	if card.todoRow() != -1 {
		t.Errorf("todoRow() = %d without todo selection, want -1", card.todoRow())
	}
}

// footerText returns the issue's help footer without ANSI escape sequences.
func footerText(ish Issue) string {
	return escapes.ReplaceAllString(ish.renderFooter(), "")
}

// todoIssue returns an issue whose second card has todo items, with that card
// selected.
func todoIssue(t *testing.T, s *saver) Issue {
	t.Helper()

	ish := newTestIssue("No todos here", todoContent)
	ish.SetClosable(true)

	if s != nil {
		ish.SetSave(s.save)
	}

	ish = resize(t, ish, 30)
	ish.setCursor(ish.firstCard() + 1)

	return ish
}

func TestRightSelectsTodos(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{right, l} {
		ish := todoIssue(t, nil)

		if footer := footerText(ish); !strings.Contains(footer, "→/l todos") {
			t.Errorf("footer %q doesn't list the todos key on a card with todo items", footer)
		}

		ish = update(t, ish, key)

		if !ish.selectingTodo() || ish.Conversation[1].todo != 0 {
			t.Fatalf("after %s: selectingTodo, todo = %v, %d, want true, 0",
				key, ish.selectingTodo(), ish.Conversation[1].todo)
		}
		if footer := footerText(ish); !strings.Contains(footer, "check/uncheck") {
			t.Errorf("footer %q while selecting todos, want the todo keys", footer)
		}
	}
}

func TestRightWithoutTodos(t *testing.T) {
	ish := todoIssue(t, nil)

	// On a card without todo items.
	ish.setCursor(ish.firstCard())
	ish = update(t, ish, right)

	if ish.selectingTodo() {
		t.Error("selecting todos on a card without todo items")
	}
	if footer := footerText(ish); strings.Contains(footer, "→/l todos") {
		t.Errorf("footer %q lists the todos key on a card without todo items", footer)
	}

	// On the title.
	ish.setCursor(1)
	ish = update(t, ish, right)

	if ish.selectingTodo() {
		t.Error("selecting todos with the title selected")
	}
}

func TestToggleTodosAndSave(t *testing.T) {
	s := &saver{}
	ish := todoIssue(t, s)

	// Check the first item, and uncheck the second.
	ish = update(t, ish, right, enter, j, enter)

	want := "Intro\n\n- [x] first\n- [ ] second\n  - [X] nested\n- plain item"

	if got := *ish.Conversation[1].Content; got != want {
		t.Errorf("Content = %q, want %q", got, want)
	}
	if !ish.selectingTodo() {
		t.Error("stopped selecting todos after toggling")
	}

	// The toggled items are saved on exit.
	_, msg := exitWith(t, ish, q)

	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q sent %T, want tea.QuitMsg", msg)
	}
	if len(s.saved) != 1 || *s.saved[0].Conversation[1].Content != want {
		t.Errorf("saved %v, want one save with the toggled items", s.saved)
	}
}

func TestDoneSelectingTodos(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{left, h, esc} {
		ish := todoIssue(t, nil)
		ish = update(t, ish, right)

		m, cmd := ish.Update(key)
		ish = m.(Issue)

		if ish.selectingTodo() {
			t.Errorf("still selecting todos after %s", key)
		}
		if cmd != nil {
			t.Errorf("%s returned a command, want it to only stop selecting todos", key)
		}
		if got := ish.cardIndex(); got != 1 {
			t.Errorf("cardIndex() = %d after %s, want the card to stay selected", got, key)
		}
	}
}

func TestTabStopsSelectingTodos(t *testing.T) {
	ish := todoIssue(t, nil)
	ish = update(t, ish, right, tab)

	if ish.Conversation[1].selectingTodo {
		t.Error("card still selecting todos after tab moved the cursor away")
	}
}

func TestSelectedTodoScrollsIntoView(t *testing.T) {
	items := make([]string, 30)
	for i := range items {
		items[i] = "- [ ] item"
	}

	ish := resize(t, newTestIssue(strings.Join(items, "\n")), 8)
	ish.setCursor(ish.firstCard())
	ish = update(t, ish, right)

	for i := range 29 {
		ish = update(t, ish, j)

		line := ish.spans[0].top + ish.Conversation[0].todoRow()
		top, area := ish.viewport.YOffset(), ish.viewport.Height()

		if line < top || line >= top+area {
			t.Fatalf("item %d on line %d not in view at offset %d with area %d", i+1, line, top, area)
		}
	}
}
