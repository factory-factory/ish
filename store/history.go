package store

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// commitMarker starts each commit in gitHistory's output, followed by the
// commit's author date.
const commitMarker = "\x00"

// gitHistory returns git's list of the commits in revs, such as a commit
// and its ancestors, or a range like old..new, that changed files in dir,
// or only the given files in dir, if there are any. Each commit is listed as
// commitMarker and its author date on a line, followed by the files it
// changed, one per line. Tests replace it.
var gitHistory = func(dir, revs string, paths []string) ([]byte, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// git writes the marker for %x00, since an argument can't hold one.
	args := []string{"-C", dir, "log", "--format=%x00%aI", "--name-only", revs, "--"}

	return exec.Command("git", append(args, paths...)...).Output()
}

// gitHead returns the commit HEAD is at in the repository dir is in. Tests
// replace it.
var gitHead = func(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", "HEAD^{commit}").Output()

	return strings.TrimSpace(string(out)), err
}

// gitIsAncestor reports whether commit is head or one of its ancestors in
// the repository dir is in. It's false if either commit doesn't exist. Tests
// replace it.
var gitIsAncestor = func(dir, commit, head string) bool {
	return exec.Command("git", "-C", dir, "merge-base", "--is-ancestor", commit, head).Run() == nil
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

// commitHistory returns the modifications in the git history of dir made by
// the commits in revs, to the given files, or to any if there are none. It
// returns none if dir isn't in a git repository, or git isn't installed.
func commitHistory(dir, revs string, paths []string) []modification {
	out, err := gitHistory(dir, revs, paths)
	if err != nil {
		return nil
	}

	return parseHistory(out)
}

// indexedHead returns the commit HEAD was at when the history in db was last
// indexed, or "" if it hasn't been.
func indexedHead(db querier) (string, error) {
	var head string

	err := db.QueryRow(`SELECT commitId FROM versionHeadMarker`).Scan(&head)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	return head, err
}

// setIndexedHead records that the history in db has been indexed up to head.
func setIndexedHead(db querier, head string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO versionHeadMarker (id, commitId) VALUES (1, ?)`, head)

	return err
}

// addAllHistory adds the history of the commit HEAD is at in dir's
// repository, and its ancestors, to db, and records it as indexed. If dir
// isn't in a repository with commits, it adds none.
func addAllHistory(db querier, dir string) (int, error) {
	// HEAD is read first, so a commit made meanwhile is indexed next time.
	head, err := gitHead(dir)
	if err != nil {
		return 0, nil
	}

	added, err := addHistory(db, commitHistory(dir, head, nil))
	if err != nil {
		return added, err
	}

	return added, setIndexedHead(db, head)
}

// IndexHistory brings the history in the metadata database in dir up to
// date with the git history, as ish does when it starts and whenever HEAD
// moves, and returns how many modifications it added. If HEAD has moved on
// from the commit the history was last indexed at, the commits since then
// are added, whatever their dates, as a pull or merge brings in commits
// older than ones already indexed. If it hasn't, as after a checkout of
// another branch, a reset, or a rebase, the history is rebuilt, so it's the
// history of the checked-out branch. If dir isn't in a git repository with
// commits, it adds none.
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

	head, err := gitHead(dir)
	if err != nil {
		return 0, nil
	}

	indexed, err := indexedHead(db)
	if err != nil || indexed == head {
		return 0, err
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	defer tx.Rollback()

	revs := indexed + ".." + head

	if indexed == "" || !gitIsAncestor(dir, indexed, head) {
		if _, err := tx.Exec(`DELETE FROM history`); err != nil {
			return 0, err
		}

		revs = head
	}

	out, err := gitHistory(dir, revs, nil)
	if err != nil {
		return 0, err
	}

	if added, err = addHistory(tx, parseHistory(out)); err != nil {
		return 0, err
	}

	if err := setIndexedHead(tx, head); err != nil {
		return 0, err
	}

	return added, tx.Commit()
}

// addFileHistory adds the history of the issue files at paths in dir, up to
// the commit the history was last indexed at, to db. Issues added to the
// database after the commits that changed them were indexed, such as files
// that couldn't be read until they were fixed, would be missing it.
func addFileHistory(db querier, dir string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	head, err := indexedHead(db)
	if err != nil || head == "" {
		return err
	}

	_, err = addHistory(db, commitHistory(dir, head, paths))

	return err
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
