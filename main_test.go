package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/factory-factory/ish/store"
)

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
	good := "# Good\n\n[alice 2026-10-11T08:59:23-07:00]\n"

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

func TestEnsureDatabase(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".ish")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	good := "# Good\n\n[alice 2026-10-11T08:59:23-07:00]\n"

	for name, source := range map[string]string{"AAAAA-11111.md": good, "BBBBB-22222.md": "Not an issue\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A missing database is built with runInit. A file it can't index is
	// reported, but the database can still be used.
	var stdout, stderr bytes.Buffer

	if !ensureDatabase(dir, &stdout, &stderr) {
		t.Fatalf("ensureDatabase() = false, want true (stderr: %s)", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Indexed 1 issue") || !strings.Contains(stderr.String(), "BBBBB-22222.md") {
		t.Errorf("stdout = %q, stderr = %q, want runInit's report", stdout.String(), stderr.String())
	}

	// An existing database is synced, not rebuilt. The file that still can't
	// be indexed is reported again, since it's missing from the database.
	stdout.Reset()
	stderr.Reset()

	if !ensureDatabase(dir, &stdout, &stderr) {
		t.Fatal("ensureDatabase() with a database = false, want true")
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "Couldn't index "+filepath.Join(dir, "BBBBB-22222.md")) {
		t.Errorf("stdout = %q, stderr = %q, want only the bad file reported", stdout.String(), stderr.String())
	}

	// Once it's fixed, it's indexed as a new issue.
	if err := os.WriteFile(filepath.Join(dir, "BBBBB-22222.md"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()

	if !ensureDatabase(dir, &stdout, &stderr) || stdout.String() != "Indexed new issue: BBBBB-22222\n" || stderr.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q, want the fixed file indexed", stdout.String(), stderr.String())
	}

	// With nothing changed, nothing is reported.
	stdout.Reset()
	stderr.Reset()

	if !ensureDatabase(dir, &stdout, &stderr) || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("ensureDatabase() with nothing changed: output %q %q, want none", stdout.String(), stderr.String())
	}

	// A database that can't be created stops ish.
	blocked := filepath.Join(root, "file")

	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if ensureDatabase(filepath.Join(blocked, ".ish"), io.Discard, io.Discard) {
		t.Error("ensureDatabase() under a file = true, want false")
	}
}

func TestEnsureDatabaseRebuildsOldSchema(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ish")

	if code := runInit(dir, io.Discard, io.Discard); code != 0 {
		t.Fatalf("runInit() = %d, want 0", code)
	}

	// Mark the database as coming from an older version of ish.
	db, err := store.Open(filepath.Join(dir, store.DatabaseName))

	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}

	db.Close()

	var stdout, stderr bytes.Buffer

	if !ensureDatabase(dir, &stdout, &stderr) {
		t.Fatalf("ensureDatabase() = false, want true (stderr: %s)", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Rebuilt "+filepath.Join(dir, store.DatabaseName)) {
		t.Errorf("stdout = %q, want the database rebuilt", stdout.String())
	}
	if current, err := store.Current(dir); !current || err != nil {
		t.Errorf("Current() after rebuilding = %v, %v, want true, nil", current, err)
	}
}
