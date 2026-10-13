package models

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

const testBackground = "\x1b[48;5;236m"

// unpainted reports whether any visible text in a painted line follows a
// reset that isn't immediately followed by the background again.
func unpainted(line, bg string) bool {
	for _, loc := range sgrReset.FindAllStringIndex(line, -1) {
		rest := line[loc[1]:]

		if rest != "" && !strings.HasPrefix(rest, bg) {
			return true
		}
	}

	return !strings.HasPrefix(line, bg)
}

func TestExpandTabs(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"\tx", "    x"},
		{"a\tb", "a   b"},
		{"abcd\te", "abcd    e"},
		{"\t\tx", "        x"},
		{"no tabs", "no tabs"},
		// Escape sequences don't count as columns.
		{"\x1b[38;5;39ma\x1b[0m\tb", "\x1b[38;5;39ma\x1b[0m   b"},
	}

	for _, tt := range tests {
		if got := expandTabs(tt.in); got != tt.want {
			t.Errorf("expandTabs(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPaintLine(t *testing.T) {
	// A line like glamour's: styled text, padded with separately styled
	// spaces, each followed by a reset.
	line := "  \x1b[38;5;39mfunc\x1b[0m\x1b[38;5;251m \x1b[0m\t\x1b[38;5;42mmain\x1b[0m" +
		strings.Repeat("\x1b[38;5;252m \x1b[m", 10)
	width := lipgloss.Width(line)

	painted := paintLine(line, width, testBackground)

	if unpainted(painted, testBackground) {
		t.Errorf("paintLine() = %q, has text without the background", painted)
	}
	if !strings.HasSuffix(painted, "\x1b[0m") {
		t.Errorf("paintLine() = %q, want it to end with a reset", painted)
	}
	if strings.Contains(painted, "\t") {
		t.Errorf("paintLine() = %q, want the tab expanded", painted)
	}
	if got := lipgloss.Width(painted); got != width {
		t.Errorf("painted width = %d, want %d", got, width)
	}

	// The text's own colors are kept.
	for _, color := range []string{"\x1b[38;5;39m", "\x1b[38;5;42m"} {
		if !strings.Contains(painted, color) {
			t.Errorf("paintLine() lost the color %q", color)
		}
	}
}

func TestPaintCodeBlocks(t *testing.T) {
	rendered := strings.Join([]string{
		"Text before.  ",
		"  " + codeStartMarker + "code one   ",
		"",
		"  code three  ",
		"  " + codeEndMarker + "           ",
		"Text after.   ",
		"  " + codeStartMarker + codeEndMarker + "          ",
	}, "\n")

	lines := strings.Split(paintCodeBlocks(rendered, testBackground), "\n")

	for i, want := range []bool{false, true, true, true, false, false, false} {
		if got := strings.Contains(lines[i], testBackground); got != want {
			t.Errorf("line %d painted = %v, want %v: %q", i, got, want, lines[i])
		}
	}

	for i, line := range lines {
		if strings.Contains(line, codeStartMarker) || strings.Contains(line, codeEndMarker) {
			t.Errorf("line %d still has a marker: %q", i, line)
		}
	}

	// Lines that had a marker keep the width glamour gave them with it.
	for _, i := range []int{1, 4, 6} {
		if got := lipgloss.Width(lines[i]); got != 14 {
			t.Errorf("line %d width = %d, want 14", i, got)
		}
	}
}

func TestCodeBackground(t *testing.T) {
	if dark, light := codeBackground(true), codeBackground(false); dark == light || !strings.HasPrefix(dark, "\x1b[48;5;") || !strings.HasPrefix(light, "\x1b[48;5;") {
		t.Errorf("codeBackground() = %q dark, %q light, want two different backgrounds", dark, light)
	}
}

func TestCardViewPaintsCodeBlocks(t *testing.T) {
	content := "Text before.\n\n```go\nfunc main() {\n\n\tfmt.Println(\"hi\")\n}\n```\n\nBetween.\n\n```\nno language\n```\n\nText after."
	card := Card{
		DateCreated: "2023-05-27T13:41:50-0400",
		Author:      "author",
		Content:     &content,
	}

	rendered, _ := card.renderContent()
	bg := regexp.MustCompile(`\x1b\[48;5;\d+m`).FindString(rendered)

	if bg == "" {
		t.Fatalf("no code block background in:\n%s", rendered)
	}

	for _, s := range []string{codeStartMarker, codeEndMarker, "\t"} {
		if strings.Contains(rendered, s) {
			t.Errorf("rendered card contains %q", s)
		}
	}

	lines := strings.Split(rendered, "\n")
	width := lipgloss.Width(lines[0])

	for i, line := range lines {
		text := strings.TrimSpace(escapes.ReplaceAllString(line, ""))
		code := text == "func main() {" || text == `fmt.Println("hi")` || text == "}" || text == "no language"

		if code && unpainted(line, bg) {
			t.Errorf("code line %d %q has text without the background", i, text)
		}
		if strings.HasPrefix(text, "Text") || text == "Between." {
			if strings.Contains(line, bg) {
				t.Errorf("text line %d %q is painted", i, text)
			}
		}

		// The block is a rectangle: every line keeps the full width.
		if got := lipgloss.Width(line); got != width {
			t.Errorf("line %d %q width = %d, want %d", i, text, got, width)
		}
	}

	// The blank line inside the Go block is painted too.
	for i := range lines {
		if strings.TrimSpace(escapes.ReplaceAllString(lines[i], "")) == "func main() {" {
			if blank := lines[i+1]; !strings.Contains(blank, bg) {
				t.Errorf("blank line in the code block isn't painted: %q", blank)
			}
		}
	}

	// Syntax highlighting is kept: func is colored differently from main.
	if !regexp.MustCompile(`\x1b\[[0-9;]*38;5;\d+[0-9;]*mfunc`).MatchString(rendered) {
		t.Error("code block lost its syntax highlighting")
	}
}
