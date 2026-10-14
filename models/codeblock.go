package models

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

// Code blocks are painted with a grey background after glamour renders them.
// Glamour writes these private use characters before and after each code
// block, through the code block style's BlockPrefix and BlockSuffix, which
// marks the lines to paint. They can't appear in content.
const (
	codeStartMarker = ""
	codeEndMarker   = ""
)

// codeTabWidth is the number of columns between tab stops in code blocks.
// Terminals don't paint the background under a tab, so tabs are expanded to
// spaces.
const codeTabWidth = 4

// Code block backgrounds, as 256-color palette indexes: dark grey on dark
// terminals, and light grey on light ones.
const (
	codeBackgroundDark  = 0
	codeBackgroundLight = 254
)

// sgrReset matches SGR sequences that reset all attributes, which would turn
// the background off: \x1b[m, \x1b[0m, and \x1b[0;...m.
var sgrReset = regexp.MustCompile(`\x1b\[0*(;[0-9;]*)?m`)

// trailingSpace matches spaces at the end of a line, and the escape sequences
// among them.
var trailingSpace = regexp.MustCompile(`(\x1b\[[0-9;]*m| )+$`)

// codeBackground returns the SGR sequence for the code block background.
func codeBackground(dark bool) string {
	color := codeBackgroundLight
	if dark {
		color = codeBackgroundDark
	}

	return fmt.Sprintf("\x1b[48;5;%dm", color)
}

// paintCodeBlocks paints each code block in glamour's rendered output, which
// is marked with codeStartMarker and codeEndMarker, with the background
// sequence bg, and removes the markers. Glamour pads each line to the full
// width, so a painted block is a solid rectangle.
func paintCodeBlocks(rendered, bg string) string {
	lines := strings.Split(rendered, "\n")
	inBlock := false

	for i, line := range lines {
		starts := strings.Contains(line, codeStartMarker)
		ends := strings.Contains(line, codeEndMarker)

		// Glamour counted each marker as a column when it padded the line,
		// so the line keeps the width it had with its marker.
		width := lipgloss.Width(line)
		line = strings.NewReplacer(codeStartMarker, "", codeEndMarker, "").Replace(line)

		switch {
		case starts && ends:
			// An empty code block has nothing to paint.
			line = padLine(line, width)
		case starts:
			inBlock = true
			line = paintLine(line, width, bg)
		case ends:
			// The end marker is on the line after the code.
			inBlock = false
			line = padLine(line, width)
		case inBlock:
			line = paintLine(line, width, bg)
		}

		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// padLine pads line with spaces to width.
func padLine(line string, width int) string {
	return line + strings.Repeat(" ", max(width-lipgloss.Width(line), 0))
}

// paintLine paints a line of a code block with the background bg, padded to
// width. Tabs are expanded to spaces, so they're painted too.
func paintLine(line string, width int, bg string) string {
	// Expanding tabs widens the line, so the padding glamour added is
	// replaced to keep it the same width.
	line = padLine(trailingSpace.ReplaceAllString(expandTabs(line), ""), width)

	// A reset turns the background off, so it's turned back on after each.
	painted := sgrReset.ReplaceAllStringFunc(line, func(reset string) string {
		return reset + bg
	})

	return bg + painted + "\x1b[0m"
}

// expandTabs replaces each tab in line with spaces to the next tab stop,
// counting columns without escape sequences.
func expandTabs(line string) string {
	if !strings.Contains(line, "\t") {
		return line
	}

	var b strings.Builder
	column := 0

	for i := 0; i < len(line); {
		if line[i] == '\x1b' {
			// Copy the escape sequence through its final byte.
			end := i + 1
			for end < len(line) && (line[end] < '@' || line[end] > '~' || end == i+1) {
				end++
			}

			b.WriteString(line[i:min(end+1, len(line))])
			i = end + 1

			continue
		}

		r, size := utf8.DecodeRuneInString(line[i:])

		if r == '\t' {
			spaces := codeTabWidth - column%codeTabWidth
			b.WriteString(strings.Repeat(" ", spaces))
			column += spaces
		} else {
			b.WriteRune(r)
			column += lipgloss.Width(string(r))
		}

		i += size
	}

	return b.String()
}
