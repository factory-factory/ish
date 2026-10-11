package parse

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNewIssueID(t *testing.T) {
	dir := t.TempDir()
	seen := map[string]bool{}

	for range 100 {
		id, err := NewIssueID(dir)

		if err != nil {
			t.Fatalf("NewIssueID() error = %v", err)
		}
		if len(id) != idLength {
			t.Errorf("NewIssueID() = %q, want %d characters", id, idLength)
		}
		if strings.Trim(id, idAlphabet) != "" {
			t.Errorf("NewIssueID() = %q, want only characters from %q", id, idAlphabet)
		}
		if seen[id] {
			t.Errorf("NewIssueID() returned %q twice", id)
		}

		seen[id] = true
	}
}

func TestNewIssueIDSkipsUsedIDs(t *testing.T) {
	dir := writeIssues(t, map[string]string{"0000000000": header})

	// The first ID read is in use, so the next one is used.
	random := bytes.NewReader(append(make([]byte, idLength), bytes.Repeat([]byte{33}, idLength)...))
	id, err := newIssueID(dir, random)

	if err != nil {
		t.Fatalf("newIssueID() error = %v", err)
	}
	if id != "1111111111" {
		t.Errorf("newIssueID() = %q, want %q", id, "1111111111")
	}
}

func TestNewIssueIDErrors(t *testing.T) {
	dir := writeIssues(t, map[string]string{"0000000000": header})

	// Every ID read is in use.
	random := bytes.NewReader(make([]byte, idLength*maxIDAttempts))

	if _, err := newIssueID(dir, random); err == nil {
		t.Error("newIssueID() with every ID in use: error = nil, want an error")
	}

	// The random source fails.
	if _, err := newIssueID(dir, failingReader{}); err == nil {
		t.Error("newIssueID() with a failing random source: error = nil, want an error")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("no randomness")
}
