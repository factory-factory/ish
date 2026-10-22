package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeConfig writes a configuration file at path, creating its directory.
func writeConfig(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAppliesProjectOverUser(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()

	writeConfig(t, filepath.Join(home, ".ish", FileName), `
editor = "vi"
labels = ["a", "b"]

[user]
name = "Alice"
email = "alice@example.com"

[ui.colors]
accent = 4
border = 11
`)
	writeConfig(t, filepath.Join(dir, FileName), `
labels = ["c"]

[user]
email = "alice@work.example"

[ui.colors]
accent = 5

[[remotes]]
name = "origin"
url = "git@example.com:ish.git"
`)

	cfg, err := Load(home, dir)
	if err != nil {
		t.Fatal(err)
	}

	want := Config{
		"editor": "vi",
		// An array in the project's file replaces the user's.
		"labels": []any{"c"},
		// Tables are merged key by key.
		"user": map[string]any{"name": "Alice", "email": "alice@work.example"},
		"ui":   map[string]any{"colors": map[string]any{"accent": int64(5), "border": int64(11)}},
		// An array of tables is a slice of maps.
		"remotes": []any{map[string]any{"name": "origin", "url": "git@example.com:ish.git"}},
	}

	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %#v, want %#v", cfg, want)
	}
}

func TestLoadWithoutFiles(t *testing.T) {
	cfg, err := Load(t.TempDir(), t.TempDir())

	if err != nil || !reflect.DeepEqual(cfg, Defaults()) {
		t.Errorf("Load() = %v, %v, want the defaults", cfg, err)
	}

	// Without a home directory, only the project's file is read.
	dir := t.TempDir()
	writeConfig(t, filepath.Join(dir, FileName), `x = 1`)

	if cfg, err := Load("", dir); err != nil || cfg["x"] != int64(1) {
		t.Errorf("Load() without a home = %v, %v, want the project's file", cfg, err)
	}
}

func TestLoadReportsBadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeConfig(t, path, "this isn't toml")

	if _, err := Load(t.TempDir(), dir); err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("Load() error = %v, want one naming %s", err, path)
	}
}

func TestLookup(t *testing.T) {
	cfg := Config{
		"user":    map[string]any{"email": "alice@example.com", "tags": []any{"x", "y"}},
		"remotes": []any{map[string]any{"url": "git@example.com:ish.git"}},
		"count":   int64(3),
	}

	for _, tt := range []struct {
		path string
		want any
		ok   bool
	}{
		{".user.email", "alice@example.com", true},
		{"user.email", "alice@example.com", true},
		{".user", map[string]any{"email": "alice@example.com", "tags": []any{"x", "y"}}, true},
		{".user.tags.1", "y", true},
		{".remotes.0.url", "git@example.com:ish.git", true},
		{".count", int64(3), true},

		{".user.name", nil, false},
		{".user.tags.2", nil, false},
		{".user.tags.x", nil, false},
		{".user.tags.-1", nil, false},
		{".count.x", nil, false},
		{".missing.x", nil, false},
	} {
		got, ok := cfg.Lookup(tt.path)

		if ok != tt.ok || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Lookup(%q) = %#v, %v, want %#v, %v", tt.path, got, ok, tt.want, tt.ok)
		}
	}

	// The whole configuration is at the empty path.
	if got, ok := cfg.Lookup("."); !ok || !reflect.DeepEqual(got, map[string]any(cfg)) {
		t.Errorf("Lookup(.) = %#v, %v, want the configuration", got, ok)
	}
}

func TestLookupReturnsCopy(t *testing.T) {
	cfg := Config{"user": map[string]any{"tags": []any{"x"}}}

	user, _ := cfg.Lookup(".user")
	user.(map[string]any)["tags"].([]any)[0] = "changed"
	user.(map[string]any)["name"] = "added"

	if got, _ := cfg.Lookup(".user.tags.0"); got != "x" {
		t.Errorf("tags.0 = %v after changing a looked-up copy, want x", got)
	}
	if _, ok := cfg.Lookup(".user.name"); ok {
		t.Error("a key added to a looked-up copy is in the configuration")
	}
}

// TestMain keeps the tests from reading git's settings on the machine
// they're run on. Tests of git's settings replace gitConfig themselves.
func TestMain(m *testing.M) {
	gitConfig = func(dir, key string) (string, error) {
		return "", errors.New("no git settings in tests")
	}

	os.Exit(m.Run())
}

// stubGit makes gitConfig return settings, and records the keys it's asked
// for.
func stubGit(t *testing.T, settings map[string]string) *[]string {
	t.Helper()

	var asked []string
	original := gitConfig

	gitConfig = func(dir, key string) (string, error) {
		asked = append(asked, key)

		if v, ok := settings[key]; ok {
			return v, nil
		}

		return "", errors.New("not set")
	}

	t.Cleanup(func() { gitConfig = original })

	return &asked
}

func TestLoadFillsUserFromGit(t *testing.T) {
	git := map[string]string{"user.name": "Git Name", "user.email": "git@example.com"}

	for _, tt := range []struct {
		name    string
		user    string
		project string
		want    map[string]any
		asked   string
	}{
		{
			"neither file sets them", "", "",
			map[string]any{"name": "Git Name", "email": "git@example.com"}, "user.name user.email",
		},
		{
			"the user's file sets the email", "[user]\nemail = \"me@example.com\"\n", "",
			map[string]any{"name": "Git Name", "email": "me@example.com"}, "user.name",
		},
		{
			"the files set both", "[user]\nname = \"Me\"\n", "[user]\nemail = \"me@work.example\"\n",
			map[string]any{"name": "Me", "email": "me@work.example"}, "",
		},
	} {
		asked := stubGit(t, git)
		home, dir := t.TempDir(), t.TempDir()

		if tt.user != "" {
			writeConfig(t, filepath.Join(home, ".ish", FileName), tt.user)
		}
		if tt.project != "" {
			writeConfig(t, filepath.Join(dir, FileName), tt.project)
		}

		cfg, err := Load(home, dir)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cfg["user"], tt.want) {
			t.Errorf("%s: user = %#v, want %#v", tt.name, cfg["user"], tt.want)
		}

		// git is only asked for what the files don't set.
		if got := strings.Join(*asked, " "); got != tt.asked {
			t.Errorf("%s: asked git for %q, want %q", tt.name, got, tt.asked)
		}
	}
}

func TestLoadWithoutGitUser(t *testing.T) {
	stubGit(t, nil)

	cfg, err := Load(t.TempDir(), t.TempDir())

	if err != nil || len(cfg) != 0 {
		t.Errorf("Load() = %v, %v, want no user without git's settings", cfg, err)
	}

	// A user that isn't a table is left alone.
	stubGit(t, map[string]string{"user.email": "git@example.com"})
	dir := t.TempDir()
	writeConfig(t, filepath.Join(dir, FileName), `user = "me"`)

	if cfg, err := Load("", dir); err != nil || cfg["user"] != "me" {
		t.Errorf("Load() = %v, %v, want user left as the file has it", cfg, err)
	}
}
