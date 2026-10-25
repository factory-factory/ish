// Package store maintains the metadata database, a SQLite cache of the issues
// in the ish base directory that's used to search, filter, and report on them.
// The issue files are always the source of truth, and the database can be
// rebuilt from them at any time.
package store

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"

	// Registers the "sqlite" database/sql driver.
	_ "modernc.org/sqlite"
)

// DatabaseName is the name of the metadata database in the base directory.
const DatabaseName = ".metadata.ish"

// gitignoreEntry keeps the database, its temporary rebuilds, and SQLite's
// journal files out of version control.
const gitignoreEntry = ".metadata.ish*"

// schemaVersion is stored in the database's user_version, so ish can tell
// which schema a database has, and rebuild one with an older schema.
//
// Version 2 stopped allowing colons in attribute keys, version 3 added the
// user table, version 4 gave each name and email one user, with the date
// they were last seen together, version 5 stored authors as their emails and
// emails in attribute values without angle brackets, version 6 added the
// history table, version 7 a table of the commit HEAD was at when the
// history was indexed, and version 8 named it versionHeadMarker.
const schemaVersion = 8

// schema creates the database's tables. Dates are stored in UTC as RFC 3339
// strings, with the byline's offset from UTC in minutes alongside. Authors
// are stored as their emails. Attribute values are JSON arrays of strings,
// with emails stored without angle brackets. Users are the authors of the commits in
// the git repository the issues are in, and of bylines with display names,
// each with the name they were last seen with. History holds the author
// dates of the commits on the checked-out branch that changed each issue, and
// versionHeadMarker, in its one row, the commit HEAD was at when it was
// indexed.
const schema = `
CREATE TABLE issue (
	id                TEXT PRIMARY KEY,
	path              TEXT NOT NULL UNIQUE,
	title             TEXT NOT NULL,
	author            TEXT NOT NULL,
	dateCreated       TEXT NOT NULL,
	dateCreatedOffset INTEGER NOT NULL,
	sha256            TEXT NOT NULL
);

CREATE TABLE attribute (
	uuid    TEXT PRIMARY KEY,
	issueId TEXT NOT NULL REFERENCES issue (id) ON DELETE CASCADE,
	key     TEXT NOT NULL CHECK (key <> '' AND key NOT GLOB '*[^A-Za-z0-9._/-]*'),
	value   TEXT NOT NULL CHECK (json_valid(value)),
	UNIQUE (issueId, key)
);

CREATE INDEX attributeKey ON attribute (key);

CREATE TABLE comment (
	uuid              TEXT PRIMARY KEY,
	issueId           TEXT NOT NULL REFERENCES issue (id) ON DELETE CASCADE,
	sortIndex         INTEGER NOT NULL CHECK (sortIndex >= 0),
	author            TEXT NOT NULL,
	dateCreated       TEXT NOT NULL,
	dateCreatedOffset INTEGER NOT NULL,
	content           TEXT NOT NULL,
	UNIQUE (issueId, sortIndex)
);

CREATE TABLE user (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	email       TEXT NOT NULL,
	dateCreated TEXT NOT NULL
);

CREATE UNIQUE INDEX userName ON user (name);
CREATE UNIQUE INDEX userEmail ON user (email);

CREATE TABLE history (
	uuid         TEXT PRIMARY KEY,
	issueId      TEXT NOT NULL REFERENCES issue (id) ON DELETE CASCADE,
	dateModified TEXT NOT NULL
);

CREATE UNIQUE INDEX historyIssueDate ON history (issueId, dateModified);

CREATE TABLE versionHeadMarker (
	id       INTEGER PRIMARY KEY CHECK (id = 1),
	commitId TEXT NOT NULL
);
`

// keyPattern matches the characters an attribute key may contain.
var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// Result describes what Init did.
type Result struct {
	// CreatedDir reports whether the base directory was created.
	CreatedDir bool

	// UpdatedGitignore reports whether the base directory's .gitignore was
	// created or had the database added to it.
	UpdatedGitignore bool

	// Rebuilt reports whether a database already existed and was replaced.
	Rebuilt bool

	// Indexed holds the IDs of the issues added to the database.
	Indexed []string

	// Failed holds the issue files that couldn't be added to the database.
	Failed []FileError

	// Users is the number of commit authors added to the database, from
	// the repository's whole history.
	Users int

	// UserConflicts holds the commit and byline authors that weren't added
	// because another email has their name.
	UserConflicts []UserConflict

	// Modifications is the number of the issues' modifications added from
	// the git history.
	Modifications int
}

// FileError is an error indexing an issue file.
type FileError struct {
	Path string
	Err  error
}

func (e FileError) Error() string {
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}

func (e FileError) Unwrap() error {
	return e.Err
}

// Init sets up the base directory dir and its metadata database. It creates
// dir if it doesn't exist, makes sure dir's .gitignore ignores the database,
// and builds the database from the issue files in dir, and the authors of
// every commit in the git repository dir is in, if any, replacing any
// database that's already there.
//
// Issue files that can't be indexed are skipped and listed in the result's
// Failed field. An error is returned only if the database can't be built.
func Init(dir string) (*Result, error) {
	result := &Result{}

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		result.CreatedDir = true
	} else if err != nil {
		return nil, err
	}

	updated, err := ensureGitignore(dir)
	if err != nil {
		return nil, err
	}
	result.UpdatedGitignore = updated

	path := filepath.Join(dir, DatabaseName)

	if _, err := os.Stat(path); err == nil {
		result.Rebuilt = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err := build(dir, path, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ensureGitignore adds the database to dir's .gitignore, creating it if
// needed, and reports whether it changed.
func ensureGitignore(dir string) (bool, error) {
	path := filepath.Join(dir, ".gitignore")
	contents, err := os.ReadFile(path)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	for _, line := range strings.Split(string(contents), "\n") {
		if strings.TrimSpace(line) == gitignoreEntry {
			return false, nil
		}
	}

	if len(contents) > 0 && !bytes.HasSuffix(contents, []byte("\n")) {
		contents = append(contents, '\n')
	}

	contents = append(contents, gitignoreEntry+"\n"...)

	return true, os.WriteFile(path, contents, 0o644)
}

// build creates a new database from the issue files in dir, then moves it to
// path, so a database that's being replaced is left untouched if the build
// fails.
func build(dir, path string, result *Result) error {
	tmp, err := os.CreateTemp(dir, DatabaseName+"-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	tmp.Close()

	// Removing the temporary database fails harmlessly once it's been moved.
	defer os.Remove(tmpPath)

	if err := populate(tmpPath, dir, result); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// populate installs the schema in the empty database at path, and adds the
// issue files in dir, the authors of every commit, and the issues' history,
// to it.
func populate(path, dir string, result *Result) (err error) {
	db, err := Open(path)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	if err := installSchema(db); err != nil {
		return err
	}

	if err := index(db, dir, result); err != nil {
		return err
	}

	var conflicts []UserConflict

	result.Users, conflicts, err = addUsers(db, commitAuthors(dir, 0))
	result.UserConflicts = append(result.UserConflicts, conflicts...)
	if err != nil {
		return err
	}

	result.Modifications, err = addAllHistory(db, dir)

	return err
}

// Open opens the database at path with foreign keys enforced.
func Open(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
}

// installSchema creates the tables in an empty database.
func installSchema(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err := tx.Exec(schema); err != nil {
		return err
	}

	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return err
	}

	return tx.Commit()
}

// index adds each issue file in dir to the database in one transaction,
// recording the files it adds and the ones it can't in result.
func index(db *sql.DB, dir string, result *Result) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// ReadDir returns entries sorted by file name, and so by issue ID.
	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || filepath.Ext(name) != ".md" {
			continue
		}

		id := strings.TrimSuffix(name, ".md")

		conflicts, err := indexFile(tx, dir, name)
		result.UserConflicts = append(result.UserConflicts, conflicts...)

		if err != nil {
			result.Failed = append(result.Failed, FileError{Path: filepath.Join(dir, name), Err: err})
			continue
		}

		result.Indexed = append(result.Indexed, id)
	}

	return tx.Commit()
}

// indexFile adds the issue in the file name in dir to the database. If the
// issue can't be added, the database is left as it was.
func indexFile(tx *sql.Tx, dir, name string) (conflicts []UserConflict, err error) {
	source, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}

	issue, err := parse.ParseIssue(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}

	issue.Id = strings.TrimSuffix(name, ".md")

	if err := validate(issue); err != nil {
		return nil, err
	}

	// A savepoint undoes this issue's rows if any of them can't be added.
	if _, err := tx.Exec("SAVEPOINT issue"); err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Exec("ROLLBACK TO issue")
		}
		tx.Exec("RELEASE issue")
	}()

	created, offset, err := utcDate(issue.DateCreated)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(source)

	if _, err := tx.Exec(
		`INSERT INTO issue (id, path, title, author, dateCreated, dateCreatedOffset, sha256)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		issue.Id, name, issue.Title, models.AuthorEmail(issue.Author), created, offset, hex.EncodeToString(sum[:]),
	); err != nil {
		return nil, err
	}

	for _, tag := range issue.Metadata {
		value, err := json.Marshal(storedValues(tag.Values))
		if err != nil {
			return nil, err
		}

		if _, err := tx.Exec(
			`INSERT INTO attribute (uuid, issueId, key, value) VALUES (?, ?, ?, ?)`,
			newUUID(), issue.Id, tag.Key, string(value),
		); err != nil {
			return nil, err
		}
	}

	for i, card := range issue.Conversation {
		created, offset, err := utcDate(card.DateCreated)
		if err != nil {
			return nil, fmt.Errorf("comment %d: %w", i+1, err)
		}

		content := ""
		if card.Content != nil {
			content = *card.Content
		}

		if _, err := tx.Exec(
			`INSERT INTO comment (uuid, issueId, sortIndex, author, dateCreated, dateCreatedOffset, content)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			newUUID(), issue.Id, i, models.AuthorEmail(card.Author), created, offset, content,
		); err != nil {
			return nil, err
		}
	}

	// Bylines with display names add their users.
	_, conflicts, err = addUsers(tx, bylineUsers(issue))

	return conflicts, err
}

// validate checks the rules the database has for an issue's attributes: each
// key is made of allowed characters, and appears only once.
func validate(issue *models.Issue) error {
	seen := map[string]bool{}

	for _, tag := range issue.Metadata {
		if !keyPattern.MatchString(tag.Key) {
			return fmt.Errorf("metadata key %q may only contain letters, digits, and . _ - /", tag.Key)
		}

		if seen[tag.Key] {
			return fmt.Errorf("metadata key %q appears more than once", tag.Key)
		}

		seen[tag.Key] = true
	}

	return nil
}

// utcDate converts a byline's date to an RFC 3339 string in UTC, and returns
// the date's offset from UTC in minutes.
func utcDate(s string) (string, int, error) {
	t, err := models.ParseDate(s)
	if err != nil {
		return "", 0, fmt.Errorf("byline date %q: %w", s, err)
	}

	_, offset := t.Zone()

	return t.UTC().Format(time.RFC3339), offset / 60, nil
}

// storedValues returns a tag's values as they're stored in the database, with
// emails in angle brackets stored without the brackets. The issue files keep
// them as they're written.
func storedValues(tokens []models.Token) []string {
	values := models.Canonical(tokens)

	for i, v := range values {
		values[i] = models.StoredValue(v)
	}

	return values
}

// newUUID returns a random version 4 UUID.
func newUUID() string {
	var b [16]byte

	// crypto/rand.Read never returns an error.
	rand.Read(b[:])

	b[6] = b[6]&0x0f | 0x40 // Version 4.
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant.

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
