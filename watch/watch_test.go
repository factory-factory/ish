package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newWatcher returns a Watcher for a new temporary directory.
func newWatcher(t *testing.T) (*Watcher, string) {
	t.Helper()

	dir := t.TempDir()

	w, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { w.Close() })

	return w, dir
}

func write(t *testing.T, dir, name string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte("# Title\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// changed reports whether a change is received within wait.
func changed(w *Watcher, wait time.Duration) bool {
	select {
	case <-w.Changes():
		return true
	case <-time.After(wait):
		return false
	}
}

func TestReportsIssueFileChanges(t *testing.T) {
	w, dir := newWatcher(t)

	write(t, dir, "AAA.md")

	if !changed(w, 5*time.Second) {
		t.Fatal("no change reported for a new issue file")
	}

	write(t, dir, "AAA.md")

	if !changed(w, 5*time.Second) {
		t.Fatal("no change reported for a changed issue file")
	}

	if err := os.Remove(filepath.Join(dir, "AAA.md")); err != nil {
		t.Fatal(err)
	}

	if !changed(w, 5*time.Second) {
		t.Fatal("no change reported for a removed issue file")
	}
}

func TestReportsRenamedIssueFiles(t *testing.T) {
	w, dir := newWatcher(t)

	// Editors often save by writing a temporary file, then renaming it.
	write(t, dir, "AAA.md.tmp")

	if err := os.Rename(filepath.Join(dir, "AAA.md.tmp"), filepath.Join(dir, "AAA.md")); err != nil {
		t.Fatal(err)
	}

	if !changed(w, 5*time.Second) {
		t.Fatal("no change reported for a file renamed to an issue file")
	}
}

func TestIgnoresOtherFiles(t *testing.T) {
	w, dir := newWatcher(t)

	for _, name := range []string{".metadata.ish", ".metadata.ish-journal", "config.toml", ".AAA.md.swp", "AAA.md~"} {
		write(t, dir, name)
	}

	if changed(w, 5*delay) {
		t.Error("a change was reported for files that aren't issue files")
	}
}

func TestReportsBurstOnce(t *testing.T) {
	w, dir := newWatcher(t)

	for _, name := range []string{"AAA.md", "BBB.md", "CCC.md", "DDD.md"} {
		write(t, dir, name)
	}

	if !changed(w, 5*time.Second) {
		t.Fatal("no change reported")
	}

	if changed(w, 5*delay) {
		t.Error("a burst of changes was reported more than once")
	}
}

func TestCloseClosesChanges(t *testing.T) {
	dir := t.TempDir()

	w, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if _, ok := <-w.Changes(); ok {
		t.Error("Changes() received a value after Close, want it closed")
	}
}

func TestNewWithMissingDirectory(t *testing.T) {
	if _, err := New(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("New() with a missing directory error = nil, want an error")
	}
}
