package store

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
)

func listIssues(t *testing.T, dir string) []Summary {
	t.Helper()

	summaries, err := ListIssues(dir, nil)

	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}

	return summaries
}

func TestListIssues(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"BBBBB-22222": issueSource("Second", []string{"- assignees: alice, bob", "- state: in progress", "- priority: high"}),
		"AAAAA-11111": issueSource("First", []string{"- state: open"}),
		"CCCCC-33333": issueSource("Third", nil),
	})
	initDir(t, dir)

	summaries := listIssues(t, dir)

	var ids []string
	for _, s := range summaries {
		ids = append(ids, s.ID)
	}

	if !slices.Equal(ids, []string{"AAAAA-11111", "BBBBB-22222", "CCCCC-33333"}) {
		t.Fatalf("IDs = %v, want every issue in ID order", ids)
	}

	second := summaries[1]

	// Several values keep their order, and other attributes aren't included.
	if second.Title != "Second" || !slices.Equal(second.Assignees, []string{"alice", "bob"}) || !slices.Equal(second.State, []string{"in progress"}) {
		t.Errorf("second = %+v, want its title, assignees, and state", second)
	}

	// An issue without the attributes has none.
	if third := summaries[2]; third.Assignees != nil || third.State != nil {
		t.Errorf("third assignees, state = %v, %v, want none", third.Assignees, third.State)
	}

	// The author is their email.
	if got := summaries[0].Author; got != "alice@example.com" {
		t.Errorf("author = %q, want alice@example.com", got)
	}

	// The opening date is in the byline's time zone.
	if got := summaries[0].Opened.Format(time.RFC3339); got != "2026-10-11T08:59:23-07:00" {
		t.Errorf("opened = %s, want 2026-10-11T08:59:23-07:00", got)
	}
}

func TestListIssuesWithFilter(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", []string{"- state: open"}),
		"BBBBB-22222": issueSource("Second", []string{"- state: closed"}),
		"CCCCC-33333": issueSource("Second draft", []string{"- state: open"}),
	})
	initDir(t, dir)

	// The filter's fields and attributes work alongside the list's joins.
	where, err := filter.Parse("{@title ~ /^Second/} and {state != 'closed'}")
	if err != nil {
		t.Fatal(err)
	}

	summaries, err := ListIssues(dir, where)

	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != "CCCCC-33333" {
		t.Errorf("ListIssues() = %+v, want only CCCCC-33333", summaries)
	}
}

func TestAttributes(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", []string{"- assignees: alice, bob", "- state: open", "- priority: high"}),
		"BBBBB-22222": issueSource("Second", []string{"- state: closed"}),
		"CCCCC-33333": issueSource("Third", nil),
	})
	initDir(t, dir)

	values, err := Attributes(dir, []string{"assignees", "state", "missing"})
	if err != nil {
		t.Fatalf("Attributes() error = %v", err)
	}

	// Only the requested keys are included, and several values keep their
	// order. Issues without any of them have no entry.
	want := map[string]map[string][]string{
		"AAAAA-11111": {"assignees": {"alice", "bob"}, "state": {"open"}},
		"BBBBB-22222": {"state": {"closed"}},
	}

	if !reflect.DeepEqual(values, want) {
		t.Errorf("Attributes() = %v, want %v", values, want)
	}

	// No keys select nothing.
	if values, err := Attributes(dir, nil); err != nil || len(values) != 0 {
		t.Errorf("Attributes(nil) = %v, %v, want none", values, err)
	}
}

func TestListIssuesWithoutDatabase(t *testing.T) {
	dir := t.TempDir()

	if _, err := ListIssues(dir, nil); err == nil || !strings.Contains(err.Error(), "ish init") {
		t.Errorf("ListIssues() without a database: error = %v, want one suggesting ish init", err)
	}

	// Looking for the database doesn't create an empty one.
	if exists, _ := Exists(dir); exists {
		t.Error("ListIssues() created a database")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()

	if exists, err := Exists(dir); exists || err != nil {
		t.Errorf("Exists() before init = %v, %v, want false, nil", exists, err)
	}

	initDir(t, dir)

	if exists, err := Exists(dir); !exists || err != nil {
		t.Errorf("Exists() after init = %v, %v, want true, nil", exists, err)
	}
}

func TestSaveIssueUpdatesDatabase(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", []string{"- state: open"}, "One"),
		"BBBBB-22222": issueSource("Second", []string{"- state: open"}),
	})
	initDir(t, dir)

	issue, err := parse.LoadIssue(dir, "AAAAA-11111")

	if err != nil {
		t.Fatal(err)
	}

	// Edit the title and state, and add a comment.
	issue.Title = "First, edited"
	issue.Metadata[0].Values = models.Tokens("closed")
	content := "Two"
	issue.Conversation = append(issue.Conversation, models.Card{
		Author: "carol@example.com", DateCreated: "2026-10-12T09:00:00-07:00", Content: &content,
	})

	if err := SaveIssue(dir, *issue); err != nil {
		t.Fatalf("SaveIssue() error = %v", err)
	}

	// The file is written.
	if saved, err := parse.LoadIssue(dir, "AAAAA-11111"); err != nil || saved.Title != "First, edited" {
		t.Errorf("saved file title = %v (err = %v), want the edited title", saved, err)
	}

	// The list reflects the edit.
	if first := listIssues(t, dir)[0]; first.Title != "First, edited" || !slices.Equal(first.State, []string{"closed"}) {
		t.Errorf("listed first = %+v, want the edited title and state", first)
	}

	db := openDB(t, dir)

	// The issue's rows were replaced, not added to.
	if got := query(t, db, "SELECT sortIndex, content FROM comment WHERE issueId = 'AAAAA-11111' ORDER BY sortIndex"); !slices.Equal(got, []string{"0|One", "1|Two"}) {
		t.Errorf("comments = %v, want both comments once", got)
	}
	if got := query(t, db, "SELECT count(*) FROM attribute WHERE issueId = 'AAAAA-11111'"); !slices.Equal(got, []string{"1"}) {
		t.Errorf("attributes = %v, want 1", got)
	}

	// The file's new checksum is recorded.
	source, _ := os.ReadFile(filepath.Join(dir, "AAAAA-11111.md"))
	if got := query(t, db, "SELECT sha256 FROM issue WHERE id = 'AAAAA-11111'"); !slices.Equal(got, []string{sha256Hex(source)}) {
		t.Errorf("sha256 = %v, want the saved file's", got)
	}

	// Other issues are untouched.
	if got := query(t, db, "SELECT title FROM issue WHERE id = 'BBBBB-22222'"); !slices.Equal(got, []string{"Second"}) {
		t.Errorf("other issue = %v, want it unchanged", got)
	}
}

func TestSaveNewIssue(t *testing.T) {
	dir := t.TempDir()
	initDir(t, dir)

	issue := models.NewIssue("AAAAA-11111", "alice@example.com", time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC))
	issue.Title = "New"

	if err := SaveIssue(dir, issue); err != nil {
		t.Fatalf("SaveIssue() error = %v", err)
	}

	if summaries := listIssues(t, dir); len(summaries) != 1 || summaries[0].Title != "New" {
		t.Errorf("listed %+v, want the new issue", summaries)
	}
}

func TestSaveIssueRejectsInvalidMetadata(t *testing.T) {
	for _, tt := range []struct {
		name     string
		metadata []models.Tag
		want     string
	}{
		{"duplicate key", []models.Tag{{Key: "state", Values: models.Tokens("open")}, {Key: "state", Values: models.Tokens("closed")}}, "more than once"},
		{"key characters", []models.Tag{{Key: "due@date", Values: models.Tokens("tomorrow")}}, "may only contain"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			source := issueSource("First", []string{"- state: open"})
			writeIssues(t, dir, map[string]string{"AAAAA-11111": source})
			initDir(t, dir)

			issue, err := parse.LoadIssue(dir, "AAAAA-11111")

			if err != nil {
				t.Fatal(err)
			}

			issue.Metadata = tt.metadata

			if err := SaveIssue(dir, *issue); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("SaveIssue() error = %v, want one containing %q", err, tt.want)
			}

			// Nothing is written.
			if got, _ := os.ReadFile(filepath.Join(dir, "AAAAA-11111.md")); string(got) != source {
				t.Errorf("file = %q, want it unchanged", got)
			}
			if first := listIssues(t, dir)[0]; !slices.Equal(first.State, []string{"open"}) {
				t.Errorf("listed state = %v, want it unchanged", first.State)
			}
		})
	}
}

func TestSaveIssueWithoutDatabase(t *testing.T) {
	dir := t.TempDir()
	issue := models.NewIssue("AAAAA-11111", "alice@example.com", time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC))
	issue.Title = "New"

	// The file is written, but the error says the database wasn't updated.
	err := SaveIssue(dir, issue)

	if err == nil || !strings.Contains(err.Error(), "couldn't update the metadata database") {
		t.Errorf("SaveIssue() without a database: error = %v, want one about the database", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "AAAAA-11111.md")); statErr != nil {
		t.Errorf("issue file not written: %v", statErr)
	}
}

func TestIndexIssueLeavesDatabaseWhenFileIsBad(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issueSource("First", []string{"- state: open"}, "One")})
	initDir(t, dir)

	writeIssues(t, dir, map[string]string{"AAAAA-11111": "Not an issue\n"})

	if err := IndexIssue(dir, "AAAAA-11111"); err == nil {
		t.Fatal("IndexIssue() of a bad file: error = nil, want an error")
	}

	// The transaction is rolled back, so the old rows are still there.
	db := openDB(t, dir)

	for table, want := range map[string]string{"issue": "1", "attribute": "1", "comment": "1"} {
		if got := query(t, db, "SELECT count(*) FROM "+table); !slices.Equal(got, []string{want}) {
			t.Errorf("%s rows = %v, want %s", table, got, want)
		}
	}
}

func TestCurrent(t *testing.T) {
	dir := t.TempDir()

	if _, err := Current(dir); err == nil {
		t.Error("Current() without a database: error = nil, want an error")
	}

	initDir(t, dir)

	if current, err := Current(dir); !current || err != nil {
		t.Errorf("Current() after init = %v, %v, want true, nil", current, err)
	}

	// A database from an older version of ish.
	if _, err := openDB(t, dir).Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}

	if current, err := Current(dir); current || err != nil {
		t.Errorf("Current() with version 1 = %v, %v, want false, nil", current, err)
	}
}

func TestIssueIDs(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", nil),
		"BBBBB-22222": issueSource("Second", nil),
	})
	initDir(t, dir)

	ids, err := IssueIDs(dir)

	if err != nil {
		t.Fatalf("IssueIDs() error = %v", err)
	}
	if len(ids) != 2 || !ids["AAAAA-11111"] || !ids["BBBBB-22222"] {
		t.Errorf("IssueIDs() = %v, want both issues", ids)
	}

	if _, err := IssueIDs(t.TempDir()); err == nil {
		t.Error("IssueIDs() without a database error = nil, want an error")
	}
}

func TestMetadataValues(t *testing.T) {
	dir := t.TempDir()
	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", []string{"- assignees: zaphine, alice", "- state: open"}),
		"BBBBB-22222": issueSource("Second", []string{"- assignees: Bob, alice", "- state: closed"}),
	})
	initDir(t, dir)

	values, err := MetadataValues(dir)

	if err != nil {
		t.Fatalf("MetadataValues() error = %v", err)
	}

	// Each value appears once, in order regardless of case.
	want := map[string][]string{
		"assignees": {"alice", "Bob", "zaphine"},
		"state":     {"closed", "open"},
	}

	if len(values) != len(want) {
		t.Errorf("MetadataValues() = %v, want %v", values, want)
	}
	for key, w := range want {
		if !slices.Equal(values[key], w) {
			t.Errorf("values for %s = %q, want %q", key, values[key], w)
		}
	}

	if _, err := MetadataValues(t.TempDir()); err == nil {
		t.Error("MetadataValues() without a database error = nil, want an error")
	}
}

func TestCountIssues(t *testing.T) {
	dir := t.TempDir()

	if _, err := CountIssues(dir); err == nil {
		t.Error("CountIssues() without a database: error = nil, want an error")
	}

	initDir(t, dir)

	if count, err := CountIssues(dir); count != 0 || err != nil {
		t.Errorf("CountIssues() of an empty database = %d, %v, want 0, nil", count, err)
	}

	writeIssues(t, dir, map[string]string{
		"AAAAA-11111": issueSource("First", nil),
		"BBBBB-22222": issueSource("Second", nil),
	})
	initDir(t, dir)

	if count, err := CountIssues(dir); count != 2 || err != nil {
		t.Errorf("CountIssues() = %d, %v, want 2, nil", count, err)
	}
}
