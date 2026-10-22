// Package config loads ish's configuration: a free-form, nested object read
// from TOML files, whose values can be looked up by a dotted path such as
// .user.email.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// FileName is the name of the configuration files, in the user's ~/.ish
// directory and in a project's issue directory.
const FileName = "config.toml"

// Config is a configuration: TOML tables are maps, arrays are slices, and
// other values are strings, int64s, float64s, bools, or times, as the TOML
// package decodes them. A Config isn't changed once it's loaded.
type Config map[string]any

// Defaults returns the configuration that the files are applied to. There
// are no settings yet, so it's empty.
func Defaults() Config {
	return Config{}
}

// gitConfig returns git's setting for key, such as user.email, in the
// repository dir is in. Tests replace it.
var gitConfig = func(dir, key string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "config", key).Output()
	return strings.TrimSpace(string(out)), err
}

// userKeys are the settings for the user running ish, which are read from git
// if neither configuration file sets them.
var userKeys = []string{"name", "email"}

// Load returns the configuration: the defaults, then the per-user file
// ~/.ish/config.toml in home, then the project's file in dir, each applied on
// top of the ones before. A file that doesn't exist is skipped. An empty home
// skips the per-user file. Then user.name and user.email, if neither file
// sets them, are read from git's settings for the repository dir is in. This
// is the only time git is asked for them.
func Load(home, dir string) (Config, error) {
	cfg := Defaults()

	var paths []string
	if home != "" {
		paths = append(paths, filepath.Join(home, ".ish", FileName))
	}
	paths = append(paths, filepath.Join(dir, FileName))

	for _, path := range paths {
		file, err := readFile(path)
		if err != nil {
			return nil, err
		}

		cfg = merge(cfg, file)
	}

	return withGitUser(cfg, dir), nil
}

// withGitUser returns cfg with the user's name and email from git, for
// whichever of them cfg doesn't set. A user that isn't a table is left as it
// is.
func withGitUser(cfg Config, dir string) Config {
	user, ok := cfg["user"].(map[string]any)

	if _, set := cfg["user"]; set && !ok {
		return cfg
	}

	var found map[string]any

	for _, key := range userKeys {
		if _, set := user[key]; set {
			continue
		}

		if value, err := gitConfig(dir, "user."+key); err == nil && value != "" {
			if found == nil {
				found = map[string]any{}
			}

			found[key] = value
		}
	}

	if found == nil {
		return cfg
	}

	// git's values are the lowest layer, under the files' settings.
	return merge(Config{"user": found}, cfg)
}

// readFile reads the configuration file at path, or returns an empty
// configuration if it doesn't exist.
func readFile(path string) (Config, error) {
	var raw map[string]any

	if _, err := toml.DecodeFile(path, &raw); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, nil
		}

		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return normalize(raw).(map[string]any), nil
}

// normalize returns v with every table as a map[string]any and every array
// as a []any, so lookups only meet those two kinds of container. The TOML
// package decodes an array of tables as a []map[string]any.
func normalize(v any) any {
	switch v := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(v))
		for k, e := range v {
			m[k] = normalize(e)
		}
		return m

	case Config:
		return normalize(map[string]any(v))

	case []map[string]any:
		s := make([]any, len(v))
		for i, e := range v {
			s[i] = normalize(e)
		}
		return s

	case []any:
		s := make([]any, len(v))
		for i, e := range v {
			s[i] = normalize(e)
		}
		return s
	}

	return v
}

// merge returns base with over applied on top: tables in both are merged key
// by key, and any other value in over replaces base's. Neither is changed.
func merge(base, over map[string]any) Config {
	out := make(Config, len(base)+len(over))

	for k, v := range base {
		out[k] = normalize(v)
	}

	for k, v := range over {
		baseTable, baseOK := out[k].(map[string]any)
		overTable, overOK := v.(map[string]any)

		if baseOK && overOK {
			out[k] = map[string]any(merge(baseTable, overTable))
			continue
		}

		out[k] = normalize(v)
	}

	return out
}

// Lookup returns the value at path, a dotted path of keys from the top of
// the configuration, such as .user.email or user.email. A key that's a
// number indexes an array, as in .remotes.0.url. The value is a copy, so
// changing it doesn't change the configuration.
func (c Config) Lookup(path string) (any, bool) {
	path = strings.TrimPrefix(path, ".")

	var v any = map[string]any(c)

	if path == "" {
		return normalize(v), true
	}

	for _, key := range strings.Split(path, ".") {
		switch container := v.(type) {
		case map[string]any:
			next, ok := container[key]
			if !ok {
				return nil, false
			}
			v = next

		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(container) {
				return nil, false
			}
			v = container[i]

		default:
			return nil, false
		}
	}

	return normalize(v), true
}

// UserHome returns the directory the per-user configuration is read from:
// the user's home directory, or "" if it can't be found.
func UserHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return home
}
