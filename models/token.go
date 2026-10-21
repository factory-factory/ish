package models

// Token is a metadata value: its canonical text, which is written to the
// issue file and the metadata database, and the text it's displayed as, if
// that's different. A user's email in angle brackets is displayed as their
// name.
type Token struct {
	Display   string
	Canonical string
}

// Tokens returns tokens for canonical values, with nothing to display in
// their place.
func Tokens(values ...string) []Token {
	tokens := make([]Token, len(values))

	for i, v := range values {
		tokens[i] = Token{Canonical: v}
	}

	return tokens
}

// Canonical returns the canonical text of each token.
func Canonical(tokens []Token) []string {
	values := make([]string, len(tokens))

	for i, t := range tokens {
		values[i] = t.Canonical
	}

	return values
}

// Text returns the token as it's shown and edited: its display name after
// mentionPrefix, or its canonical text if it has no display.
func (t Token) Text() string {
	if t.Display != "" {
		return mentionPrefix + t.Display
	}

	return t.Canonical
}

// tokenFor returns the token for a value's canonical text, displayed as the
// user's name if it's the email of a user in d.
func (d *Directory) tokenFor(canonical string) Token {
	name, _ := d.mention(canonical)

	return Token{Display: name, Canonical: canonical}
}
