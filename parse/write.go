package parse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/factory-factory/ish/models"
)

// FormatIssue returns the markdown for an issue, in the format ParseIssue
// reads. Card content is written verbatim.
func FormatIssue(issue models.Issue) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n[%s %s]\n", issue.Title, issue.Author, issue.DateCreated)

	if len(issue.Metadata) > 0 {
		b.WriteString("\n")
	}

	for _, tag := range issue.Metadata {
		fmt.Fprintf(&b, "- %s: %s\n", tag.Key, strings.Join(tag.Values, ", "))
	}

	for _, card := range issue.Conversation {
		fmt.Fprintf(&b, "\n---\n\n[%s %s]\n", card.Author, card.DateCreated)

		if card.Content != nil && *card.Content != "" {
			fmt.Fprintf(&b, "\n%s\n", *card.Content)
		}
	}

	return b.String()
}

// SaveIssue writes an issue to the file for its ID in dir. The file is
// replaced atomically, so it's never left partly written. An issue that
// wouldn't read back the same, such as one with a comment containing a
// horizontal rule, is not written.
func SaveIssue(dir string, issue models.Issue) error {
	source := FormatIssue(issue)

	if err := checkRoundTrip(issue, source); err != nil {
		return err
	}

	path := filepath.Join(dir, issue.Id+".md")
	mode := os.FileMode(0o644)

	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, "."+issue.Id+"-*.md")

	if err != nil {
		return err
	}

	// Removing the temporary file fails harmlessly once it's been renamed.
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(source); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

// checkRoundTrip returns an error if source, formatted from issue, doesn't
// parse back to the same issue.
func checkRoundTrip(issue models.Issue, source string) error {
	parsed, err := ParseIssue(strings.NewReader(source))

	if err != nil {
		return fmt.Errorf("issue can't be written as an ish file: %w", err)
	}

	switch {
	case parsed.Title != issue.Title:
		return fmt.Errorf("title %q can't be written as an ish file", issue.Title)

	case len(parsed.Metadata) != len(issue.Metadata):
		return errors.New("metadata can't be written as an ish file")

	case len(parsed.Conversation) != len(issue.Conversation):
		return errors.New("comments can't be written as an ish file; " +
			"a comment may contain a horizontal rule")
	}

	for i, tag := range issue.Metadata {
		got := parsed.Metadata[i]

		if got.Key != tag.Key || !slices.Equal(got.Values, tag.Values) {
			return fmt.Errorf("metadata %q can't be written as an ish file", tag.Key)
		}
	}

	for i, card := range issue.Conversation {
		want := ""
		if card.Content != nil {
			want = trimContent(*card.Content)
		}

		if *parsed.Conversation[i].Content != want {
			return fmt.Errorf("comment %d can't be written as an ish file", i+1)
		}
	}

	return nil
}
