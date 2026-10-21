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

// todoOffsets returns the byte offset in content at which the text of each
// todo item starts, after "- [ ]" and the space that follows it, if any.
func todoOffsets(content string) []int {
	lines := strings.Split(content, "\n")
	starts := make([]int, len(lines))

	for i, offset := 0, 0; i < len(lines); i++ {
		starts[i] = offset
		offset += len(lines[i]) + 1
	}

	var offsets []int

	for _, i := range todoLines(content) {
		offsets = append(offsets, starts[i]+todoPattern.FindStringSubmatchIndex(lines[i])[1])
	}

	return offsets
}

// toggleTodo checks the todo item on line i of the card's content if it's
// unchecked, and unchecks it if it's checked.
func (c *Card) toggleTodo(i int) {
	lines := strings.Split(*c.Content, "\n")
	line := lines[i]
	match := todoPattern.FindStringSubmatchIndex(line)

	mark := "x"
	if line[match[4]:match[5]] != " " {
		mark = " "
	}

	lines[i] = line[:match[4]] + mark + line[match[5]:]
	content := strings.Join(lines, "\n")
	c.Content = &content
}

// highlightTodo finds the line of rendered markdown containing todoMarker,
// removes the marker, and highlights the line's text, underlining any links
// in it. It returns the result and the index of the highlighted line, or -1
// if there's no marker.
func highlightTodo(rendered string) (string, int) {
	lines := strings.Split(rendered, "\n")

	for i, line := range lines {
		if !strings.Contains(line, todoMarker) {
			continue
		}

		// The selected item is highlighted as a whole, including any
		// users' names in it.
		text := escapePattern.ReplaceAllString(strings.Replace(line, todoMarker, "", 1), "")
		text = strings.NewReplacer(mentionMarker, "", mentionEndMarker, "").Replace(text)
		indent := len(text) - len(strings.TrimLeft(text, " "))
		lines[i] = text[:indent] + renderLinks(strings.TrimSpace(text), styles.TodoSelected, todoLinkStyle, todoLinkStyle)

		return strings.Join(lines, "\n"), i
	}

	return rendered, -1
}

// todoLinkStyle is the style of a link in the selected todo item, which keeps
// the item's colors, so the link stays visible on its background.
var todoLinkStyle = styles.TodoSelected.Underline(true)
