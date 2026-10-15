package models

import (
	"reflect"
	"strings"
	"testing"
)

// knownIssues reports the issues that tests refer to.
func knownIssues(id string) bool {
	switch id {
	case "AAAAA-11111", "BBBBB-22222", "ABC", "ABCDEFGHIJKLMNOPQRST", "ABCDEFGHIJKLMNOPQRSTU", "AB":
		return true
	}

	return false
}

// ids returns the IDs of refs, and checks that each one's offsets cover # and
// its ID in s.
func ids(t *testing.T, s string, refs []reference) []string {
	t.Helper()

	var got []string

	for _, ref := range refs {
		if s[ref.start:ref.end] != "#"+ref.id {
			t.Errorf("reference %+v covers %q in %q, want %q", ref, s[ref.start:ref.end], s, "#"+ref.id)
		}

		got = append(got, ref.id)
	}

	return got
}

func TestFindReferences(t *testing.T) {
	tests := []struct {
		text string
		want []string
	}{
		{"#AAAAA-11111", []string{"AAAAA-11111"}},
		{"blocked by #AAAAA-11111 and #BBBBB-22222", []string{"AAAAA-11111", "BBBBB-22222"}},

		// Trailing punctuation is left out.
		{"see #AAAAA-11111.", []string{"AAAAA-11111"}},
		{"(#AAAAA-11111)", []string{"AAAAA-11111"}},
		{"#AAAAA-11111, #BBBBB-22222;", []string{"AAAAA-11111", "BBBBB-22222"}},
		{`"#AAAAA-11111"?!`, []string{"AAAAA-11111"}},
		{"#AAAAA-11111).", []string{"AAAAA-11111"}},

		// Other characters after the ID make it a different word.
		{"#AAAAA-11111's", nil},
		{"#AAAAA-11111x", nil},

		// Only issues that exist, matched exactly.
		{"#CCCCC-33333", nil},
		{"#aaaaa-11111", nil},

		// IDs are 3 to 20 characters long.
		{"#ABC", []string{"ABC"}},
		{"#AB", nil},
		{"#ABCDEFGHIJKLMNOPQRST", []string{"ABCDEFGHIJKLMNOPQRST"}},
		{"#ABCDEFGHIJKLMNOPQRSTU", nil},

		// # inside a word doesn't start a reference.
		{"issue#AAAAA-11111", nil},
		{"9#AAAAA-11111", nil},
		{"##AAAAA-11111", []string{"AAAAA-11111"}},

		{"# heading", nil},
		{"#", nil},
		{"", nil},
	}

	for _, tt := range tests {
		if got := ids(t, tt.text, findReferences(tt.text, knownIssues)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("findReferences(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}

	if refs := findReferences("#AAAAA-11111", nil); refs != nil {
		t.Errorf("findReferences() without a lookup = %v, want none", refs)
	}
}

func TestContentReferences(t *testing.T) {
	source := strings.Join([]string{
		"## About #AAAAA-11111",
		"",
		"Blocked by #BBBBB-22222, see `#AAAAA-11111` and *#ABC*.",
		"",
		"```",
		"#AAAAA-11111 in a fenced block",
		"```",
		"",
		"    #BBBBB-22222 in an indented block",
		"",
		"- [ ] item for #ABC",
		"- [link to #BBBBB-22222](#AAAAA-11111)",
		"",
		`<span title="#AAAAA-11111">text</span>`,
	}, "\n")

	want := []string{"AAAAA-11111", "BBBBB-22222", "ABC", "ABC", "BBBBB-22222"}

	if got := ids(t, source, contentReferences(source, knownIssues)); !reflect.DeepEqual(got, want) {
		t.Errorf("contentReferences() = %v, want %v", got, want)
	}
}
