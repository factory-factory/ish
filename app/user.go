package app

import (
	"time"

	"github.com/factory-factory/ish/models"
)

// now returns the current time. Tests replace it.
var now = time.Now

// author returns the byline author of new issues and comments: the user's
// name and email from a snapshot of the application state, as in
// "Alice" <alice@example.com>, or just the email if there's no name. ish
// doesn't start without both; see checkUser in main.
func (m Model) author() string {
	snap := m.State().Snapshot()
	name, _ := snap.LookupString(".user.name")
	email, _ := snap.LookupString(".user.email")

	return models.Mailbox{Name: name, Email: email}.String()
}
