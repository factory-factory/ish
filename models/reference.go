package models

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// IssueLookup reports whether an issue with the given ID exists, so that a
// reference to it can be shown as a link.
type IssueLookup func(id string) bool

// A reference to another issue is # followed by its ID, which is between
// minReference and maxReference characters long.
const (
	minReference = 3
	maxReference = 20
)

// trailingPunctuation is removed from the end of a reference that isn't an
// issue ID as written, so references can end a sentence or a parenthesis.
const trailingPunctuation = `.,;:!?)]}'"`

// reference is a reference to another issue, at source[start:end], where the
// text starts with # and is followed by the ID.
type reference struct {
	start, end int
	id         string
}

// findReferences returns the references in s to issues that exists reports,
// in order. A reference starts with # at the start of s or after a character
// that isn't a letter or digit, and runs to the next whitespace, without any
// trailing punctuation if that isn't part of an ID. Offsets are bytes from
// the start of s.
func findReferences(s string, exists IssueLookup) []reference {
	if exists == nil {
		return nil
	}

	var refs []reference

	for i := 0; i < len(s); i++ {
		if s[i] != '#' {
			continue
		}

		if before, _ := utf8.DecodeLastRuneInString(s[:i]); i > 0 && (unicode.IsLetter(before) || unicode.IsDigit(before)) {
			continue
		}

		run := s[i+1:]
		if end := strings.IndexFunc(run, unicode.IsSpace); end >= 0 {
			run = run[:end]
		}

		id, ok := referencedID(run, exists)
		if !ok {
			continue
		}

		end := i + 1 + len(id)
		refs = append(refs, reference{start: i, end: end, id: id})
		i = end - 1
	}

	return refs
}

// referencedID returns the issue ID that run, the text after a #, refers to:
// run itself if it's an issue ID, or run without trailing punctuation.
func referencedID(run string, exists IssueLookup) (string, bool) {
	for _, id := range []string{run, strings.TrimRight(run, trailingPunctuation)} {
		if n := utf8.RuneCountInString(id); n >= minReference && n <= maxReference && exists(id) {
			return id, true
		}
	}

	return "", false
}

// markdown parses card content the way glamour does, to find the text that
// will be rendered as text, rather than code.
var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM, extension.DefinitionList))

// contentReferences returns the references in markdown source to issues that
// exists reports, in order. References in code spans and code blocks are
// left out, as are link destinations and HTML.
func contentReferences(source string, exists IssueLookup) []reference {
	if exists == nil || !strings.Contains(source, "#") {
		return nil
	}

	src := []byte(source)
	doc := markdown.Parser().Parse(text.NewReader(src))

	var refs []reference

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.CodeSpan, *ast.RawHTML:
			return ast.WalkSkipChildren, nil

		case *ast.Text:
			segment := n.Segment

			for _, ref := range findReferences(string(segment.Value(src)), exists) {
				ref.start += segment.Start
				ref.end += segment.Start
				refs = append(refs, ref)
			}
		}

		return ast.WalkContinue, nil
	})

	// Text is visited in document order, but sort in case an extension
	// produces nodes out of order.
	slices.SortFunc(refs, func(a, b reference) int { return a.start - b.start })

	return refs
}
