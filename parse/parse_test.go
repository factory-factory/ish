package parse

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/factory-factory/ish/models"
)

const header = `# Title

[author@example.com 2023-05-26T23:35:36-0400]

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
	if issue.Author != "author@example.com" {
		t.Errorf("Author = %q, want %q", issue.Author, "author@example.com")
	}
	if issue.DateCreated != "2023-05-26T23:35:36-0400" {
		t.Errorf("DateCreated = %q, want %q", issue.DateCreated, "2023-05-26T23:35:36-0400")
	}
	if len(issue.Metadata) != 2 || issue.Metadata[1].Key != "assignees" ||
		strings.Join(models.Canonical(issue.Metadata[1].Values), ",") != "alice,bob" {
		t.Errorf("Metadata = %+v, want state and assignees tags", issue.Metadata)
	}
	if len(issue.Conversation) != 0 {
		t.Errorf("Conversation has %d cards, want 0", len(issue.Conversation))
	}
}

func TestParseIssueCardBylines(t *testing.T) {
	source := header + `
---

[carol@example.com 2023-05-27T13:41:50-0400]

First.

---

[dave@example.com 2023-05-28T13:41:50-0400]

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
		{"carol@example.com", "2023-05-27T13:41:50-0400"},
		{"dave@example.com", "2023-05-28T13:41:50-0400"},
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

	source := header + "\n---\n\n[carol@example.com 2023-05-27T13:41:50-0400]\n\n" + first +
		"\n\n---\n\n[dave@example.com 2023-05-28T13:41:50-0400]\n\n" + second + "\n\n"

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
	source := header + "\n---\n\n[carol@example.com 2023-05-27T13:41:50-0400]\n"

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

func TestParseIssueSamplesKeepCardSource(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", IssuesDir, "*.md"))

	if err != nil || len(paths) == 0 {
		t.Fatalf("no sample issues found (err = %v)", err)
	}

	for _, path := range paths {
		source, err := os.ReadFile(path)

		if err != nil {
			t.Fatal(err)
		}

		issue, err := ParseIssue(strings.NewReader(string(source)))

		if err != nil {
			t.Errorf("%s: ParseIssue() error = %v", path, err)
			continue
		}

		// Each card's content is copied verbatim from the file, so nested
		// lists and other indentation are kept.
		for i, card := range issue.Conversation {
			if !strings.Contains(string(source), *card.Content) {
				t.Errorf("%s: card %d content isn't verbatim from the file:\n%s", path, i, *card.Content)
			}
		}
	}
}

// writeIssues writes each issue source to a file named for its ID in a new
// temporary directory, which it returns.
func writeIssues(t *testing.T, sources map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	for id, source := range sources {
		if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestLoadIssue(t *testing.T) {
	dir := writeIssues(t, map[string]string{"ABC123": header})

	issue, err := LoadIssue(dir, "ABC123")

	if err != nil {
		t.Fatalf("LoadIssue() error = %v", err)
	}
	if issue.Id != "ABC123" || issue.Title != "Title" {
		t.Errorf("LoadIssue() = Id %q, Title %q, want %q, %q", issue.Id, issue.Title, "ABC123", "Title")
	}
}

func TestLoadIssueErrors(t *testing.T) {
	dir := writeIssues(t, map[string]string{"BAD": "Not an issue\n"})

	if _, err := LoadIssue(dir, "MISSING"); err == nil {
		t.Error("LoadIssue() of a missing file: error = nil, want an error")
	}

	_, err := LoadIssue(dir, "BAD")

	if err == nil || !strings.Contains(err.Error(), "BAD.md") {
		t.Errorf("LoadIssue() of an invalid file: error = %v, want an error naming BAD.md", err)
	}
}

func TestListIssues(t *testing.T) {
	dir := writeIssues(t, map[string]string{
		"BBB": strings.Replace(header, "# Title", "# Second", 1),
		"AAA": strings.Replace(header, "# Title", "# First", 1),
	})

	// Files that aren't markdown, and directories, are ignored.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	issues, err := ListIssues(dir)

	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}

	var got []string
	for _, issue := range issues {
		got = append(got, issue.Id+" "+issue.Title)
	}

	if want := []string{"AAA First", "BBB Second"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("ListIssues() = %q, want %q", got, want)
	}
}

func TestListIssuesErrors(t *testing.T) {
	if _, err := ListIssues(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("ListIssues() of a missing directory: error = nil, want an error")
	}

	dir := writeIssues(t, map[string]string{"AAA": header, "BAD": "Not an issue\n"})

	if _, err := ListIssues(dir); err == nil {
		t.Error("ListIssues() with an invalid file: error = nil, want an error")
	}
}

func TestParseIssueWithoutMetadata(t *testing.T) {
	source := "# Title\n\n[author@example.com 2023-05-26T23:35:36-0400]\n\n---\n\n[carol@example.com 2023-05-27T13:41:50-0400]\n\nComment\n"

	issue, err := ParseIssue(strings.NewReader(source))

	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}
	if len(issue.Metadata) != 0 {
		t.Errorf("Metadata = %v, want none", issue.Metadata)
	}
	if len(issue.Conversation) != 1 || *issue.Conversation[0].Content != "Comment" {
		t.Errorf("Conversation = %v, want one card with %q", issue.Conversation, "Comment")
	}
}

func TestParseBylineForms(t *testing.T) {
	for _, tt := range []struct {
		byline, author, date string
	}{
		{"[alice@example.com 2026-10-01T09:00:00Z]", "alice@example.com", "2026-10-01T09:00:00Z"},
		{"[<alice@example.com> 2026-10-01T09:00:00Z]", "<alice@example.com>", "2026-10-01T09:00:00Z"},
		{`["Alice Liddell" <alice@example.com> 2026-10-01T09:00:00Z]`, `"Alice Liddell" <alice@example.com>`, "2026-10-01T09:00:00Z"},
		{"[Alice Liddell <alice@example.com> 2026-10-01T09:00:00Z]", "Alice Liddell <alice@example.com>", "2026-10-01T09:00:00Z"},
		{"[<alice@example.com> 2023-05-28 11:55:23 -0400]", "<alice@example.com>", "2023-05-28 11:55:23 -0400"},
	} {
		source := "# Title\n\n" + tt.byline + "\n"
		issue, err := ParseIssue(strings.NewReader(source))

		if err != nil {
			t.Errorf("%s: ParseIssue() error = %v", tt.byline, err)
			continue
		}
		if issue.Author != tt.author || issue.DateCreated != tt.date {
			t.Errorf("%s: byline = %q %q, want %q %q", tt.byline, issue.Author, issue.DateCreated, tt.author, tt.date)
		}

		// The byline is written back as it was.
		if got := FormatIssue(*issue); !strings.Contains(got, tt.byline) {
			t.Errorf("%s: FormatIssue() = %q, want the byline unchanged", tt.byline, got)
		}
	}

	// A byline needs an email.
	for _, byline := range []string{"[Claude 2026-10-01T09:00:00Z]", "[Alice alice@example.com 2026-10-01T09:00:00Z]", "[alice@example.com]"} {
		if _, err := ParseIssue(strings.NewReader("# Title\n\n" + byline + "\n")); err == nil {
			t.Errorf("%s: ParseIssue() error = nil, want an error", byline)
		}
	}
}

func TestParseIssueBlanksTabsAndNewlines(t *testing.T) {
	source := "# A\ttabbed  title\n\n[author@example.com 2023-05-26T23:35:36-0400]\n\n" +
		"- greeting: hello  \nworld\n" +
		"- farewell: goodbye\n  \t moon\n" +
		"- label: tab\tseparated, two\t\ttabs\n" +
		"- assignees: alice,\n  bob\n"

	issue, err := ParseIssue(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}

	if want := "A tabbed  title"; issue.Title != want {
		t.Errorf("Title = %q, want %q", issue.Title, want)
	}

	// A line break, with the space around it, becomes one space. Each tab
	// becomes a space, and values continued on the next line are split as
	// though they were on one.
	want := map[string][]string{
		"greeting":  {"hello world"},
		"farewell":  {"goodbye moon"},
		"label":     {"tab separated", "two  tabs"},
		"assignees": {"alice", "bob"},
	}

	if len(issue.Metadata) != len(want) {
		t.Fatalf("Metadata = %+v, want %d tags", issue.Metadata, len(want))
	}

	for _, tag := range issue.Metadata {
		if got := models.Canonical(tag.Values); !slices.Equal(got, want[tag.Key]) {
			t.Errorf("%s = %q, want %q", tag.Key, got, want[tag.Key])
		}
	}
}

func TestSingleLine(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"a\tb", "a b"},
		{"a  \nb", "a b"},
		{"a \t\r\n\t b", "a b"},
		{"a\n\nb", "a b"},
		{"a\rb", "a b"},
		{"two  spaces", "two  spaces"},
	}

	for _, tt := range tests {
		if got := singleLine(tt.in); got != tt.want {
			t.Errorf("singleLine(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
