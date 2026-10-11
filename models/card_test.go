package models

import (
	"regexp"
	"strings"
	"testing"
)

var escapes = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// indentOf returns the column at which text first appears in the rendered
// output, ignoring ANSI escape sequences.
func indentOf(t *testing.T, rendered, text string) int {
	t.Helper()

	for _, line := range strings.Split(escapes.ReplaceAllString(rendered, ""), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return i
		}
	}

	t.Fatalf("%q not found in rendered card:\n%s", text, rendered)
	return -1
}

func TestCardViewIndentsNestedList(t *testing.T) {
	content := "- [ ] Parent\n  - [ ] Child\n    - [ ] Grandchild"
	card := Card{
		DateCreated: "2023-05-27T13:41:50-0400",
		Author:      "author",
		Content:     &content,
	}

	rendered := card.View().Content

	parent := indentOf(t, rendered, "Parent")
	child := indentOf(t, rendered, "Child")
	grandchild := indentOf(t, rendered, "Grandchild")

	if !(parent < child && child < grandchild) {
		t.Errorf("indents of Parent, Child, Grandchild = %d, %d, %d, want increasing",
			parent, child, grandchild)
	}
}
