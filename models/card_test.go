package models

import (
	"regexp"
	"slices"
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

func TestCardViewIndentsWrappedListItems(t *testing.T) {
	long := "with enough text in it that it has to wrap onto a second line here"
	content := strings.Join([]string{
		"- A bullet item " + long,
		"  - A nested item " + long,
		"",
		"1. A numbered item " + long,
		"",
		"- [ ] A todo item " + long,
	}, "\n")
	card := Card{
		DateCreated: "2023-05-27T13:41:50-0400",
		Author:      "author",
		Content:     &content,
	}

	rendered, _ := card.renderContent()

	if strings.Contains(rendered, "") {
		t.Error("the hanging indent mark is in the rendered card")
	}

	lines := strings.Split(escapes.ReplaceAllString(rendered, ""), "\n")

	// Each item's continuation line starts under the item's text.
	for _, tt := range []struct {
		marker string
		indent int
	}{
		{"• A bullet item", 2},
		{"  • A nested item", 4},
		{"1. A numbered item", 3},
		{"[ ] A todo item", 4},
	} {
		i := slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, tt.marker) })

		if i < 0 || i+1 >= len(lines) {
			t.Errorf("no wrapped item starting %q in:\n%s", tt.marker, strings.Join(lines, "\n"))
			continue
		}

		next := lines[i+1]

		if got := len(next) - len(strings.TrimLeft(next, " ")); got != tt.indent || strings.TrimSpace(next) == "" {
			t.Errorf("line after %q = %q, want text indented by %d", tt.marker, next, tt.indent)
		}
	}
}
