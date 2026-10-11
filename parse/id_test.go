package parse

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIDTimestamp(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"epoch", idEpoch, "00000"},
		{"partial minute", idEpoch.Add(59 * time.Second), "00000"},
		{"one minute", idEpoch.Add(time.Minute), "00001"},
		{"base32 carry", idEpoch.Add(32 * time.Minute), "00010"},
		{"date", time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), "3CRF0"},
		{"other time zone", time.Date(2026, 10, 10, 17, 0, 0, 0, time.FixedZone("", -7*60*60)), "3CRF0"},
		{"last minute", idEpoch.Add((1<<25 - 1) * time.Minute), "ZZZZZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := idTimestamp(tt.time)

			if err != nil {
				t.Fatalf("idTimestamp() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("idTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIDTimestampOutOfRange(t *testing.T) {
	for _, tt := range []time.Time{
		idEpoch.Add(-time.Second),
		idEpoch.Add((1 << 25) * time.Minute),
	} {
		if got, err := idTimestamp(tt); err == nil {
			t.Errorf("idTimestamp(%s) = %q, want an error", tt, got)
		}
	}
}

func TestIDsSortByCreation(t *testing.T) {
	random := bytes.NewReader(bytes.Repeat([]byte{31}, idRandomLength))
	earlier, err := newIssueID(t.TempDir(), time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), random)

	if err != nil {
		t.Fatal(err)
	}

	random = bytes.NewReader(make([]byte, idRandomLength))
	later, err := newIssueID(t.TempDir(), time.Date(2026, 10, 11, 0, 1, 0, 0, time.UTC), random)

	if err != nil {
		t.Fatal(err)
	}

	if earlier >= later {
		t.Errorf("ID %q created earlier sorts after %q", earlier, later)
	}
}

func TestNewIssueID(t *testing.T) {
	dir := t.TempDir()
	seen := map[string]bool{}

	for range 100 {
		before, _ := idTimestamp(time.Now())
		id, err := NewIssueID(dir)
		after, _ := idTimestamp(time.Now())

		if err != nil {
			t.Fatalf("NewIssueID() error = %v", err)
		}
		if len(id) != idLength {
			t.Fatalf("NewIssueID() = %q, want %d characters", id, idLength)
		}

		timestamp, separator, random := id[:idTimeLength], id[idTimeLength:idLength-idRandomLength], id[idLength-idRandomLength:]

		if separator != idSeparator {
			t.Errorf("NewIssueID() = %q, want %q after the timestamp", id, idSeparator)
		}
		if strings.Trim(timestamp+random, idAlphabet) != "" {
			t.Errorf("NewIssueID() = %q, want the timestamp and random part from %q", id, idAlphabet)
		}
		if timestamp != before && timestamp != after {
			t.Errorf("NewIssueID() = %q, want it to start with the current timestamp %q", id, before)
		}
		if seen[id] {
			t.Errorf("NewIssueID() returned %q twice", id)
		}

		seen[id] = true
	}
}

var created = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

func TestNewIssueIDSkipsUsedIDs(t *testing.T) {
	dir := writeIssues(t, map[string]string{"3CRF0-00000": header})

	// The first ID is in use, so the next random characters are used.
	random := bytes.NewReader(append(make([]byte, idRandomLength), bytes.Repeat([]byte{33}, idRandomLength)...))
	id, err := newIssueID(dir, created, random)

	if err != nil {
		t.Fatalf("newIssueID() error = %v", err)
	}
	if id != "3CRF0-11111" {
		t.Errorf("newIssueID() = %q, want %q", id, "3CRF0-11111")
	}
}

func TestNewIssueIDErrors(t *testing.T) {
	dir := writeIssues(t, map[string]string{"3CRF0-00000": header})

	// Every ID is in use.
	random := bytes.NewReader(make([]byte, idRandomLength*maxIDAttempts))

	if _, err := newIssueID(dir, created, random); err == nil {
		t.Error("newIssueID() with every ID in use: error = nil, want an error")
	}

	// The random source fails.
	if _, err := newIssueID(dir, created, failingReader{}); err == nil {
		t.Error("newIssueID() with a failing random source: error = nil, want an error")
	}

	// The time can't be encoded.
	random = bytes.NewReader(make([]byte, idRandomLength))

	if _, err := newIssueID(dir, idEpoch.Add(-time.Minute), random); err == nil {
		t.Error("newIssueID() before the epoch: error = nil, want an error")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("no randomness")
}
