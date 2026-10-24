package store

import (
	"bufio"
	"bytes"
	"database/sql"
	"os/exec"
	"path/filepath"
	"strings"
)

// commitMarker starts each commit in gitHistory's output, followed by the
// commit's author date.
const commitMarker = "\x00"

// gitHistory returns git's list of the commits on the checked-out branch
// that changed files in dir, each as commitMarker and its author date on a
// line, followed by the files it changed, one per line. With since set, only
// commits made since then are listed: git compares since with when commits
// were made, not their author dates. Tests replace it.
var gitHistory = func(dir, since string) ([]byte, error) {
	// git writes the marker for %x00, since an argument can't hold one.
	args := []string{"-C", dir, "log", "--format=%x00%aI", "--name-only"}

	if since != "" {
		args = append(args, "--since="+since)
	}

	return exec.Command("git", append(args, "--", ".")...).Output()
}

// modification is an issue that a commit changed, and when.
type modification struct {
	issueID      string
	dateModified string
}

// parseHistory returns the issues changed in gitHistory's output, by their
// file names, with the commits' author dates in UTC.
func parseHistory(out []byte) []modification {
	var mods []modification
	var date string

	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(nil, 1<<20)

	for scanner.Scan() {
		line := scanner.Text()

		if after, ok := strings.CutPrefix(line, commitMarker); ok {
			date = ""

			if utc, _, err := utcDate(after); err == nil {
				date = utc
			}

			continue
		}

		name := filepath.Base(line)

		if date == "" || line == "" || !strings.HasSuffix(name, ".md") {
			continue
		}

		mods = append(mods, modification{strings.TrimSuffix(name, ".md"), date})
	}

	return mods
}

// addHistory adds modifications to the database, skipping those it already
// has, and those of issues it doesn't have, such as deleted or renamed ones.
// It returns how many it added.
func addHistory(db querier, mods []modification) (int, error) {
	added := 0

	for _, m := range mods {
		result, err := db.Exec(
			`INSERT OR IGNORE INTO history (uuid, issueId, dateModified)
			SELECT ?, ?, ? WHERE EXISTS (SELECT 1 FROM issue WHERE id = ?)`,
			newUUID(), m.issueID, m.dateModified, m.issueID,
		)
		if err != nil {
			return added, err
		}

		if n, err := result.RowsAffected(); err == nil {
			added += int(n)
		}
	}

	return added, nil
}

// commitHistory returns the modifications in the git history of dir since
// since, or all of them if it's empty. It returns none if dir isn't in a git
// repository, or git isn't installed.
func commitHistory(dir, since string) []modification {
	out, err := gitHistory(dir, since)
	if err != nil {
		return nil
	}

	return parseHistory(out)
}

// IndexHistory adds the commits made since the newest modification in the
// metadata database in dir, as ish does each time it starts, and returns how
// many modifications it added. With no modifications yet, it adds the whole
// history. If dir isn't in a git repository, it adds none.
func IndexHistory(dir string) (added int, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return 0, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	var since string
	if err := db.QueryRow(`SELECT coalesce(max(dateModified), '') FROM history`).Scan(&since); err != nil {
		return 0, err
	}

	return addHistory(db, commitHistory(dir, since))
}

// savedHistory returns the dates of the issue's modifications, so they can
// be restored after it's re-indexed, which deletes them with the issue's
// other rows. They come from git, not the issue's file.
func savedHistory(tx *sql.Tx, id string) ([]string, error) {
	rows, err := tx.Query(`SELECT dateModified FROM history WHERE issueId = ?`, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var dates []string

	for rows.Next() {
		var date string

		if err := rows.Scan(&date); err != nil {
			return nil, err
		}

		dates = append(dates, date)
	}

	return dates, rows.Err()
}

// restoreHistory adds the issue's saved modifications back.
func restoreHistory(tx *sql.Tx, id string, dates []string) error {
	mods := make([]modification, len(dates))
	for i, d := range dates {
		mods[i] = modification{id, d}
	}

	_, err := addHistory(tx, mods)

	return err
}
