package parse

import (
	"os"
	"strings"
	"testing"
)

const header = `# Title

[author 2023-05-26T23:35:36-0400]

- state: open
- assignees: alice, bob
`

func parse(t *testing.T, source string) []string {
	t.Helper()

	issue, err := ParseIssue(strings.NewReader(source))

	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}

	contents := make([]string, len(issue.Conversation))
	for i, card := range issue.Conversation {
		contents[i] = *card.Content
	}

	return contents
}

func TestParseIssueHeader(t *testing.T) {
	issue, err := ParseIssue(strings.NewReader(header))

	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}

	if issue.Title != "Title" {
		t.Errorf("Title = %q, want %q", issue.Title, "Title")
	}
	if issue.Author != "author" {
		t.Errorf("Author = %q, want %q", issue.Author, "author")
	}
	if issue.DateCreated != "2023-05-26T23:35:36-0400" {
		t.Errorf("DateCreated = %q, want %q", issue.DateCreated, "2023-05-26T23:35:36-0400")
	}
	if len(issue.Metadata) != 2 || issue.Metadata[1].Key != "assignees" ||
		strings.Join(issue.Metadata[1].Values, ",") != "alice,bob" {
		t.Errorf("Metadata = %+v, want state and assignees tags", issue.Metadata)
	}
	if len(issue.Conversation) != 0 {
		t.Errorf("Conversation has %d cards, want 0", len(issue.Conversation))
	}
}

func TestParseIssueCardBylines(t *testing.T) {
	source := header + `
---

[carol 2023-05-27T13:41:50-0400]

First.

---

[dave 2023-05-28T13:41:50-0400]

Second.
`

	issue, err := ParseIssue(strings.NewReader(source))

	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}

	if len(issue.Conversation) != 2 {
		t.Fatalf("Conversation has %d cards, want 2", len(issue.Conversation))
	}

	for i, want := range []struct{ author, date string }{
		{"carol", "2023-05-27T13:41:50-0400"},
		{"dave", "2023-05-28T13:41:50-0400"},
	} {
		card := issue.Conversation[i]
		if card.Author != want.author || card.DateCreated != want.date {
			t.Errorf("card %d byline = %q %q, want %q %q",
				i, card.Author, card.DateCreated, want.author, want.date)
		}
	}
}

func TestParseIssueKeepsCardSource(t *testing.T) {
	first := "- [ ] Parent\n  - [ ] Child\n    - [ ] Grandchild\n\n3. Three\n4. Four"
	second := "- Item\n\n  Second paragraph\n\n  ```\n  ---\n  code\n  ```\n\n    indented code"

	source := header + "\n---\n\n[carol 2023-05-27T13:41:50-0400]\n\n" + first +
		"\n\n---\n\n[dave 2023-05-28T13:41:50-0400]\n\n" + second + "\n\n"

	got := parse(t, source)

	if len(got) != 2 {
		t.Fatalf("parsed %d cards, want 2: %q", len(got), got)
	}
	if got[0] != first {
		t.Errorf("first card content = %q, want %q", got[0], first)
	}
	if got[1] != second {
		t.Errorf("second card content = %q, want %q", got[1], second)
	}
}

func TestParseIssueEmptyCard(t *testing.T) {
	source := header + "\n---\n\n[carol 2023-05-27T13:41:50-0400]\n"

	got := parse(t, source)

	if len(got) != 1 || got[0] != "" {
		t.Errorf("cards = %q, want one empty card", got)
	}
}

func TestParseIssueRequiresBreakBetweenCards(t *testing.T) {
	source := header + "\nNot a horizontal rule.\n"

	if _, err := ParseIssue(strings.NewReader(source)); err == nil {
		t.Error("ParseIssue() error = nil, want an error for a missing horizontal rule")
	}
}

func TestParseIssueSampleKeepsNestedList(t *testing.T) {
	f, err := os.Open("../issues/01AN4Z07BY.md")

	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	issue, err := ParseIssue(f)

	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}

	want := "- [ ] Create ish commands\n  - [ ] ish init\n"

	for _, card := range issue.Conversation {
		if strings.Contains(*card.Content, want) {
			return
		}
	}

	t.Errorf("no card contains the nested list %q", want)
}
