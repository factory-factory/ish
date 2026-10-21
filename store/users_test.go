package store

import (
	"errors"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// stubAuthors replaces gitAuthors with one that returns out, and records
// the limits it's called with.
func stubAuthors(t *testing.T, out string, err error) *[]int {
	t.Helper()

	var limits []int
	original := gitAuthors
	gitAuthors = func(dir string, limit int) ([]byte, error) {
		limits = append(limits, limit)
		return []byte(out), err
	}
	t.Cleanup(func() { gitAuthors = original })

	return &limits
}

// authors returns git log output listing each name, email, and date.
func authors(entries ...[3]string) string {
	var b strings.Builder

	for _, e := range entries {
		b.WriteString(e[0] + "\x00" + e[1] + "\x00" + e[2] + "\n")
	}

	return b.String()
}

func TestParseAuthors(t *testing.T) {
	out := authors(
		[3]string{"Alice Example", "alice@example.com", "2026-10-02T09:00:00-07:00"},
		[3]string{"Bob <the builder>", "bob@example.com", "2026-10-01T09:00:00Z"},
		// An older commit by the same author keeps the newer date.
		[3]string{"Alice Example", "alice@example.com", "2026-09-01T09:00:00-07:00"},
		[3]string{"Alice Example", "alice@work.example", "2026-09-01T09:00:00-07:00"},
	) + "\n" + "no separators\n" + "Carol\x00carol@example.com\x00not a date\n"

	want := []User{
		{"Alice Example", "alice@example.com", "2026-10-02T16:00:00Z"},
		{"Bob <the builder>", "bob@example.com", "2026-10-01T09:00:00Z"},
		{"Alice Example", "alice@work.example", "2026-09-01T16:00:00Z"},
	}

	if got := parseAuthors([]byte(out)); !slices.Equal(got, want) {
		t.Errorf("parseAuthors() = %q, want %q", got, want)
	}
}

// userRows returns the users in the database in dir as "name email date".
func userRows(t *testing.T, dir string) []string {
	t.Helper()

	users, err := Users(dir)
	if err != nil {
		t.Fatal(err)
	}

	var rows []string
	for _, u := range users {
		rows = append(rows, u.Name+" "+u.Email+" "+u.DateCreated)
	}

	return rows
}

func TestInitIndexesEveryAuthor(t *testing.T) {
	limits := stubAuthors(t, authors(
		[3]string{"Alice", "alice@example.com", "2026-10-02T09:00:00Z"},
		[3]string{"Bob", "bob@example.com", "2026-10-01T09:00:00Z"},
	), nil)
	dir := t.TempDir()
	result := initDir(t, dir)

	if result.Users != 2 || len(result.UserConflicts) != 0 {
		t.Errorf("Users, UserConflicts = %d, %v, want 2 and none", result.Users, result.UserConflicts)
	}
	if !slices.Equal(*limits, []int{0}) {
		t.Errorf("git log limits = %v, want every commit (0)", *limits)
	}

	if got, want := userRows(t, dir), []string{
		"Alice alice@example.com 2026-10-02T09:00:00Z",
		"Bob bob@example.com 2026-10-01T09:00:00Z",
	}; !slices.Equal(got, want) {
		t.Errorf("users = %q, want %q", got, want)
	}

	// Each user has a version 4 UUID.
	db := openDB(t, dir)
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for _, id := range query(t, db, "SELECT id FROM user") {
		if !uuidPattern.MatchString(id) {
			t.Errorf("user id %q isn't a version 4 UUID", id)
		}
	}
}

func TestAddUser(t *testing.T) {
	stubAuthors(t, "", nil)
	dir := t.TempDir()
	initDir(t, dir)
	db := openDB(t, dir)

	for _, tt := range []struct {
		name     string
		user     User
		changed  bool
		conflict string
		want     []string
	}{
		{
			"a new user", User{"Alice", "alice@example.com", "2026-10-02T00:00:00Z"}, true, "",
			[]string{"Alice alice@example.com 2026-10-02T00:00:00Z"},
		},
		{
			// A newer name for the email replaces the older one.
			"a newer name", User{"Alice Liddell", "alice@example.com", "2026-10-03T00:00:00Z"}, true, "",
			[]string{"Alice Liddell alice@example.com 2026-10-03T00:00:00Z"},
		},
		{
			// An older name is skipped.
			"an older name", User{"Ally", "alice@example.com", "2026-10-01T00:00:00Z"}, false, "",
			[]string{"Alice Liddell alice@example.com 2026-10-03T00:00:00Z"},
		},
		{
			// The same name, seen more recently, updates the date.
			"the same name, newer", User{"Alice Liddell", "alice@example.com", "2026-10-04T00:00:00Z"}, true, "",
			[]string{"Alice Liddell alice@example.com 2026-10-04T00:00:00Z"},
		},
		{
			// A name another email has is skipped, however new.
			"a taken name", User{"Alice Liddell", "alice@work.example", "2026-10-05T00:00:00Z"}, false, "alice@example.com",
			[]string{"Alice Liddell alice@example.com 2026-10-04T00:00:00Z"},
		},
		{
			"another user", User{"Bob", "bob@example.com", "2026-10-01T00:00:00Z"}, true, "",
			[]string{"Alice Liddell alice@example.com 2026-10-04T00:00:00Z", "Bob bob@example.com 2026-10-01T00:00:00Z"},
		},
		{
			// A newer name another email has is skipped too.
			"a newer, taken name", User{"Bob", "alice@example.com", "2026-10-06T00:00:00Z"}, false, "bob@example.com",
			[]string{"Alice Liddell alice@example.com 2026-10-04T00:00:00Z", "Bob bob@example.com 2026-10-01T00:00:00Z"},
		},
		{
			"no name", User{"", "carol@example.com", "2026-10-01T00:00:00Z"}, false, "",
			[]string{"Alice Liddell alice@example.com 2026-10-04T00:00:00Z", "Bob bob@example.com 2026-10-01T00:00:00Z"},
		},
	} {
		changed, conflict, err := addUser(db, tt.user)

		if err != nil {
			t.Fatalf("%s: addUser() error = %v", tt.name, err)
		}

		gotConflict := ""
		if conflict != nil {
			gotConflict = conflict.Email
		}

		if changed != tt.changed || gotConflict != tt.conflict {
			t.Errorf("%s: addUser() = %v, conflict with %q, want %v, %q", tt.name, changed, gotConflict, tt.changed, tt.conflict)
		}
		if got := userRows(t, dir); !slices.Equal(got, tt.want) {
			t.Errorf("%s: users = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestUserConflictError(t *testing.T) {
	c := UserConflict{User: User{Name: "Alice", Email: "alice@work.example"}, Email: "alice@example.com"}

	if got, want := c.Error(), `"Alice" <alice@work.example>: the name is already used by alice@example.com`; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestIndexUsersAddsRecentAuthors(t *testing.T) {
	stubAuthors(t, authors([3]string{"Alice", "alice@example.com", "2026-10-01T00:00:00Z"}), nil)
	dir := t.TempDir()
	initDir(t, dir)

	limits := stubAuthors(t, authors(
		[3]string{"Carol", "carol@example.com", "2026-10-02T00:00:00Z"},
		[3]string{"Alice", "alice@example.com", "2026-10-01T00:00:00Z"},
		[3]string{"Alice", "alice@work.example", "2026-10-02T00:00:00Z"},
	), nil)
	changed, conflicts, err := IndexUsers(dir)

	if err != nil || changed != 1 {
		t.Fatalf("IndexUsers() = %d, %v, want 1 new user", changed, err)
	}
	if len(conflicts) != 1 || conflicts[0].User.Email != "alice@work.example" {
		t.Errorf("conflicts = %v, want Alice's other email", conflicts)
	}
	if !slices.Equal(*limits, []int{recentCommits}) {
		t.Errorf("git log limits = %v, want the most recent %d commits", *limits, recentCommits)
	}

	if changed, _, err := IndexUsers(dir); err != nil || changed != 0 {
		t.Errorf("IndexUsers() again = %d, %v, want 0", changed, err)
	}
}

func TestBylinesAddUsers(t *testing.T) {
	stubAuthors(t, authors([3]string{"Alice", "alice@example.com", "2026-10-01T00:00:00Z"}), nil)
	dir := t.TempDir()

	issue := "# Bylines\n\n[\"Robert\" <bob@example.com> 2026-10-02T09:00:00-07:00]\n" +
		"\n---\n\n[Alice Liddell <alice@example.com> 2026-10-03T00:00:00Z]\n\nNewer than git.\n" +
		"\n---\n\n[<carol@example.com> 2026-10-03T00:00:00Z]\n\nNo name.\n" +
		"\n---\n\n[\"Robert\" <rob@example.com> 2026-10-04T00:00:00Z]\n\nA taken name.\n"
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issue})

	result := initDir(t, dir)

	if got, want := userRows(t, dir), []string{
		"Alice Liddell alice@example.com 2026-10-03T00:00:00Z",
		"Robert bob@example.com 2026-10-02T16:00:00Z",
	}; !slices.Equal(got, want) {
		t.Errorf("users = %q, want %q", got, want)
	}
	if len(result.UserConflicts) != 1 || result.UserConflicts[0].User.Email != "rob@example.com" {
		t.Errorf("UserConflicts = %v, want Robert's second email", result.UserConflicts)
	}

	// Changing a byline adds its user when the issue is synced.
	writeIssues(t, dir, map[string]string{"AAAAA-11111": issue + "\n---\n\n[Dave <dave@example.com> 2026-10-05T00:00:00Z]\n\nHi\n"})

	synced, err := Sync(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got := userRows(t, dir); !slices.Contains(got, "Dave dave@example.com 2026-10-05T00:00:00Z") {
		t.Errorf("users after sync = %q, want Dave", got)
	}
	if len(synced.UserConflicts) != 1 {
		t.Errorf("sync conflicts = %v, want Robert's second email again", synced.UserConflicts)
	}
}

func TestUsersWithoutGit(t *testing.T) {
	// Outside a git repository, or without git, there are no users, and
	// nothing fails.
	stubAuthors(t, "", errors.New("not a git repository"))
	dir := t.TempDir()

	if result := initDir(t, dir); result.Users != 0 {
		t.Errorf("Users = %d, want 0", result.Users)
	}
	if changed, _, err := IndexUsers(dir); err != nil || changed != 0 {
		t.Errorf("IndexUsers() = %d, %v, want 0, nil", changed, err)
	}
	if _, _, err := IndexUsers(t.TempDir()); err == nil {
		t.Error("IndexUsers() without a database error = nil, want an error")
	}
}

func TestUserTableIndexes(t *testing.T) {
	stubAuthors(t, "", nil)
	dir := t.TempDir()
	initDir(t, dir)
	db := openDB(t, dir)

	if got := query(t, db, "SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'user' AND sql IS NOT NULL ORDER BY name"); !slices.Equal(got, []string{"userEmail", "userName"}) {
		t.Errorf("user indexes = %v, want userEmail and userName", got)
	}

	// Names and emails are each unique.
	if _, err := db.Exec("INSERT INTO user VALUES ('a', 'Alice', 'alice@example.com', '2026-10-01T00:00:00Z')"); err != nil {
		t.Fatal(err)
	}

	for _, row := range []string{
		"('b', 'Alice', 'alice@work.example', '2026-10-01T00:00:00Z')",
		"('c', 'Ally', 'alice@example.com', '2026-10-01T00:00:00Z')",
	} {
		if _, err := db.Exec("INSERT INTO user VALUES " + row); err == nil {
			t.Errorf("inserted %s, which shares a name or email", row)
		}
	}
}

func TestGitAuthors(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git isn't installed")
	}

	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	git := func(env []string, args ...string) {
		t.Helper()

		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(cmd.Environ(), env...)

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	git(nil, "init", "-q")

	for _, author := range []User{
		{"Alice Example", "alice@example.com", "2026-10-01T09:00:00-07:00"},
		{"Bob", "bob@example.com", "2026-10-02T09:00:00-07:00"},
	} {
		git([]string{
			"GIT_AUTHOR_NAME=" + author.Name, "GIT_AUTHOR_EMAIL=" + author.Email, "GIT_AUTHOR_DATE=" + author.DateCreated,
			"GIT_COMMITTER_NAME=ish", "GIT_COMMITTER_EMAIL=ish@example.com",
		}, "commit", "-q", "--allow-empty", "-m", "commit by "+author.Name)
	}

	out, err := gitAuthors(dir, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Each author is dated by their commit's author date, in UTC.
	if got, want := parseAuthors(out), []User{
		{"Bob", "bob@example.com", "2026-10-02T16:00:00Z"},
		{"Alice Example", "alice@example.com", "2026-10-01T16:00:00Z"},
	}; !slices.Equal(got, want) {
		t.Errorf("every commit's authors = %q, want %q", got, want)
	}

	out, err = gitAuthors(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := parseAuthors(out); len(got) != 1 || got[0].Name != "Bob" {
		t.Errorf("the latest commit's authors = %q, want Bob's", got)
	}

	if _, err := gitAuthors(t.TempDir(), 0); err == nil {
		t.Error("gitAuthors() outside a repository error = nil, want an error")
	}
}
