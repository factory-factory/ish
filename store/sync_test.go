package store

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/factory-factory/ish/parse"
)

func syncDir(t *testing.T, dir string) *SyncResult {
	t.Helper()

	result, err := Sync(dir)

	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	return result
}

// uuids returns the UUIDs of an issue's attributes and comments, which
// change whenever the issue is re-indexed.
func uuids(t *testing.T, dir, id string) []string {
	t.Helper()

	return query(t, openDB(t, dir), `
		SELECT uuid FROM attribute WHERE issueId = ?
		UNION ALL
		SELECT uuid FROM comment WHERE issueId = ?
		ORDER BY 1`, id, id)
}

// syncedDir returns a directory with three indexed issues.
func syncedDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", []string{"- state: open"}, "One"),
		"BBBBB-22222": issueSource("Second", []string{"- state: open"}, "Two"),
		"CCCCC-33333": issueSource("Third", []string{"- state: open"}, "Three"),
	})
	initDir(t, dir)

	return dir
}

func TestSyncWithoutChanges(t *testing.T) {
	dir := syncedDir(t)
	before := uuids(t, dir, "AAAAA-11111")

	// Files that aren't issues are ignored.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := syncDir(t, dir)

	if result.Changed() {
		t.Errorf("result = %+v, want no changes", result)
	}

	// Unchanged issues aren't re-indexed.
	if after := uuids(t, dir, "AAAAA-11111"); !slices.Equal(after, before) {
		t.Errorf("uuids changed from %v to %v, want the rows left alone", before, after)
	}
}

func TestSyncChangedFile(t *testing.T) {
	dir := syncedDir(t)
	before := map[string][]string{
		"AAAAA-11111": uuids(t, dir, "AAAAA-11111"),
		"BBBBB-22222": uuids(t, dir, "BBBBB-22222"),
	}

	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First, edited", []string{"- state: closed", "- priority: high"}, "One", "Another"),
	})

	result := syncDir(t, dir)

	if !slices.Equal(result.Updated, []string{"AAAAA-11111"}) || len(result.Added)+len(result.Removed)+len(result.Failed) != 0 {
		t.Fatalf("result = %+v, want only AAAAA-11111 updated", result)
	}

	db := openDB(t, dir)

	if got := query(t, db, "SELECT title FROM issue WHERE id = 'AAAAA-11111'"); !slices.Equal(got, []string{"First, edited"}) {
		t.Errorf("title = %v, want the edited title", got)
	}
	if got := query(t, db, "SELECT key, value FROM attribute WHERE issueId = 'AAAAA-11111' ORDER BY key"); !slices.Equal(got, []string{`priority|["high"]`, `state|["closed"]`}) {
		t.Errorf("attributes = %v, want the edited attributes only", got)
	}
	if got := query(t, db, "SELECT sortIndex, content FROM comment WHERE issueId = 'AAAAA-11111' ORDER BY sortIndex"); !slices.Equal(got, []string{"0|One", "1|Another"}) {
		t.Errorf("comments = %v, want the edited comments only", got)
	}

	// The stored checksum now matches the file, so the next sync finds
	// nothing.
	if again := syncDir(t, dir); again.Changed() {
		t.Errorf("second sync = %+v, want no changes", again)
	}

	// The changed issue was re-indexed; the other was left alone.
	if after := uuids(t, dir, "AAAAA-11111"); slices.Equal(after, before["AAAAA-11111"]) {
		t.Error("changed issue's rows weren't replaced")
	}
	if after := uuids(t, dir, "BBBBB-22222"); !slices.Equal(after, before["BBBBB-22222"]) {
		t.Error("unchanged issue's rows were replaced")
	}
}

func TestSyncAddedAndRemovedFiles(t *testing.T) {
	dir := syncedDir(t)

	writeIssues(t, dir, map[string]string{"DDDDD-44444": issueSource("Fourth", []string{"- state: open"}, "Four")})

	if err := os.Remove(filepath.Join(dir, "BBBBB-22222.md")); err != nil {
		t.Fatal(err)
	}

	result := syncDir(t, dir)

	if !slices.Equal(result.Added, []string{"DDDDD-44444"}) || !slices.Equal(result.Removed, []string{"BBBBB-22222"}) ||
		len(result.Updated)+len(result.Failed) != 0 {
		t.Fatalf("result = %+v, want DDDDD-44444 added and BBBBB-22222 removed", result)
	}

	db := openDB(t, dir)

	if got := query(t, db, "SELECT id FROM issue ORDER BY id"); !slices.Equal(got, []string{"AAAAA-11111", "CCCCC-33333", "DDDDD-44444"}) {
		t.Errorf("issues = %v", got)
	}

	// The removed issue's attributes and comments are gone too.
	for _, table := range []string{"attribute", "comment"} {
		if got := query(t, db, "SELECT count(*) FROM "+table+" WHERE issueId = 'BBBBB-22222'"); !slices.Equal(got, []string{"0"}) {
			t.Errorf("%s rows for the removed issue = %v, want 0", table, got)
		}
	}
}

func TestSyncFilesThatCantBeIndexed(t *testing.T) {
	dir := syncedDir(t)

	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": "Not an issue any more\n",
		"EEEEE-55555": issueSource("Duplicate key", []string{"- state: open", "- state: closed"}),
	})

	result := syncDir(t, dir)

	var failed []string
	for _, failure := range result.Failed {
		failed = append(failed, filepath.Base(failure.Path))
	}

	if !slices.Equal(failed, []string{"AAAAA-11111.md", "EEEEE-55555.md"}) || len(result.Updated)+len(result.Added) != 0 {
		t.Fatalf("result = %+v, want both files failed", result)
	}

	// The changed file's stale rows are removed, and the new file isn't
	// added.
	db := openDB(t, dir)

	if got := query(t, db, "SELECT id FROM issue ORDER BY id"); !slices.Equal(got, []string{"BBBBB-22222", "CCCCC-33333"}) {
		t.Errorf("issues = %v, want only the issues that can be indexed", got)
	}
	if got := query(t, db, "SELECT count(*) FROM attribute WHERE issueId = 'AAAAA-11111'"); !slices.Equal(got, []string{"0"}) {
		t.Errorf("stale attributes = %v, want 0", got)
	}

	// Files that still can't be indexed are reported on every sync.
	if again := syncDir(t, dir); len(again.Failed) != 2 {
		t.Errorf("second sync failed = %v, want both files again", again.Failed)
	}
}

func TestSyncAfterSaveIssue(t *testing.T) {
	dir := syncedDir(t)

	issue, err := parse.LoadIssue(dir, "AAAAA-11111")

	if err != nil {
		t.Fatal(err)
	}

	issue.Title = "Saved from ish"

	if err := SaveIssue(dir, *issue); err != nil {
		t.Fatal(err)
	}

	// Saving updates the checksum, so the saved file isn't seen as changed.
	if result := syncDir(t, dir); result.Changed() {
		t.Errorf("result = %+v, want no changes after saving through ish", result)
	}
}

func TestSyncWithoutDatabase(t *testing.T) {
	if _, err := Sync(t.TempDir()); err == nil || !strings.Contains(err.Error(), "ish init") {
		t.Errorf("Sync() without a database: error = %v, want one suggesting ish init", err)
	}
}
