package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
)

// Summary is the part of an issue that the issue list shows.
type Summary struct {
	ID        string
	Title     string
	Assignees []string
	State     []string

	// Opened is when the issue was opened, in its byline's time zone.
	Opened time.Time
}

// listQuery selects issues with the attributes the issue list shows. The
// unique (issueId, key) constraint means each join matches at most one
// attribute. A filter's condition is added after it, then the order.
const listQuery = `
SELECT issue.id, issue.title, issue.dateCreated, issue.dateCreatedOffset,
       assignees.value, state.value
FROM issue
LEFT JOIN attribute AS assignees
  ON assignees.issueId = issue.id AND assignees.key = 'assignees'
LEFT JOIN attribute AS state
  ON state.issueId = issue.id AND state.key = 'state'
`

// Exists reports whether the base directory dir has a metadata database.
func Exists(dir string) (bool, error) {
	_, err := os.Stat(filepath.Join(dir, DatabaseName))

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return err == nil, err
}

// Current reports whether the metadata database in dir has the schema this
// version of ish uses. A database with an older schema should be rebuilt with
// Init.
func Current(dir string) (current bool, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return false, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	var version int

	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return false, err
	}

	return version == schemaVersion, nil
}

// openExisting opens the metadata database in dir, returning an error if it
// doesn't exist rather than creating an empty one.
func openExisting(dir string) (*sql.DB, error) {
	exists, err := Exists(dir)

	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("%s doesn't exist; run ish init to create it", filepath.Join(dir, DatabaseName))
	}

	return Open(filepath.Join(dir, DatabaseName))
}

// CountIssues returns the number of issues in the metadata database in dir.
func CountIssues(dir string) (count int, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return 0, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	err = db.QueryRow("SELECT count(*) FROM issue").Scan(&count)

	return count, err
}

// ListIssues returns a summary of each issue in the metadata database in dir
// that matches where, ordered by ID. If where is nil, every issue is listed.
func ListIssues(dir string, where filter.Node) (summaries []Summary, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	query := listQuery
	var args []any

	if where != nil {
		var clause string
		clause, args = where.Where()
		query += "WHERE " + clause + "\n"
	}

	rows, err := db.Query(query+"ORDER BY issue.id", args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var s Summary
		var date string
		var offset int
		var assignees, state sql.NullString

		if err := rows.Scan(&s.ID, &s.Title, &date, &offset, &assignees, &state); err != nil {
			return nil, err
		}

		utc, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, fmt.Errorf("issue %s: %w", s.ID, err)
		}

		s.Opened = utc.In(time.FixedZone("", offset*60))

		if s.Assignees, err = decodeValues(assignees); err != nil {
			return nil, fmt.Errorf("issue %s assignees: %w", s.ID, err)
		}

		if s.State, err = decodeValues(state); err != nil {
			return nil, fmt.Errorf("issue %s state: %w", s.ID, err)
		}

		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

// decodeValues decodes an attribute's JSON array of values, or returns no
// values if the issue doesn't have the attribute.
func decodeValues(value sql.NullString) ([]string, error) {
	if !value.Valid {
		return nil, nil
	}

	var values []string

	return values, json.Unmarshal([]byte(value.String), &values)
}

// SaveIssue writes issue to its file in the base directory dir, and replaces
// its rows in dir's metadata database. The issue is checked against the
// database's rules first, so an issue the database can't hold isn't written.
func SaveIssue(dir string, issue models.Issue) error {
	if err := validate(&issue); err != nil {
		return err
	}

	if err := parse.SaveIssue(dir, issue); err != nil {
		return err
	}

	if err := IndexIssue(dir, issue.Id); err != nil {
		return fmt.Errorf("saved %s, but couldn't update the metadata database: %w", issue.Id, err)
	}

	return nil
}

// IndexIssue replaces the rows of the issue with the given ID in the metadata
// database in dir with the contents of its file, in one transaction.
func IndexIssue(dir, id string) (err error) {
	db, err := openExisting(dir)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// The issue's attributes and comments are deleted with it.
	if _, err := tx.Exec("DELETE FROM issue WHERE id = ?", id); err != nil {
		return err
	}

	if err := indexFile(tx, dir, id+".md"); err != nil {
		return err
	}

	return tx.Commit()
}
