package models

import (
	"regexp"
	"strings"
)

// User is someone with a display name and an email, from the git history.
type User struct {
	Name  string
	Email string
}

// Directory maps users' emails to their display names and back, so emails
// can be shown as names, and names typed in place of emails.
type Directory struct {
	names  map[string]string
	emails map[string][]string
}

// NewDirectory returns a directory of users. An email with several names
// is shown with the first of them.
func NewDirectory(users []User) *Directory {
	d := &Directory{names: map[string]string{}, emails: map[string][]string{}}

	for _, user := range users {
		if user.Name == "" || user.Email == "" {
			continue
		}

		if _, ok := d.names[user.Email]; !ok {
			d.names[user.Email] = user.Name
		}

		d.emails[user.Name] = append(d.emails[user.Name], user.Email)
	}

	return d
}

// Name returns the display name for email, if it's a user's.
func (d *Directory) Name(email string) (string, bool) {
	if d == nil {
		return "", false
	}

	name, ok := d.names[email]

	return name, ok
}

// Email returns the email for the display name, if exactly one user has it.
// A name that several emails share is ambiguous, and has none.
func (d *Directory) Email(name string) (string, bool) {
	if d == nil {
		return "", false
	}

	emails := d.emails[name]
	if len(emails) != 1 {
		return "", false
	}

	return emails[0], true
}

// mention returns the name to show for the email in angle brackets in text,
// such as <alice@example.com>, if it's a user's.
func (d *Directory) mention(text string) (string, bool) {
	email, ok := bracketedEmail(text)
	if !ok {
		return "", false
	}

	return d.Name(email)
}

// emailPattern matches an email in angle brackets, as in an autolink.
var emailPattern = regexp.MustCompile(`<([^<>@\s]+@[^<>@\s]+)>`)

// StoredValue returns a metadata value as it's stored in the metadata
// database: an email in angle brackets without the brackets, and anything
// else as it is.
func StoredValue(value string) string {
	if email, ok := bracketedEmail(value); ok {
		return email
	}

	return value
}

// AuthorEmail returns the email of a byline's author, without angle
// brackets, or the author as it is if it isn't a mailbox.
func AuthorEmail(author string) string {
	box, err := ParseMailbox(author)
	if err != nil {
		return author
	}

	return box.Email
}

// bracketedEmail returns the email in text, if text is an email in angle
// brackets.
func bracketedEmail(text string) (string, bool) {
	if m := emailPattern.FindStringSubmatch(text); m != nil && m[0] == text {
		return m[1], true
	}

	return "", false
}

// Show returns a metadata value from the metadata database as it's shown: a
// user's display name after mentionPrefix in place of their email, or the
// value as it is. The database stores emails without angle brackets, but
// they're accepted too.
func (d *Directory) Show(value string) string {
	if name, ok := d.Name(StoredValue(value)); ok {
		return mentionPrefix + name
	}

	return StoredValue(value)
}

// author returns how a byline's author is shown: the user's newest display
// name after mentionPrefix, if the author's email is a user's, or the
// byline's own display name if it has one. Otherwise, the author's email is
// shown on its own, or any other author as it is.
func (d *Directory) author(author string) string {
	box, err := ParseMailbox(author)
	if err != nil {
		return author
	}

	if name, ok := d.Name(box.Email); ok {
		return mentionPrefix + name
	}

	if box.Name != "" {
		return mentionPrefix + box.Name
	}

	return box.Email
}

// mentionPrefix starts a display name shown or typed in place of an email.
const mentionPrefix = "@"

// resolve returns the canonical text of a value as it's typed: a display
// name after mentionPrefix becomes the user's email in angle brackets, if
// the name is a single user's. Anything else is kept as it's typed.
func (d *Directory) resolve(text string) string {
	text = strings.TrimSpace(text)

	if name, ok := strings.CutPrefix(text, mentionPrefix); ok {
		if email, ok := d.Email(name); ok {
			return "<" + email + ">"
		}
	}

	return text
}
