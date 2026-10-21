package app

import (
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/factory-factory/ish/models"
)

// now returns the current time. Tests replace it.
var now = time.Now

// gitUserEmail returns git's user.email setting for the repository containing
// dir. Tests replace it.
var gitUserEmail = func(dir string) (string, error) {
	cmd := exec.Command("git", "config", "user.email")
	cmd.Dir = dir

	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// gitUserName returns git's user.name setting for the repository containing
// dir. Tests replace it.
var gitUserName = func(dir string) (string, error) {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = dir

	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// currentUserName returns the name of the user running ish. Tests replace it.
var currentUserName = func() (string, error) {
	u, err := user.Current()

	if err != nil {
		return "", err
	}

	return u.Username, nil
}

// CurrentUser returns the author of new issues and comments created in dir:
// git's user.email, or the name of the user running ish if that isn't set.
// Bylines separate the author from the date with a space, so spaces are
// removed.
func CurrentUser(dir string) string {
	author, err := gitUserEmail(dir)

	if err != nil || author == "" {
		author, err = currentUserName()
	}

	if err != nil || author == "" {
		author = "unknown"
	}

	return strings.Join(strings.Fields(author), "")
}

// CurrentAuthor returns the byline author of new issues and comments
// created in dir: git's user.name as a quoted display name and its
// user.email in angle brackets, as in "Alice" <alice@example.com>, or just
// the email if there's no name. Without an email, it's CurrentUser, which a
// byline doesn't accept, so the issue can't be saved until git's user.email
// is set.
func CurrentAuthor(dir string) string {
	email, err := gitUserEmail(dir)
	if err != nil || strings.TrimSpace(email) == "" {
		return CurrentUser(dir)
	}

	name, err := gitUserName(dir)
	if err != nil {
		name = ""
	}

	return models.Mailbox{Name: strings.TrimSpace(name), Email: strings.TrimSpace(email)}.String()
}
