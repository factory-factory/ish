package models

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

// These markers are inserted around references in a card's markdown before
// it's rendered, to find the links in the rendered output. They're private
// use characters, so they can't appear in content.
const (
	linkMarker         = ""
	selectedLinkMarker = ""
	linkEndMarker      = ""
)

// sgrPattern matches an ANSI escape sequence that sets text style.
var sgrPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// activeStyle returns the style escape sequences in s that are still in
// effect at its end: those since the last reset.
func activeStyle(s string) string {
	var active []string

	for _, seq := range sgrPattern.FindAllString(s, -1) {
		if seq == "\x1b[0m" || seq == "\x1b[m" {
			active = active[:0]
			continue
		}

		active = append(active, seq)
	}

	return strings.Join(active, "")
}

// styleLinks styles the text between link markers in rendered markdown as
// links, removing the markers and any styling glamour gave the text. It
// returns the result and the index of the line with the selected link, or
// -1 if there isn't one.
func styleLinks(rendered string) (string, int) {
	if !strings.Contains(rendered, linkEndMarker) {
		return rendered, -1
	}

	lines := strings.Split(rendered, "\n")
	row := -1

	for i, line := range lines {
		var b strings.Builder

		for {
			start, marker := nextLinkMarker(line)
			if start < 0 {
				break
			}

			if marker == selectedLinkMarker {
				row = i
			}

			rest := line[start+len(marker):]
			end := strings.Index(rest, linkEndMarker)

			// A link that's wrapped onto the next line isn't styled.
			if end < 0 {
				b.WriteString(line[:start])
				line = rest
				continue
			}

			style := styles.Link
			if marker == selectedLinkMarker {
				style = styles.LinkSelected
			}

			// The link's own reset would end the style of the text around
			// it, so that style is restored after it.
			before := line[:start]
			restore := activeStyle(b.String() + before + rest[:end])

			b.WriteString(before)
			b.WriteString(style.Render(escapePattern.ReplaceAllString(rest[:end], "")))
			b.WriteString(restore)

			line = rest[end+len(linkEndMarker):]
		}

		b.WriteString(strings.ReplaceAll(line, linkEndMarker, ""))
		lines[i] = b.String()
	}

	return strings.Join(lines, "\n"), row
}

// nextLinkMarker returns the position of the first link marker in s, and the
// marker, or -1 if there isn't one.
func nextLinkMarker(s string) (int, string) {
	link := strings.Index(s, linkMarker)
	selected := strings.Index(s, selectedLinkMarker)

	switch {
	case link < 0 && selected < 0:
		return -1, ""
	case selected < 0 || link >= 0 && link < selected:
		return link, linkMarker
	default:
		return selected, selectedLinkMarker
	}
}

// renderLinks renders plain text containing link markers, with the links in
// link, the selected link in selected, and the rest in base.
func renderLinks(text string, base, link, selected lipgloss.Style) string {
	var b strings.Builder

	for {
		start, marker := nextLinkMarker(text)
		if start < 0 {
			break
		}

		rest := text[start+len(marker):]
		end := strings.Index(rest, linkEndMarker)

		if end < 0 {
			text = text[:start] + rest
			continue
		}

		if start > 0 {
			b.WriteString(base.Render(text[:start]))
		}

		style := link
		if marker == selectedLinkMarker {
			style = selected
		}

		b.WriteString(style.Render(rest[:end]))
		text = rest[end+len(linkEndMarker):]
	}

	if text != "" {
		b.WriteString(base.Render(text))
	}

	return b.String()
}
