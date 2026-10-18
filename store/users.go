package store

import (
	"bytes"
	"database/sql"
	"os/exec"
	"strconv"
)

// User is someone who has committed to the repository the issues are in, by
// the name and email of their git author attributions.
type User struct {
	Name  string
	Email string
}

// recentCommits is the number of recent commits whose authors are indexed
// each time ish starts. Init indexes the authors of every commit.
const recentCommits = 50

// gitAuthors returns git's list of the authors of the commits on every
// branch of the repository dir is in, newest first, each as a name and email
// separated by a NUL character, one per line. With limit above 0, only the
// most recent limit commits are listed. Tests replace it.
var gitAuthors = func(dir string, limit int) ([]byte, error) {
	args := []string{"-C", dir, "log", "--all", "--format=%an%x00%ae"}

	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}

	return exec.Command("git", args...).Output()
}

// parseAuthors returns the distinct users in gitAuthors' output, in the order
// they're first listed.
func parseAuthors(out []byte) []User {
	var users []User
	seen := map[User]bool{}

	for _, line := range bytes.Split(out, []byte("\n")) {
		name, email, ok := bytes.Cut(line, []byte{0})
		user := User{Name: string(name), Email: string(email)}

		if !ok || user == (User{}) || seen[user] {
			continue
		}

		seen[user] = true
		users = append(users, user)
	}

	return users
}

// commitAuthors returns the authors of the most recent limit commits in the
// repository dir is in, or of every commit if limit is 0. It returns none if
// dir isn't in a git repository, or git isn't installed.
func commitAuthors(dir string, limit int) []User {
	out, err := gitAuthors(dir, limit)
	if err != nil {
		return nil
	}

	return parseAuthors(out)
}

// execer runs SQL statements, in a database or a transaction.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// addUsers adds the users the database doesn't have yet, and returns how
// many it added.
func addUsers(db execer, users []User) (int, error) {
	added := 0

	for _, user := range users {
		result, err := db.Exec(
			`INSERT INTO user (id, name, email) VALUES (?, ?, ?) ON CONFLICT (name, email) DO NOTHING`,
			newUUID(), user.Name, user.Email,
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

// IndexUsers adds the authors of the repository's most recent commits to the
// metadata database in dir, as ish does each time it starts, and returns how
// many it added. If dir isn't in a git repository, it adds none.
func IndexUsers(dir string) (added int, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return 0, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	return addUsers(db, commitAuthors(dir, recentCommits))
}

// Users returns the users in the metadata database in dir, ordered by name
// and email.
func Users(dir string) (users []User, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := db.Close(); err == nil {
			err = closeErr
		}
	}()

	rows, err := db.Query(`SELECT name, email FROM user ORDER BY name, email`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user User

		if err := rows.Scan(&user.Name, &user.Email); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, rows.Err()
}
