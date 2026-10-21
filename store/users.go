package store

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/factory-factory/ish/models"
)

// User is someone with a display name and an email: a commit's author in the
// git history, or the author of a byline with a display name. DateCreated is
// when they were seen with that name, in UTC as an RFC 3339 string.
type User struct {
	Name        string
	Email       string
	DateCreated string
}

// UserConflict is a user who wasn't added, because another email already
// has their name. Names and emails are each unique.
type UserConflict struct {
	User User

	// Email is the email that has the name.
	Email string
}

func (c UserConflict) Error() string {
	return fmt.Sprintf("%s: the name is already used by %s", models.Mailbox{Name: c.User.Name, Email: c.User.Email}, c.Email)
}

// recentCommits is the number of recent commits whose authors are indexed
// each time ish starts. Init indexes the authors of every commit.
const recentCommits = 50

// gitAuthors returns git's list of the authors of the commits on every
// branch of the repository dir is in, newest first, each as a name, email,
// and author date separated by NUL characters, one per line. With limit
// above 0, only the most recent limit commits are listed. Tests replace it.
var gitAuthors = func(dir string, limit int) ([]byte, error) {
	args := []string{"-C", dir, "log", "--all", "--format=%an%x00%ae%x00%aI"}

	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}

	return exec.Command("git", args...).Output()
}

// parseAuthors returns the distinct users in gitAuthors' output, in the order
// they're first listed, each with the date they last authored a commit.
func parseAuthors(out []byte) []User {
	var users []User
	index := map[[2]string]int{}

	for _, line := range bytes.Split(out, []byte("\n")) {
		fields := strings.Split(string(line), "\x00")
		if len(fields) != 3 || fields[0] == "" && fields[1] == "" {
			continue
		}

		date, _, err := utcDate(fields[2])
		if err != nil {
			continue
		}

		user := User{Name: fields[0], Email: fields[1], DateCreated: date}
		key := [2]string{user.Name, user.Email}

		if i, ok := index[key]; ok {
			users[i].DateCreated = max(users[i].DateCreated, date)
			continue
		}

		index[key] = len(users)
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

// querier runs SQL statements and queries, in a database or a transaction.
type querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// addUser adds user to the database, or updates the name of the user with
// their email if user is newer. A user without a name or email is ignored.
// It reports whether it changed the database, or the conflict if another
// email already has the name, in which case the user is skipped.
func addUser(db querier, user User) (bool, *UserConflict, error) {
	if user.Name == "" || user.Email == "" {
		return false, nil, nil
	}

	var id, name, date string

	err := db.QueryRow(`SELECT id, name, dateCreated FROM user WHERE email = ?`, user.Email).Scan(&id, &name, &date)
	found := err == nil

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, nil, err
	}

	// An older name for the email is kept.
	if found && user.DateCreated <= date {
		return false, nil, nil
	}

	if !found || user.Name != name {
		var other string

		err := db.QueryRow(`SELECT email FROM user WHERE name = ? AND email <> ?`, user.Name, user.Email).Scan(&other)
		if err == nil {
			return false, &UserConflict{User: user, Email: other}, nil
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return false, nil, err
		}
	}

	if found {
		_, err = db.Exec(`UPDATE user SET name = ?, dateCreated = ? WHERE id = ?`, user.Name, user.DateCreated, id)
	} else {
		_, err = db.Exec(`INSERT INTO user (id, name, email, dateCreated) VALUES (?, ?, ?, ?)`,
			newUUID(), user.Name, user.Email, user.DateCreated)
	}

	return err == nil, nil, err
}

// addUsers adds users to the database as addUser does, and returns how many
// it added or updated, and the conflicts.
func addUsers(db querier, users []User) (int, []UserConflict, error) {
	changed := 0

	var conflicts []UserConflict

	for _, user := range users {
		ok, conflict, err := addUser(db, user)
		if err != nil {
			return changed, conflicts, err
		}

		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}

		if ok {
			changed++
		}
	}

	return changed, conflicts, nil
}

// bylineUsers returns the users named in the bylines of issue: those with a
// display name, dated by their bylines.
func bylineUsers(issue *models.Issue) []User {
	var users []User

	add := func(author, date string) {
		box, err := models.ParseMailbox(author)
		if err != nil || box.Name == "" {
			return
		}

		if utc, _, err := utcDate(date); err == nil {
			users = append(users, User{Name: box.Name, Email: box.Email, DateCreated: utc})
		}
	}

	add(issue.Author, issue.DateCreated)

	for _, card := range issue.Conversation {
		add(card.Author, card.DateCreated)
	}

	return users
}

// IndexUsers adds the authors of the repository's most recent commits to the
// metadata database in dir, as ish does each time it starts, and returns how
// many users it added or updated, and those it skipped because another email
// has their name. If dir isn't in a git repository, it adds none.
func IndexUsers(dir string) (changed int, conflicts []UserConflict, err error) {
	db, err := openExisting(dir)
	if err != nil {
		return 0, nil, err
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

	rows, err := db.Query(`SELECT name, email, dateCreated FROM user ORDER BY name, email`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user User

		if err := rows.Scan(&user.Name, &user.Email, &user.DateCreated); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, rows.Err()
}
