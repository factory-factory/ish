package config

import (
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
