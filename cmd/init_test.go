package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".ish")

	// A new directory.
	var stdout, stderr bytes.Buffer

	if code := runInit(dir, &stdout, &stderr); code != 0 {
		t.Fatalf("runInit() = %d, want 0 (stderr: %s)", code, stderr.String())
	}

	want := "Created " + dir + "/\n" +
		"Added the metadata database to " + filepath.Join(dir, ".gitignore") + "\n" +
		"Created " + filepath.Join(dir, ".metadata.ish") + "\n" +
		"Indexed 0 issues\n"

	if stdout.String() != want || stderr.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q, want stdout %q", stdout.String(), stderr.String(), want)
	}

	// An existing directory with one good and one bad issue.
	good := "# Good\n\n[alice@example.com 2026-10-11T08:59:23-07:00]\n"

	for name, source := range map[string]string{"AAAAA-11111.md": good, "BBBBB-22222.md": "Not an issue\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	stdout.Reset()
	stderr.Reset()

	if code := runInit(dir+"/", &stdout, &stderr); code != 1 {
		t.Errorf("runInit() with a bad issue = %d, want 1", code)
	}

	want = "Rebuilt " + filepath.Join(dir, ".metadata.ish") + "\nIndexed 1 issue\n"

	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	if got := stderr.String(); !strings.HasPrefix(got, "Couldn't index "+filepath.Join(dir, "BBBBB-22222.md")+": ") {
		t.Errorf("stderr = %q, want the bad issue reported", got)
	}

	// A directory that can't be created.
	blocked := filepath.Join(root, "file")

	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	stderr.Reset()

	if code := runInit(filepath.Join(blocked, ".ish"), io.Discard, &stderr); code != 1 || !strings.Contains(stderr.String(), "Couldn't create the metadata database") {
		t.Errorf("runInit() under a file = %d, stderr %q, want 1 and an error", code, stderr.String())
	}
}
