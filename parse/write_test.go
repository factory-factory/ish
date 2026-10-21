package parse

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/factory-factory/ish/models"
)

const fullIssue = `# Title

[author@example.com 2023-05-26T23:35:36-0400]

- state: in progress
- assignees: alice, bob

---

[carol@example.com 2023-05-27T13:41:50-0400]

- [ ] Parent
  - [ ] Child

  ` + "```" + `
  code
  ` + "```" + `

---

[dave@example.com 2023-05-28 11:55:23 -0400]

---

[erin@example.com 2023-06-04T18:37:55-04:00]

Last.
`

func TestFormatIssueRoundTrip(t *testing.T) {
	issue, err := ParseIssue(strings.NewReader(fullIssue))

	if err != nil {
		t.Fatalf("ParseIssue() error = %v", err)
	}

	if got := FormatIssue(*issue); got != fullIssue {
		t.Errorf("FormatIssue() =\n%s\nwant\n%s", got, fullIssue)
	}
}

func TestFormatIssueSampleFiles(t *testing.T) {
	issues, err := ListIssues(filepath.Join("..", IssuesDir))

	if err != nil {
		t.Fatal(err)
	}

	for _, want := range issues {
		got, err := ParseIssue(strings.NewReader(FormatIssue(want)))

		if err != nil {
			t.Errorf("%s: ParseIssue(FormatIssue()) error = %v", want.Id, err)
			continue
		}

		if err := checkRoundTrip(want, FormatIssue(want)); err != nil {
			t.Errorf("%s: %v", want.Id, err)
		}
		if got.Author != want.Author || got.DateCreated != want.DateCreated {
			t.Errorf("%s: byline = %q %q, want %q %q",
				want.Id, got.Author, got.DateCreated, want.Author, want.DateCreated)
		}
	}
}

// readIssue parses the issue file for id in dir.
func readIssue(t *testing.T, dir, id string) *models.Issue {
	t.Helper()

	issue, err := LoadIssue(dir, id)

	if err != nil {
		t.Fatal(err)
	}

	return issue
}

func TestSaveIssue(t *testing.T) {
	dir := writeIssues(t, map[string]string{"ABC": fullIssue})
	path := filepath.Join(dir, "ABC.md")

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	issue := readIssue(t, dir, "ABC")
	issue.Title = "New title"
	issue.Metadata[0].Values = models.Tokens("closed")
	content := "Edited."
	issue.Conversation[1].Content = &content

	if err := SaveIssue(dir, *issue); err != nil {
		t.Fatalf("SaveIssue() error = %v", err)
	}

	saved := readIssue(t, dir, "ABC")

	if saved.Title != "New title" || !slices.Equal(models.Canonical(saved.Metadata[0].Values), []string{"closed"}) {
		t.Errorf("saved title, state = %q, %q, want %q, [closed]", saved.Title, saved.Metadata[0].Values, "New title")
	}
	if got := *saved.Conversation[1].Content; got != "Edited." {
		t.Errorf("saved comment = %q, want %q", got, "Edited.")
	}
	if got := *saved.Conversation[0].Content; got != *issue.Conversation[0].Content {
		t.Errorf("unedited comment = %q, want it unchanged", got)
	}

	info, err := os.Stat(path)

	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want the original 0600", info.Mode().Perm())
	}

	// The temporary file used to replace the issue file is gone.
	entries, err := os.ReadDir(dir)

	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory has %d entries, want only ABC.md", len(entries))
	}
}

func TestSaveIssueRefusesUnreadableIssues(t *testing.T) {
	tests := []struct {
		name string
		edit func(*models.Issue)
	}{
		{"comment with a horizontal rule", func(issue *models.Issue) {
			content := "Before\n\n---\n\nAfter"
			issue.Conversation[0].Content = &content
		}},
		{"title with a closing heading sequence", func(issue *models.Issue) {
			issue.Title = "Title ##"
		}},
		{"tag with no values", func(issue *models.Issue) {
			issue.Metadata[0].Values = nil
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeIssues(t, map[string]string{"ABC": fullIssue})
			issue := readIssue(t, dir, "ABC")
			tt.edit(issue)

			if err := SaveIssue(dir, *issue); err == nil {
				t.Fatal("SaveIssue() error = nil, want an error")
			}

			source, err := os.ReadFile(filepath.Join(dir, "ABC.md"))

			if err != nil {
				t.Fatal(err)
			}
			if string(source) != fullIssue {
				t.Errorf("file changed after a refused save:\n%s", source)
			}
		})
	}
}

func TestFormatNewIssue(t *testing.T) {
	created := time.Date(2026, 10, 10, 23, 7, 0, 0, time.FixedZone("", -4*60*60))
	issue := models.NewIssue("ABCDEFGHJK", "alice@example.com", created)

	want := "# \n\n[alice@example.com 2026-10-10T23:07:00-04:00]\n\n---\n\n[alice@example.com 2026-10-10T23:07:00-04:00]\n"

	if got := FormatIssue(issue); got != want {
		t.Errorf("FormatIssue() =\n%q\nwant\n%q", got, want)
	}

	// A new issue can be saved to a file that doesn't exist yet.
	dir := t.TempDir()

	if err := SaveIssue(dir, issue); err != nil {
		t.Fatalf("SaveIssue() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "ABCDEFGHJK.md"))

	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("file mode = %v, want 0644", info.Mode().Perm())
	}

	saved := readIssue(t, dir, "ABCDEFGHJK")

	if saved.Title != "" || saved.Author != "alice@example.com" || len(saved.Metadata) != 0 ||
		len(saved.Conversation) != 1 || *saved.Conversation[0].Content != "" {
		t.Errorf("saved issue = %+v, want the empty new issue", saved)
	}
}

func TestFormatIssueWritesCanonicalValues(t *testing.T) {
	issue, err := ParseIssue(strings.NewReader(fullIssue))

	if err != nil {
		t.Fatal(err)
	}

	// A value shown as a user's name is written as their email.
	issue.Metadata = append(issue.Metadata, models.Tag{
		Key:    "watchers",
		Values: []models.Token{{Display: "Bob", Canonical: "<bob@example.com>"}, {Canonical: "carol"}},
	})

	if got := FormatIssue(*issue); !strings.Contains(got, "- watchers: <bob@example.com>, carol\n") || strings.Contains(got, "Bob") {
		t.Errorf("FormatIssue() =\n%s\nwant the watchers' canonical values", got)
	}
}
