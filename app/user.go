package app

import (
	"os/exec"
	"os/user"
	"strings"
	"time"
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
