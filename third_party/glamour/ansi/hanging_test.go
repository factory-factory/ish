package ansi

// Tests for the local hanging indent patch for ish.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestWrapHanging(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{
			"continuation lines are indented by the marker's width",
			"• " + hangingIndentMark + "one two three four five six",
			14,
			"• one two\n  three four\n  five six",
		},
		{
			"a wider marker",
			"10. " + hangingIndentMark + "one two three four",
			12,
			"10. one two\n    three\n    four",
		},
		{
			"lines without the mark wrap as before",
			"one two three four five six",
			10,
			"one two\nthree four\nfive six",
		},
		{
			"each line is wrapped on its own",
			"• " + hangingIndentMark + "one two three\n  • " + hangingIndentMark + "four five six",
			12,
			"• one two\n  three\n  • four\n    five six",
		},
		{
			"a line that fits is unchanged",
			"• " + hangingIndentMark + "short",
			20,
			"• short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapHanging(tt.in, tt.width)

			if strings.Contains(got, hangingIndentMark) {
				t.Errorf("wrapHanging() left the mark in %q", got)
			}

			// Compare without trailing space, which lipgloss.Wrap may keep.
			var trimmed []string
			for _, line := range strings.Split(got, "\n") {
				trimmed = append(trimmed, strings.TrimRight(line, " "))
			}

			if strings.Join(trimmed, "\n") != tt.want {
				t.Errorf("wrapHanging() =\n%s\nwant\n%s", strings.Join(trimmed, "\n"), tt.want)
			}
		})
	}
}

func TestWrapHangingNarrowerThanMarker(t *testing.T) {
	// With no room after the marker, the line wraps as glamour wraps it
	// without the patch.
	got := wrapHanging("• "+hangingIndentMark+"one two", 2)
	want := lipgloss.Wrap("• one two", 2, wrapBreakpoints)

	if got != want {
		t.Errorf("wrapHanging() = %q, want %q", got, want)
	}
}

func TestWrapHangingStyledMarker(t *testing.T) {
	// The marker's width is measured without its escape sequences.
	marker := "\x1b[1m•\x1b[m "
	got := wrapHanging(marker+hangingIndentMark+"one two three", 10)
	lines := strings.Split(got, "\n")

	if len(lines) != 2 || !strings.HasPrefix(lines[1], "  three") {
		t.Errorf("wrapHanging() = %q, want the continuation indented by two columns", got)
	}
	if ansi.StringWidth(lines[0]) > 10 {
		t.Errorf("first line %q is wider than 10", lines[0])
	}
}
