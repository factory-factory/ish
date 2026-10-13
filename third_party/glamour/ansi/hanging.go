package ansi

// This file is a local patch to glamour for ish: it wraps list items with a
// hanging indent, so a wrapped line starts under the item's text rather than
// under its bullet, number, or checkbox.

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// hangingIndentMark is written after a list item's marker, to show where the
// item's text starts. It's a private use character, so it can't appear in
// rendered markdown, and it's removed when the list is wrapped.
const hangingIndentMark = ""

// wrapBreakpoints are the characters glamour breaks lines after, besides
// spaces.
const wrapBreakpoints = " ,.;-+|"

// wrapHanging wraps each line of s to width, like lipgloss.Wrap. A line that
// contains hangingIndentMark is wrapped with a hanging indent: the text after
// the mark is wrapped to the width left after the text before it, and each
// continuation line is indented by that much. The mark is removed.
func wrapHanging(s string, width int) string {
	lines := strings.Split(s, "\n")

	for i, line := range lines {
		before, after, found := strings.Cut(line, hangingIndentMark)

		indent := ansi.StringWidth(before)

		if !found || width-indent < 1 {
			lines[i] = lipgloss.Wrap(strings.Replace(line, hangingIndentMark, "", 1), width, wrapBreakpoints)
			continue
		}

		wrapped := strings.Split(lipgloss.Wrap(after, width-indent, wrapBreakpoints), "\n")

		for j := range wrapped {
			if j == 0 {
				wrapped[j] = before + wrapped[j]
			} else {
				wrapped[j] = strings.Repeat(" ", indent) + wrapped[j]
			}
		}

		lines[i] = strings.Join(wrapped, "\n")
	}

	return strings.Join(lines, "\n")
}
