package models

import (
	"regexp"
	"strings"

	"github.com/factory-factory/ish/styles"
)

// todoPattern matches a todo item: a line starting with "- [ ]" or "- [x]",
// optionally indented. The second group is the check mark.
var todoPattern = regexp.MustCompile(`^(\s*- \[)([ xX])\](\s|$)`)

// todoMarker is inserted into the selected todo item's markdown before it's
// rendered, to find the item in the rendered output. It's a private use
// character, so it can't appear in content.
const todoMarker = ""

// escapePattern matches ANSI escape sequences, including hyperlinks.
var escapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)`)

// todoLines returns the indices of the lines in content that are todo items.
// Lines in fenced code blocks are not todo items.
func todoLines(content string) []int {
	var lines []int
	fenced := false

	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
			continue
		}

		if !fenced && todoPattern.MatchString(line) {
			lines = append(lines, i)
		}
	}

	return lines
}

// todos returns the indices of the lines in the card's content that are todo
// items.
func (c Card) todos() []int {
	if c.Content == nil {
		return nil
	}

	return todoLines(*c.Content)
}

// startTodos begins selecting the card's todo items, starting with the first,
// reporting whether it has any.
func (c *Card) startTodos() bool {
	if len(c.todos()) == 0 {
		return false
	}

	c.selectingTodo = true
	c.todo = 0

	return true
}

// stopTodos stops selecting the card's todo items.
func (c *Card) stopTodos() {
	c.selectingTodo = false
}

// moveTodo moves the todo item selection by delta items, stopping at the
// first and last items.
func (c *Card) moveTodo(delta int) {
	c.todo = min(max(c.todo+delta, 0), len(c.todos())-1)
}

// toggleTodo checks the selected todo item if it's unchecked, and unchecks it
// if it's checked.
func (c *Card) toggleTodo() {
	todos := c.todos()

	if !c.selectingTodo || c.todo >= len(todos) {
		return
	}

	lines := strings.Split(*c.Content, "\n")
	line := lines[todos[c.todo]]
	match := todoPattern.FindStringSubmatchIndex(line)

	mark := "x"
	if line[match[4]:match[5]] != " " {
		mark = " "
	}

	lines[todos[c.todo]] = line[:match[4]] + mark + line[match[5]:]
	content := strings.Join(lines, "\n")
	c.Content = &content
}

// markTodo returns content with todoMarker inserted at the start of the text
// of the todo item on line i.
func markTodo(content string, i int) string {
	lines := strings.Split(content, "\n")
	match := todoPattern.FindStringSubmatchIndex(lines[i])

	// Insert after "- [ ]" and the space that follows it, if any.
	end := match[1]
	lines[i] = lines[i][:end] + todoMarker + lines[i][end:]

	return strings.Join(lines, "\n")
}

// highlightTodo finds the line of rendered markdown containing todoMarker,
// removes the marker, and highlights the line's text. It returns the result
// and the index of the highlighted line, or -1 if there's no marker.
func highlightTodo(rendered string) (string, int) {
	lines := strings.Split(rendered, "\n")

	for i, line := range lines {
		if !strings.Contains(line, todoMarker) {
			continue
		}

		text := escapePattern.ReplaceAllString(strings.Replace(line, todoMarker, "", 1), "")
		indent := len(text) - len(strings.TrimLeft(text, " "))
		lines[i] = text[:indent] + styles.TodoSelected.Render(strings.TrimSpace(text))

		return strings.Join(lines, "\n"), i
	}

	return rendered, -1
}
