package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// issueSource returns the source of an issue with the given metadata lines
// and comment contents.
func issueSource(title string, metadata []string, comments ...string) string {
	source := "# " + title + "\n\n[alice@example.com 2026-10-11T08:59:23-07:00]\n"

	if len(metadata) > 0 {
		source += "\n" + strings.Join(metadata, "\n") + "\n"
	}

	for _, comment := range comments {
		source += "\n---\n\n[bob@example.com 2023-05-28 11:55:23 -0400]\n\n" + comment + "\n"
	}

	return source
}

// writeIssues writes each issue source to a file named for its ID in dir.
func writeIssues(t *testing.T, dir string, sources map[string]string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	for id, source := range sources {
		if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func initDir(t *testing.T, dir string) *Result {
	t.Helper()

	result, err := Init(dir)

	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	return result
}

// openDB opens the database in dir, closing it when the test ends.
func openDB(t *testing.T, dir string) *sql.DB {
	t.Helper()

	db, err := Open(filepath.Join(dir, DatabaseName))

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { db.Close() })

	return db
}

// query returns the rows of a query as strings, with columns separated by |.
func query(t *testing.T, db *sql.DB, q string, args ...any) []string {
	t.Helper()

	rows, err := db.Query(q, args...)

	if err != nil {
		t.Fatalf("query %q: %v", q, err)
	}

	defer rows.Close()

	columns, _ := rows.Columns()
	var result []string

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		pointers := make([]any, len(values))

		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			t.Fatal(err)
		}

		fields := make([]string, len(values))
		for i, v := range values {
			fields[i] = v.String
		}

		result = append(result, strings.Join(fields, "|"))
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return result
}

func TestInitCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ish")
	result := initDir(t, dir)

	if !result.CreatedDir || !result.UpdatedGitignore || result.Rebuilt {
		t.Errorf("result = %+v, want the directory and .gitignore created, and nothing rebuilt", result)
	}
	if len(result.Indexed) != 0 || len(result.Failed) != 0 {
		t.Errorf("indexed, failed = %v, %v, want none", result.Indexed, result.Failed)
	}

	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))

	if err != nil || string(gitignore) != ".metadata.ish*\n" {
		t.Errorf(".gitignore = %q (err = %v), want the database entry", gitignore, err)
	}

	db := openDB(t, dir)

	if got := query(t, db, "SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name"); !slices.Equal(got, []string{"attribute", "comment", "issue"}) {
		t.Errorf("tables = %v, want attribute, comment, issue", got)
	}
	if got := query(t, db, "PRAGMA user_version"); !slices.Equal(got, []string{"1"}) {
		t.Errorf("user_version = %v, want 1", got)
	}

	// No temporary databases are left behind.
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	if !slices.Equal(names, []string{".gitignore", DatabaseName}) {
		t.Errorf("directory has %v, want only .gitignore and the database", names)
	}
}

func TestInitIndexesIssues(t *testing.T) {
	dir := t.TempDir()
	source := issueSource("First issue",
		[]string{"- state: open", "- assignees: alice, bob"},
		"First comment", "- [ ] a todo\n  - [x] nested",
	)
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": source,
		"BBBBB-22222": issueSource("Second issue", nil),
	})

	// Files that aren't issues are ignored.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := initDir(t, dir)

	if !slices.Equal(result.Indexed, []string{"AAAAA-11111", "BBBBB-22222"}) || len(result.Failed) != 0 {
		t.Fatalf("indexed, failed = %v, %v, want both issues and no failures", result.Indexed, result.Failed)
	}

	db := openDB(t, dir)
	sum := sha256.Sum256([]byte(source))

	// Dates are stored in UTC, with the byline's offset in minutes.
	want := "AAAAA-11111|AAAAA-11111.md|First issue|alice@example.com|2026-10-11T15:59:23Z|-420|" + hex.EncodeToString(sum[:])

	if got := query(t, db, "SELECT * FROM issue WHERE id = 'AAAAA-11111'"); !slices.Equal(got, []string{want}) {
		t.Errorf("issue = %v, want %v", got, want)
	}

	// Attribute values are JSON arrays.
	if got := query(t, db, "SELECT key, value FROM attribute WHERE issueId = 'AAAAA-11111' ORDER BY key"); !slices.Equal(got, []string{`assignees|["alice","bob"]`, `state|["open"]`}) {
		t.Errorf("attributes = %v", got)
	}

	// Comments keep their order and raw markdown.
	wantComments := []string{
		"0|bob@example.com|2023-05-28T15:55:23Z|-240|First comment",
		"1|bob@example.com|2023-05-28T15:55:23Z|-240|- [ ] a todo\n  - [x] nested",
	}

	if got := query(t, db, "SELECT sortIndex, author, dateCreated, dateCreatedOffset, content FROM comment WHERE issueId = 'AAAAA-11111' ORDER BY sortIndex"); !slices.Equal(got, wantComments) {
		t.Errorf("comments = %q, want %q", got, wantComments)
	}

	// Every row has a distinct version 4 UUID.
	uuids := query(t, db, "SELECT uuid FROM attribute UNION ALL SELECT uuid FROM comment")
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for _, uuid := range uuids {
		if !uuidPattern.MatchString(uuid) {
			t.Errorf("uuid %q isn't a version 4 UUID", uuid)
		}
	}

	if slices.Sort(uuids); len(slices.Compact(uuids)) != 4 {
		t.Errorf("uuids aren't distinct: %v", uuids)
	}

	// Values can be filtered with SQLite's JSON functions.
	if got := query(t, db, `SELECT attribute.issueId FROM attribute, json_each(attribute.value)
		WHERE attribute.key = 'assignees' AND json_each.value = 'bob'`); !slices.Equal(got, []string{"AAAAA-11111"}) {
		t.Errorf("issues assigned to bob = %v", got)
	}
}

func TestInitRebuildsDatabase(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("Before", []string{"- state: open"}),
		"BBBBB-22222": issueSource("Removed", nil),
	})
	initDir(t, dir)

	// Change one issue and remove the other.
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("After", []string{"- state: closed"})})

	if err := os.Remove(filepath.Join(dir, "BBBBB-22222.md")); err != nil {
		t.Fatal(err)
	}

	result := initDir(t, dir)

	if !result.Rebuilt || result.CreatedDir || result.UpdatedGitignore {
		t.Errorf("result = %+v, want only a rebuild", result)
	}

	db := openDB(t, dir)

	if got := query(t, db, "SELECT id, title FROM issue"); !slices.Equal(got, []string{"AAAAA-11111|After"}) {
		t.Errorf("issues = %v, want only the changed issue", got)
	}
	if got := query(t, db, "SELECT value FROM attribute"); !slices.Equal(got, []string{`["closed"]`}) {
		t.Errorf("attributes = %v, want only the changed state", got)
	}

	// The .gitignore entry isn't added twice.
	if gitignore, _ := os.ReadFile(filepath.Join(dir, ".gitignore")); string(gitignore) != ".metadata.ish*\n" {
		t.Errorf(".gitignore = %q, want one entry", gitignore)
	}
}

func TestInitKeepsExistingGitignore(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.tmp"), 0o644); err != nil {
		t.Fatal(err)
	}

	if result := initDir(t, dir); !result.UpdatedGitignore {
		t.Error("UpdatedGitignore = false, want the entry added")
	}

	if gitignore, _ := os.ReadFile(filepath.Join(dir, ".gitignore")); string(gitignore) != "*.tmp\n.metadata.ish*\n" {
		t.Errorf(".gitignore = %q, want the entry added after the existing one", gitignore)
	}

	// An existing entry, even with surrounding space, is left alone.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("  .metadata.ish*  \n*.tmp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if result := initDir(t, dir); result.UpdatedGitignore {
		t.Error("UpdatedGitignore = true, want the existing entry kept")
	}
}

func TestInitSkipsFilesThatCantBeIndexed(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("Good", []string{"- state: open"}, "A comment"),
		"BBBBB-22222": "Not an issue\n",
		"CCCCC-33333": issueSource("Duplicate key", []string{"- state: open", "- state: closed"}),
		"DDDDD-44444": issueSource("Bad key", []string{"- due@date: tomorrow"}),
		"EEEEE-55555": strings.Replace(issueSource("Bad date", nil), "2026-10-11T08:59:23-07:00", "someday", 1),
		"FFFFF-66666": issueSource("Bad comment date", nil, "Comment") + "\n---\n\n[carol yesterday]\n\nLate\n",
	})

	result := initDir(t, dir)

	if !slices.Equal(result.Indexed, []string{"AAAAA-11111"}) {
		t.Errorf("indexed = %v, want only the good issue", result.Indexed)
	}

	wantFailures := map[string]string{
		"BBBBB-22222.md": "heading",
		"CCCCC-33333.md": `"state" appears more than once`,
		"DDDDD-44444.md": `"due@date" may only contain`,
		"EEEEE-55555.md": `byline date "someday"`,
		"FFFFF-66666.md": `comment 2: byline date "yesterday"`,
	}

	if len(result.Failed) != len(wantFailures) {
		t.Fatalf("failed = %v, want %d failures", result.Failed, len(wantFailures))
	}

	for _, failure := range result.Failed {
		want, ok := wantFailures[filepath.Base(failure.Path)]

		if !ok || !strings.Contains(failure.Error(), want) {
			t.Errorf("failure %q, want one for %s containing %q", failure.Error(), filepath.Base(failure.Path), want)
		}
		if !strings.HasPrefix(failure.Path, dir) {
			t.Errorf("failure path %q isn't in %q", failure.Path, dir)
		}
	}

	// Nothing from the skipped files is in the database, even the parts
	// added before a later part failed.
	db := openDB(t, dir)

	for table, want := range map[string]string{"issue": "1", "attribute": "1", "comment": "1"} {
		if got := query(t, db, "SELECT count(*) FROM "+table); !slices.Equal(got, []string{want}) {
			t.Errorf("%s rows = %v, want %s", table, got, want)
		}
	}
}

func TestInitLeavesDatabaseWhenBuildFails(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", nil)})
	initDir(t, dir)

	before, err := os.ReadFile(filepath.Join(dir, DatabaseName))

	if err != nil {
		t.Fatal(err)
	}

	// A directory where the database's temporary file would go makes the
	// build fail before anything is replaced.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	if _, err := Init(dir); err == nil {
		t.Fatal("Init() in a read-only directory: error = nil, want an error")
	}

	after, err := os.ReadFile(filepath.Join(dir, DatabaseName))

	if err != nil || string(after) != string(before) {
		t.Errorf("database changed after a failed build (err = %v)", err)
	}
}

func TestSchemaConstraints(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", []string{"- state: open"}, "One")})
	initDir(t, dir)

	db := openDB(t, dir)

	for _, tt := range []struct {
		name string
		sql  string
	}{
		{"duplicate attribute key", `INSERT INTO attribute VALUES ('u1', 'AAAAA-11111', 'state', '["closed"]')`},
		{"attribute key characters", `INSERT INTO attribute VALUES ('u2', 'AAAAA-11111', 'a b', '["x"]')`},
		{"empty attribute key", `INSERT INTO attribute VALUES ('u3', 'AAAAA-11111', '', '["x"]')`},
		{"attribute value JSON", `INSERT INTO attribute VALUES ('u4', 'AAAAA-11111', 'priority', 'not json')`},
		{"attribute issue", `INSERT INTO attribute VALUES ('u5', 'ZZZZZ-99999', 'priority', '["high"]')`},
		{"duplicate comment index", `INSERT INTO comment VALUES ('u6', 'AAAAA-11111', 0, 'a', 'd', 0, 'c')`},
		{"negative comment index", `INSERT INTO comment VALUES ('u7', 'AAAAA-11111', -1, 'a', 'd', 0, 'c')`},
		{"comment issue", `INSERT INTO comment VALUES ('u8', 'ZZZZZ-99999', 0, 'a', 'd', 0, 'c')`},
		{"duplicate issue path", `INSERT INTO issue VALUES ('BBBBB-22222', 'AAAAA-11111.md', 't', 'a', 'd', 0, 's')`},
	} {
		if _, err := db.Exec(tt.sql); err == nil {
			t.Errorf("%s: insert succeeded, want a constraint error", tt.name)
		}
	}

	// Keys may use letters, digits, and . _ - : /
	if _, err := db.Exec(`INSERT INTO attribute VALUES ('u9', 'AAAAA-11111', 'ns:team/sub_key.v-2', '["x"]')`); err != nil {
		t.Errorf("insert with an allowed key: %v", err)
	}

	// Deleting an issue deletes its attributes and comments.
	if _, err := db.Exec(`DELETE FROM issue WHERE id = 'AAAAA-11111'`); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{"attribute", "comment"} {
		if got := query(t, db, "SELECT count(*) FROM "+table); !slices.Equal(got, []string{"0"}) {
			t.Errorf("%s rows after deleting the issue = %v, want 0", table, got)
		}
	}
}

func TestInitReportsUnreadableDirectory(t *testing.T) {
	// A file where the directory should be.
	path := filepath.Join(t.TempDir(), ".ish")

	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Init(path); err == nil {
		t.Error("Init() with a file in place of the directory: error = nil, want an error")
	}
}

func TestFileError(t *testing.T) {
	err := FileError{Path: ".ish/A.md", Err: errors.New("bad")}

	if err.Error() != ".ish/A.md: bad" || !errors.Is(err, err.Err) {
		t.Errorf("FileError = %q, want the path and wrapped error", err.Error())
	}
}
