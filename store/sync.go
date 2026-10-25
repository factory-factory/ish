package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// SyncResult describes the changes Sync found and applied.
type SyncResult struct {
	// Updated holds the IDs of the issues whose files changed and were
	// re-indexed.
	Updated []string

	// Added holds the IDs of the issues whose files are new.
	Added []string

	// Removed holds the IDs of the issues whose files were deleted.
	Removed []string

	// Failed holds the new or changed issue files that couldn't be indexed.
	// Their stale rows, if any, are removed.
	Failed []FileError

	// UserConflicts holds the byline authors that weren't added because
	// another email has their name.
	UserConflicts []UserConflict
}

// Changed reports whether Sync found any differences between the issue files
// and the database.
func (r *SyncResult) Changed() bool {
	return len(r.Updated)+len(r.Added)+len(r.Removed)+len(r.Failed) > 0
}

// Sync brings the metadata database in dir up to date with the issue files,
// in one transaction. An issue whose file's SHA-256 checksum differs from the
// one in the database is re-indexed, an issue whose file is gone is removed,
// and a file with no issue in the database is indexed. Files that haven't
// changed aren't parsed. A new file's history, up to the commit the history
// was last indexed at, is added with it; see IndexHistory.
//
// A new or changed file that can't be indexed is listed in the result's
// Failed field, and nothing from it is in the database. An error is returned
// only if the database can't be updated.
func Sync(dir string) (result *SyncResult, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	// The checksums stored for each issue file, by file name.
	stored := map[string]string{}
	ids := map[string]string{}

	rows, err := db.Query("SELECT id, path, sha256 FROM issue ORDER BY id")
	if err != nil {
		return nil, err
	}

	var paths []string

	for rows.Next() {
		var id, path, sum string

		if err := rows.Scan(&id, &path, &sum); err != nil {
			rows.Close()
			return nil, err
		}

		stored[path] = sum
		ids[path] = id
		paths = append(paths, path)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	result = &SyncResult{}

	// Issues in the database whose files changed or are gone.
	for _, path := range paths {
		source, err := os.ReadFile(filepath.Join(dir, path))

		if errors.Is(err, os.ErrNotExist) {
			if _, err := tx.Exec("DELETE FROM issue WHERE id = ?", ids[path]); err != nil {
				return nil, err
			}

			result.Removed = append(result.Removed, ids[path])
			continue
		}

		if err != nil {
			return nil, err
		}

		if sum := sha256.Sum256(source); hex.EncodeToString(sum[:]) == stored[path] {
			continue
		}

		// The stale rows are removed even if the file can't be re-indexed,
		// but its history, which comes from git, is kept if it can.
		history, err := savedHistory(tx, ids[path])
		if err != nil {
			return nil, err
		}

		if _, err := tx.Exec("DELETE FROM issue WHERE id = ?", ids[path]); err != nil {
			return nil, err
		}

		conflicts, err := indexFile(tx, dir, path)
		result.UserConflicts = append(result.UserConflicts, conflicts...)

		if err != nil {
			result.Failed = append(result.Failed, FileError{Path: filepath.Join(dir, path), Err: err})
			continue
		}

		if err := restoreHistory(tx, ids[path], history); err != nil {
			return nil, err
		}

		result.Updated = append(result.Updated, ids[path])
	}

	// Issue files with no issue in the database.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var added []string

	for _, entry := range entries {
		name := entry.Name()

		if _, known := stored[name]; known || entry.IsDir() || filepath.Ext(name) != ".md" {
			continue
		}

		conflicts, err := indexFile(tx, dir, name)
		result.UserConflicts = append(result.UserConflicts, conflicts...)

		if err != nil {
			result.Failed = append(result.Failed, FileError{Path: filepath.Join(dir, name), Err: err})
			continue
		}

		result.Added = append(result.Added, strings.TrimSuffix(name, ".md"))
		added = append(added, name)
	}

	// A new file may have been committed before it was added, as when a
	// pull brings it in, or it's fixed after failing to be indexed.
	if err := addFileHistory(tx, dir, added); err != nil {
		return nil, err
	}

	return result, tx.Commit()
}
