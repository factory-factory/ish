package main

import "testing"

func TestIssueIdFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"issues/01AN4Z07BY.md", "01AN4Z07BY"},
		{"./issues/79KA1307SR.md", "79KA1307SR"},
		{"/home/user/project/issues/D9VDRVEVEB.md", "D9VDRVEVEB"},
		{"01AN4Z07BY.md", "01AN4Z07BY"},
		{"01AN4Z07BY", "01AN4Z07BY"},
	}

	for _, tt := range tests {
		if got := issueIdFromPath(tt.path); got != tt.want {
			t.Errorf("issueIdFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
