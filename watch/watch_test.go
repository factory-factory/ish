package watch

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// watchTemp returns a Watcher for a new temporary directory.
func watchTemp(t *testing.T) (*Watcher, string) {
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
	w, dir := watchTemp(t)

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
	w, dir := watchTemp(t)

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
	w, dir := watchTemp(t)

	for _, name := range []string{".metadata.ish", ".metadata.ish-journal", "config.toml", ".AAA.md.swp", "AAA.md~"} {
		write(t, dir, name)
	}

	if changed(w, 5*delay) {
		t.Error("a change was reported for files that aren't issue files")
	}
}

func TestReportsBurstOnce(t *testing.T) {
	w, dir := watchTemp(t)

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

func TestNewFileReportsOnlyItsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "HEAD")

	w, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { w.Close() })

	// Other files in its directory aren't reported.
	write(t, dir, "ORIG_HEAD")
	write(t, dir, "HEAD.md")

	if changed(w, 5*delay) {
		t.Error("a change was reported for another file")
	}

	// It's reported when it's created, and when it's added to.
	write(t, dir, "HEAD")

	if !changed(w, 5*time.Second) {
		t.Fatal("no change reported for the file's creation")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.WriteString("another line\n"); err != nil {
		t.Fatal(err)
	}

	f.Close()

	if !changed(w, 5*time.Second) {
		t.Error("no change reported for a line added to the file")
	}
}

// gitRepo returns a new git repository with an issue directory in it, and a
// function that runs git commands in it.
func gitRepo(t *testing.T) (string, func(args ...string)) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git isn't installed")
	}

	root := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	git := func(args ...string) {
		t.Helper()

		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=a", "GIT_AUTHOR_EMAIL=a@example.com",
			"GIT_COMMITTER_NAME=a", "GIT_COMMITTER_EMAIL=a@example.com",
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	git("init", "-q", "-b", "main")

	if err := os.Mkdir(filepath.Join(root, ".ish"), 0o755); err != nil {
		t.Fatal(err)
	}

	write(t, filepath.Join(root, ".ish"), "AAA.md")
	git("add", ".")
	git("commit", "-q", "-m", "first")

	return root, git
}

func TestReflog(t *testing.T) {
	root, git := gitRepo(t)

	path, err := Reflog(filepath.Join(root, ".ish"))
	if err != nil {
		t.Fatal(err)
	}

	if want := filepath.Join(root, ".git", "logs", "HEAD"); !sameFile(t, path, want) {
		t.Errorf("Reflog() = %q, want %q", path, want)
	}

	// A worktree has its own.
	tree := filepath.Join(t.TempDir(), "tree")
	git("worktree", "add", "-q", tree)

	path, err = Reflog(filepath.Join(tree, ".ish"))
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(path) != "HEAD" || strings.HasPrefix(path, tree) || sameFile(t, path, filepath.Join(root, ".git", "logs", "HEAD")) {
		t.Errorf("Reflog() in a worktree = %q, want the worktree's own reflog", path)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("Reflog() in a worktree = %q, which doesn't exist: %v", path, err)
	}

	if _, err := Reflog(t.TempDir()); err == nil {
		t.Error("Reflog() outside a repository error = nil, want an error")
	}
}

// sameFile reports whether a and b are the same file.
func sameFile(t *testing.T, a, b string) bool {
	t.Helper()

	ai, err := os.Stat(a)
	if err != nil {
		return false
	}

	bi, err := os.Stat(b)
	if err != nil {
		return false
	}

	return os.SameFile(ai, bi)
}

func TestReflogReportsCommits(t *testing.T) {
	root, git := gitRepo(t)

	path, err := Reflog(filepath.Join(root, ".ish"))
	if err != nil {
		t.Fatal(err)
	}

	w, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { w.Close() })

	git("commit", "-q", "--allow-empty", "-m", "second")

	if !changed(w, 5*time.Second) {
		t.Error("no change reported for a commit")
	}

	git("checkout", "-q", "-b", "other")

	if !changed(w, 5*time.Second) {
		t.Error("no change reported for a checkout")
	}
}

func TestNewFileWithoutItsDirectory(t *testing.T) {
	// In a repository without commits, the reflog's directory doesn't
	// exist yet.
	// For git, and its configuration.
	gitRepo(t)
	empty := filepath.Join(t.TempDir(), "empty")

	cmd := exec.Command("git", "init", "-q", empty)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	path, err := Reflog(empty)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewFile(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("NewFile(%q) error = %v, want one that it doesn't exist", path, err)
	}
}
