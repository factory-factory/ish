package models

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Mailbox is a byline's author: an email, and optionally a display name.
type Mailbox struct {
	Name  string
	Email string
}

// errNoEmail is returned for a byline's author that isn't a mailbox.
var errNoEmail = errors.New("a byline's author must be an email address, as in alice@example.com, or a name and an email address in angle brackets, as in Alice <alice@example.com>")

// SplitMailbox reads the mailbox at the start of text, and returns it, the
// text of the mailbox as it's written, and the rest of text after it,
// without the space between them. The mailbox is a subset of RFC 5322's
// mailbox (section 3.4): either an addr-spec on its own, or a display name
// of words and quoted strings followed by an addr-spec in angle brackets.
// Comments and obsolete forms aren't accepted.
func SplitMailbox(text string) (Mailbox, string, string, error) {
	p := &mailboxParser{text: text}

	box, err := p.parse()
	if err != nil {
		return Mailbox{}, "", "", err
	}

	written := text[:p.pos]
	rest := strings.TrimLeft(text[p.pos:], " \t")

	if len(rest) == len(text[p.pos:]) && rest != "" {
		return Mailbox{}, "", "", errNoEmail
	}

	return box, written, rest, nil
}

// ParseMailbox parses text as a mailbox, as SplitMailbox does, with nothing
// after it.
func ParseMailbox(text string) (Mailbox, error) {
	box, _, rest, err := SplitMailbox(text)
	if err != nil {
		return Mailbox{}, err
	}

	if rest != "" {
		return Mailbox{}, errNoEmail
	}

	return box, nil
}

// String returns the mailbox as it's written in a byline: the display name
// as a quoted string and the email in angle brackets, or the email on its
// own if there's no name.
func (m Mailbox) String() string {
	if m.Name == "" {
		return m.Email
	}

	quoted := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(m.Name)

	return `"` + quoted + `" <` + m.Email + `>`
}

type mailboxParser struct {
	text string
	pos  int
}

func (p *mailboxParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.text[p.pos:])
	return r
}

func (p *mailboxParser) done() bool {
	return p.pos >= len(p.text)
}

func (p *mailboxParser) skipSpace() {
	for !p.done() && (p.peek() == ' ' || p.peek() == '\t') {
		p.pos++
	}
}

// parse reads a name-addr or an addr-spec.
func (p *mailboxParser) parse() (Mailbox, error) {
	start := p.pos

	// An addr-spec on its own, with no display name.
	if email, ok := p.addrSpec(); ok && (p.done() || p.peek() == ' ' || p.peek() == '\t') {
		return Mailbox{Email: email}, nil
	}

	p.pos = start

	// An addr-spec in angle brackets, with or without a display name.
	var words []string

	for {
		p.skipSpace()

		if p.done() {
			return Mailbox{}, errNoEmail
		}

		if p.peek() == '<' {
			break
		}

		word, ok := p.word()
		if !ok {
			return Mailbox{}, errNoEmail
		}

		words = append(words, word)
	}

	p.pos++ // <

	email, ok := p.addrSpec()
	if !ok || p.done() || p.peek() != '>' {
		return Mailbox{}, errNoEmail
	}

	p.pos++ // >

	return Mailbox{Name: strings.Join(words, " "), Email: email}, nil
}

// word reads an atom or a quoted string, returning its text.
func (p *mailboxParser) word() (string, bool) {
	if p.peek() == '"' {
		return p.quotedString()
	}

	return p.atom(false)
}

// isAtext reports whether r may appear in an atom. Non-ASCII letters are
// accepted, as RFC 6532 allows.
func isAtext(r rune) bool {
	if r > unicode.MaxASCII {
		return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r)
	}

	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
		strings.ContainsRune("!#$%&'*+-/=?^_`{|}~", r)
}

// atom reads an atom, or a dot-atom if dots is set: atoms separated by
// single dots.
func (p *mailboxParser) atom(dots bool) (string, bool) {
	start := p.pos

	for !p.done() {
		r, size := utf8.DecodeRuneInString(p.text[p.pos:])

		if isAtext(r) {
			p.pos += size
			continue
		}

		// A dot must be between atom characters.
		if dots && r == '.' && p.pos > start {
			if next, _ := utf8.DecodeRuneInString(p.text[p.pos+1:]); isAtext(next) {
				p.pos++
				continue
			}
		}

		break
	}

	return p.text[start:p.pos], p.pos > start
}

// quotedString reads a quoted string, returning its text without the quotes
// and with escapes removed.
func (p *mailboxParser) quotedString() (string, bool) {
	start := p.pos
	p.pos++ // "

	var b strings.Builder

	for !p.done() {
		r, size := utf8.DecodeRuneInString(p.text[p.pos:])
		p.pos += size

		switch r {
		case '"':
			return b.String(), true

		case '\\':
			if p.done() {
				p.pos = start
				return "", false
			}

			escaped, size := utf8.DecodeRuneInString(p.text[p.pos:])
			p.pos += size
			b.WriteRune(escaped)

		default:
			b.WriteRune(r)
		}
	}

	p.pos = start

	return "", false
}

// addrSpec reads an email: a dot-atom or quoted string, @, and a dot-atom
// domain.
func (p *mailboxParser) addrSpec() (string, bool) {
	start := p.pos

	if p.peek() == '"' {
		if _, ok := p.quotedString(); !ok {
			return "", false
		}
	} else if _, ok := p.atom(true); !ok {
		return "", false
	}

	if p.done() || p.peek() != '@' {
		p.pos = start
		return "", false
	}

	p.pos++ // @

	if _, ok := p.atom(true); !ok {
		p.pos = start
		return "", false
	}

	return p.text[start:p.pos], true
}
