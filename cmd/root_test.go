package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/factory-factory/ish/config"
	"github.com/factory-factory/ish/state"
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

func TestEnsureDatabase(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".ish")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	good := "# Good\n\n[alice@example.com 2026-10-11T08:59:23-07:00]\n"

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

// commit makes an empty commit by author in the git repository root.
func commit(t *testing.T, root string, author store.User) {
	t.Helper()

	cmd := exec.Command("git", "-C", root, "commit", "-q", "--allow-empty", "-m", "commit by "+author.Name)
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME="+author.Name, "GIT_AUTHOR_EMAIL="+author.Email,
		"GIT_COMMITTER_NAME=ish", "GIT_COMMITTER_EMAIL=ish@example.com",
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func TestUsersFromGitHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git isn't installed")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	root := t.TempDir()
	dir := filepath.Join(root, ".ish")

	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	alice := store.User{Name: "Alice", Email: "alice@example.com"}
	bob := store.User{Name: "Bob", Email: "bob@example.com"}
	commit(t, root, alice)
	commit(t, root, bob)

	// Init indexes every author, and says so.
	var stdout, stderr bytes.Buffer

	if code := runInit(dir, &stdout, &stderr); code != 0 {
		t.Fatalf("runInit() = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Indexed 2 users from the git history\n") {
		t.Errorf("stdout = %q, want the users reported", stdout.String())
	}

	// The commits change no issue files, so there's no history to report.
	if strings.Contains(stdout.String(), "change") {
		t.Errorf("stdout = %q, want no changes reported", stdout.String())
	}

	// Starting ish adds new authors quietly.
	carol := store.User{Name: "Carol", Email: "carol@example.com"}
	commit(t, root, carol)
	stdout.Reset()

	if !ensureDatabase(dir, &stdout, &stderr) || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("ensureDatabase(): output %q %q, want none", stdout.String(), stderr.String())
	}

	users, err := store.Users(dir)

	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, u := range users {
		got = append(got, u.Name+" "+u.Email)
	}

	if want := []string{"Alice alice@example.com", "Bob bob@example.com", "Carol carol@example.com"}; !slices.Equal(got, want) {
		t.Errorf("Users() = %q, want %q", got, want)
	}
}

func TestLoadState(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	if err := os.MkdirAll(filepath.Join(home, ".ish"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ish", "config.toml"), []byte("[user]\nemail = \"alice@example.com\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := loadState(home, dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, _ := st.Snapshot().LookupString(".user.email"); got != "alice@example.com" {
		t.Errorf("email = %q, want the per-user file's", got)
	}

	// A configuration that can't be read is an error.
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("not toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadState(home, dir); err == nil {
		t.Error("loadState() with a bad file error = nil, want an error")
	}
}

func TestCheckUser(t *testing.T) {
	for _, tt := range []struct {
		user    map[string]any
		message string
	}{
		{map[string]any{"name": "Alice", "email": "alice@example.com"}, ""},
		{map[string]any{"name": "Alice"}, "user.email isn't configured. Set it in"},
		{map[string]any{"email": "alice@example.com", "name": "  "}, "user.name isn't configured"},
		{nil, "user.name and user.email aren't configured. Set them in"},
	} {
		cfg := config.Config{}
		if tt.user != nil {
			cfg["user"] = tt.user
		}

		err := checkUser(state.New(cfg))

		switch {
		case tt.message == "" && err != nil:
			t.Errorf("checkUser(%v) error = %v, want none", tt.user, err)
		case tt.message != "" && (err == nil || !strings.Contains(err.Error(), tt.message)):
			t.Errorf("checkUser(%v) error = %v, want one containing %q", tt.user, err, tt.message)
		}
	}
}
