package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/factory-factory/ish/app"
	"github.com/factory-factory/ish/config"
	"github.com/factory-factory/ish/parse"
	"github.com/factory-factory/ish/state"
	"github.com/factory-factory/ish/store"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:              "ish [issue file]",
		Short:            "A file-based distributed issue tracker",
		Args:             cobra.MaximumNArgs(1),
		SilenceErrors:    false,
		SilenceUsage:     false,
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			st, err := loadState(config.UserHome(), parse.IssuesDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't read the configuration: %v\n", err)
				os.Exit(1)
			}

			if err := checkUser(st); err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't start ish: %v\n", err)
				os.Exit(1)
			}

			if !ensureDatabase(parse.IssuesDir, cmd.OutOrStdout(), cmd.ErrOrStderr()) {
				os.Exit(1)
			}

			var model app.Model

			if len(args) == 1 {
				model, err = app.Open(parse.IssuesDir, issueIdFromPath(args[0]))
			} else {
				model, err = app.New(parse.IssuesDir)
			}

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			model.SetState(st)

			p := tea.NewProgram(model)
			if _, err := p.Run(); err != nil {
				fmt.Printf("Oh no, an error:\n%v", err)
				os.Exit(1)
			}
		},
	}
)

// issueIdFromPath returns the ID of the issue stored at path, which is the
// file name without its extension.
func issueIdFromPath(path string) string {
	name := filepath.Base(path)
	return strings.TrimSuffix(name, filepath.Ext(name))
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create the issue directory and its metadata database",
	Long: `Create the .ish issue directory if it doesn't exist, and build its
metadata database from the issue files in it, replacing any database that's
already there. Issue files that can't be added to the database are reported,
and ish exits with a non-zero status.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(runInit(parse.IssuesDir, cmd.OutOrStdout(), cmd.ErrOrStderr()))
	},
}

// runInit sets up the issue directory dir and its metadata database,
// reporting what it did to stdout and problems to stderr. It returns the
// exit status: 0 if every issue file was indexed, and 1 otherwise.
func runInit(dir string, stdout, stderr io.Writer) int {
	dir = filepath.Clean(dir)
	result, err := store.Init(dir)

	if err != nil {
		fmt.Fprintf(stderr, "Couldn't create the metadata database: %v\n", err)
		return 1
	}

	if result.CreatedDir {
		fmt.Fprintf(stdout, "Created %s/\n", dir)
	}

	if result.UpdatedGitignore {
		fmt.Fprintf(stdout, "Added the metadata database to %s\n", filepath.Join(dir, ".gitignore"))
	}

	verb := "Created"
	if result.Rebuilt {
		verb = "Rebuilt"
	}

	fmt.Fprintf(stdout, "%s %s\n", verb, filepath.Join(dir, store.DatabaseName))

	noun := "issues"
	if len(result.Indexed) == 1 {
		noun = "issue"
	}

	fmt.Fprintf(stdout, "Indexed %d %s\n", len(result.Indexed), noun)

	if result.Users > 0 {
		fmt.Fprintf(stdout, "Indexed %d %s from the git history\n", result.Users, plural(result.Users, "user"))
	}

	for _, failure := range result.Failed {
		fmt.Fprintf(stderr, "Couldn't index %v\n", failure)
	}

	reportConflicts(stderr, result.UserConflicts)

	if len(result.Failed) > 0 {
		return 1
	}

	return 0
}

// ensureDatabase prepares the metadata database in dir for ish to use, and
// reports whether there's a database to use. If the database doesn't exist
// yet, or has an older schema, it's built with runInit. Otherwise, it's
// brought up to date with any issue files that changed, were added, or were
// removed since it was last updated, and the authors of the repository's
// most recent commits are added to it. Issue files that can't be indexed are reported, but don't stop ish,
// since the database holds every other issue.
func ensureDatabase(dir string, stdout, stderr io.Writer) bool {
	exists, err := store.Exists(dir)

	if err != nil {
		fmt.Fprintf(stderr, "Couldn't read the metadata database: %v\n", err)
		return false
	}

	if exists {
		current, err := store.Current(dir)

		if err != nil {
			fmt.Fprintf(stderr, "Couldn't read the metadata database: %v\n", err)
			fmt.Fprintln(stderr, "Run ish init to rebuild it.")
			return false
		}

		// A database from an older version of ish is rebuilt, since its
		// schema may not match.
		if current {
			return syncDatabase(dir, stdout, stderr)
		}
	}

	runInit(dir, stdout, stderr)

	if exists, err := store.Exists(dir); err != nil || !exists {
		return false
	}

	return true
}

// syncDatabase brings the metadata database in dir up to date with the issue
// files, reporting any changes to stdout and files it couldn't index to
// stderr. It reports whether the database can be used.
func syncDatabase(dir string, stdout, stderr io.Writer) bool {
	result, err := store.Sync(dir)

	if err != nil {
		fmt.Fprintf(stderr, "Couldn't update the metadata database: %v\n", err)
		fmt.Fprintln(stderr, "Run ish init to rebuild it.")
		return false
	}

	for _, change := range []struct {
		ids  []string
		verb string
	}{
		{result.Updated, "Re-indexed changed"},
		{result.Added, "Indexed new"},
		{result.Removed, "Removed deleted"},
	} {
		if len(change.ids) > 0 {
			fmt.Fprintf(stdout, "%s %s: %s\n", change.verb, plural(len(change.ids), "issue"), strings.Join(change.ids, ", "))
		}
	}

	for _, failure := range result.Failed {
		fmt.Fprintf(stderr, "Couldn't index %v\n", failure)
	}

	reportConflicts(stderr, result.UserConflicts)

	// The authors of recent commits are added quietly, other than conflicts.
	// The database can be used without them, so a problem is only reported.
	_, conflicts, err := store.IndexUsers(dir)

	if err != nil {
		fmt.Fprintf(stderr, "Couldn't index the git history's users: %v\n", err)
	}

	reportConflicts(stderr, conflicts)

	return true
}

// plural returns noun, made plural if n isn't 1.
// loadState loads the configuration, from the per-user file in home and the
// project's file in dir, into the application state.
func loadState(home, dir string) (*state.State, error) {
	cfg, err := config.Load(home, dir)
	if err != nil {
		return nil, err
	}

	return state.New(cfg), nil
}

// checkUser returns an error if the configuration doesn't have the name and
// email of the user running ish, which bylines need.
func checkUser(st *state.State) error {
	snap := st.Snapshot()

	var missing []string

	for _, key := range []string{"user.name", "user.email"} {
		if value, _ := snap.LookupString("." + key); strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	verb, pronoun := "isn't", "it"
	if len(missing) > 1 {
		verb, pronoun = "aren't", "them"
	}

	return fmt.Errorf("%s %s configured. Set %s in ~/.ish/config.toml or .ish/config.toml, as in\n\n"+
		"[user]\nname = \"Your Name\"\nemail = \"you@example.com\"\n\nor with git config",
		strings.Join(missing, " and "), verb, pronoun)
}

// reportConflicts reports the users who weren't added to the metadata
// database because another email has their name.
func reportConflicts(stderr io.Writer, conflicts []store.UserConflict) {
	for _, c := range conflicts {
		fmt.Fprintf(stderr, "Skipped user %v\n", c)
	}
}

func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}

	return noun + "s"
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
