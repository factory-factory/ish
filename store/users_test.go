package store

import (
	"errors"
	"os/exec"
	"regexp"
	"slices"
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

func TestParseAuthors(t *testing.T) {
	out := "Alice Example\x00alice@example.com\n" +
		"Bob <the builder>\x00bob@example.com\n" +
		"Alice Example\x00alice@example.com\n" + // A duplicate.
		"Alice Example\x00alice@work.example\n" + // Another email.
		"\n" +
		"no separator\n" +
		"\x00\n" +
		"Carol\x00\n"

	want := []User{
		{"Alice Example", "alice@example.com"},
		{"Bob <the builder>", "bob@example.com"},
		{"Alice Example", "alice@work.example"},
		{"Carol", ""},
	}

	if got := parseAuthors([]byte(out)); !slices.Equal(got, want) {
		t.Errorf("parseAuthors() = %q, want %q", got, want)
	}
}

func TestInitIndexesEveryAuthor(t *testing.T) {
	limits := stubAuthors(t, "Alice\x00alice@example.com\nBob\x00bob@example.com\nAlice\x00alice@example.com\n", nil)
	dir := t.TempDir()
	result := initDir(t, dir)

	if result.Users != 2 {
		t.Errorf("Users = %d, want 2", result.Users)
	}
	if !slices.Equal(*limits, []int{0}) {
		t.Errorf("git log limits = %v, want every commit (0)", *limits)
	}

	users, err := Users(dir)

	if err != nil {
		t.Fatal(err)
	}
	if want := []User{{"Alice", "alice@example.com"}, {"Bob", "bob@example.com"}}; !slices.Equal(users, want) {
		t.Errorf("Users() = %q, want %q", users, want)
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

func TestIndexUsersAddsRecentAuthors(t *testing.T) {
	stubAuthors(t, "Alice\x00alice@example.com\n", nil)
	dir := t.TempDir()
	initDir(t, dir)

	db := openDB(t, dir)
	before := query(t, db, "SELECT id FROM user")

	limits := stubAuthors(t, "Carol\x00carol@example.com\nAlice\x00alice@example.com\n", nil)
	added, err := IndexUsers(dir)

	if err != nil || added != 1 {
		t.Fatalf("IndexUsers() = %d, %v, want 1 new user", added, err)
	}
	if !slices.Equal(*limits, []int{recentCommits}) {
		t.Errorf("git log limits = %v, want the most recent %d commits", *limits, recentCommits)
	}

	// Existing users keep their IDs, and indexing again adds nothing.
	if got := query(t, db, "SELECT id FROM user WHERE name = 'Alice'"); !slices.Equal(got, before) {
		t.Errorf("Alice's id = %v, want %v unchanged", got, before)
	}
	if added, err := IndexUsers(dir); err != nil || added != 0 {
		t.Errorf("IndexUsers() again = %d, %v, want 0", added, err)
	}

	users, _ := Users(dir)
	if want := []User{{"Alice", "alice@example.com"}, {"Carol", "carol@example.com"}}; !slices.Equal(users, want) {
		t.Errorf("Users() = %q, want %q", users, want)
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
	if added, err := IndexUsers(dir); err != nil || added != 0 {
		t.Errorf("IndexUsers() = %d, %v, want 0, nil", added, err)
	}
	if _, err := IndexUsers(t.TempDir()); err == nil {
		t.Error("IndexUsers() without a database error = nil, want an error")
	}
}

func TestUserTableIndexes(t *testing.T) {
	stubAuthors(t, "", nil)
	dir := t.TempDir()
	initDir(t, dir)
	db := openDB(t, dir)

	if got := query(t, db, "SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'user' AND sql IS NOT NULL ORDER BY name"); !slices.Equal(got, []string{"userEmail", "userName", "userNameEmail"}) {
		t.Errorf("user indexes = %v, want userEmail, userName, userNameEmail", got)
	}

	// A name and email pair is unique.
	if _, err := db.Exec("INSERT INTO user VALUES ('a', 'Alice', 'alice@example.com')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO user VALUES ('b', 'Alice', 'alice@example.com')"); err == nil {
		t.Error("inserted a duplicate name and email")
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

	for _, author := range []User{{"Alice Example", "alice@example.com"}, {"Bob", "bob@example.com"}} {
		git([]string{
			"GIT_AUTHOR_NAME=" + author.Name, "GIT_AUTHOR_EMAIL=" + author.Email,
			"GIT_COMMITTER_NAME=ish", "GIT_COMMITTER_EMAIL=ish@example.com",
		}, "commit", "-q", "--allow-empty", "-m", "commit by "+author.Name)
	}

	out, err := gitAuthors(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := parseAuthors(out), []User{{"Bob", "bob@example.com"}, {"Alice Example", "alice@example.com"}}; !slices.Equal(got, want) {
		t.Errorf("every commit's authors = %q, want %q", got, want)
	}

	out, err = gitAuthors(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := parseAuthors(out), []User{{"Bob", "bob@example.com"}}; !slices.Equal(got, want) {
		t.Errorf("the latest commit's authors = %q, want %q", got, want)
	}

	if _, err := gitAuthors(t.TempDir(), 0); err == nil {
		t.Error("gitAuthors() outside a repository error = nil, want an error")
	}
}
