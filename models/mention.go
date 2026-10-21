package models

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/factory-factory/ish/styles"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// These markers are put around a user's display name, in place of their
// email, in a card's markdown before it's rendered, to find the name in the
// rendered output. They're private use characters, so they can't appear in
// content.
const (
	mentionMarker    = ""
	mentionEndMarker = ""
)

// span is a range of bytes in a card's content, [start, end).
type byteRange struct {
	start, end int
}

// codeRanges returns the ranges of markdown source that are code, or HTML,
// where text is shown as it's written.
func codeRanges(source []byte) []byteRange {
	var ranges []byteRange

	doc := markdown.Parser().Parse(text.NewReader(source))

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.CodeSpan:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					ranges = append(ranges, byteRange{t.Segment.Start, t.Segment.Stop})
				}
			}

			return ast.WalkSkipChildren, nil

		case *ast.RawHTML:
			for i := 0; i < n.Segments.Len(); i++ {
				s := n.Segments.At(i)
				ranges = append(ranges, byteRange{s.Start, s.Stop})
			}

		case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock:
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				s := lines.At(i)
				ranges = append(ranges, byteRange{s.Start, s.Stop})
			}

			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})

	return ranges
}

// inCode reports whether [start, end) overlaps any of the code ranges.
func inCode(code []byteRange, start, end int) bool {
	return slices.ContainsFunc(code, func(r byteRange) bool {
		return start < r.end && r.start < end
	})
}

// mention is a user's email in angle brackets in a card's content, at
// [start, end), and the user's display name.
type mention struct {
	byteRange
	email, name string
}

// mentions returns the users' emails in angle brackets in content, outside
// code, that d has names for.
func (d *Directory) mentions(content string) []mention {
	if d == nil || !strings.Contains(content, "@") {
		return nil
	}

	code := codeRanges([]byte(content))

	var found []mention

	for _, m := range emailPattern.FindAllStringSubmatchIndex(content, -1) {
		if inCode(code, m[0], m[1]) {
			continue
		}

		email := content[m[2]:m[3]]

		if name, ok := d.Name(email); ok {
			found = append(found, mention{byteRange{m[0], m[1]}, email, name})
		}
	}

	return found
}

// showNames returns content as it's edited, with the users' emails in angle
// brackets replaced by their display names after mentionPrefix. An email is
// left as it is if its name is several users', so it can't change when the
// content is saved.
func (d *Directory) showNames(content string) string {
	found := d.mentions(content)

	for i := len(found) - 1; i >= 0; i-- {
		m := found[i]

		if email, ok := d.Email(m.name); ok && email == m.email {
			content = content[:m.start] + mentionPrefix + m.name + content[m.end:]
		}
	}

	return content
}

// saveEmails returns edited content as it's saved, with users' display names
// after mentionPrefix, outside code, replaced by their emails in angle
// brackets. A name several users have is left as it's typed. Where names
// overlap, as with "Ann" and "Ann Lee", the longest is used.
func (d *Directory) saveEmails(content string) string {
	if d == nil || !strings.Contains(content, mentionPrefix) {
		return content
	}

	names := make([]string, 0, len(d.emails))
	for name := range d.emails {
		names = append(names, name)
	}

	slices.SortFunc(names, func(a, b string) int { return len(b) - len(a) })

	code := codeRanges([]byte(content))

	var b strings.Builder

	for i := 0; i < len(content); {
		if name, ok := d.mentionAt(content, i, names, code); ok {
			email, _ := d.Email(name)
			b.WriteString("<" + email + ">")
			i += len(mentionPrefix) + len(name)
			continue
		}

		b.WriteByte(content[i])
		i++
	}

	return b.String()
}

// mentionAt returns the display name typed after mentionPrefix at offset i
// in content, if there is one, from names, longest first. The prefix must
// start a word, and the name end one, so emails and longer names don't
// match, and it must be outside code. Only a name that's a single user's
// counts.
func (d *Directory) mentionAt(content string, i int, names []string, code []byteRange) (string, bool) {
	if !strings.HasPrefix(content[i:], mentionPrefix) {
		return "", false
	}

	if before, _ := utf8.DecodeLastRuneInString(content[:i]); i > 0 && isWordRune(before) {
		return "", false
	}

	rest := content[i+len(mentionPrefix):]

	for _, name := range names {
		if !strings.HasPrefix(rest, name) {
			continue
		}

		if after, _ := utf8.DecodeRuneInString(rest[len(name):]); len(rest) > len(name) && isWordRune(after) {
			continue
		}

		end := i + len(mentionPrefix) + len(name)

		if _, ok := d.Email(name); ok && !inCode(code, i, end) {
			return name, true
		}
	}

	return "", false
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// styleMentions styles the display names between mention markers in rendered
// markdown, removing the markers, and restoring the surrounding style after
// each.
func styleMentions(rendered string) string {
	if !strings.Contains(rendered, mentionMarker) {
		return rendered
	}

	lines := strings.Split(rendered, "\n")

	for i, line := range lines {
		var b strings.Builder

		for {
			start := strings.Index(line, mentionMarker)
			if start < 0 {
				break
			}

			rest := line[start+len(mentionMarker):]
			end := strings.Index(rest, mentionEndMarker)

			// A name that's wrapped onto the next line isn't styled.
			if end < 0 {
				b.WriteString(line[:start])
				line = rest
				continue
			}

			before := line[:start]
			restore := activeStyle(b.String() + before + rest[:end])

			b.WriteString(before)
			b.WriteString(styles.Mention.Render(escapePattern.ReplaceAllString(rest[:end], "")))
			b.WriteString(restore)

			line = rest[end+len(mentionEndMarker):]
		}

		b.WriteString(strings.ReplaceAll(line, mentionEndMarker, ""))
		lines[i] = b.String()
	}

	return strings.Join(lines, "\n")
}
