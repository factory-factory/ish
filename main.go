package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/factory-factory/ish/app"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
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
			if !ensureDatabase(parse.IssuesDir, cmd.OutOrStdout(), cmd.ErrOrStderr()) {
				os.Exit(1)
			}

			var model tea.Model

			if len(args) == 1 {
				issue := parse.InitialModel(issueIdFromPath(args[0]))
				issue.SetSave(func(issue models.Issue) error {
					return store.SaveIssue(parse.IssuesDir, issue)
				})
				issue.SetAuthor(app.CurrentUser(parse.IssuesDir))
				model = issue
			} else {
				list, err := app.New(parse.IssuesDir)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				model = list
			}

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

	for _, failure := range result.Failed {
		fmt.Fprintf(stderr, "Couldn't index %v\n", failure)
	}

	if len(result.Failed) > 0 {
		return 1
	}

	return 0
}

// ensureDatabase prepares the metadata database in dir for ish to use, and
// reports whether there's a database to use. If the database doesn't exist
// yet, or has an older schema, it's built with runInit. Otherwise, it's
// brought up to date with any issue files that changed, were added, or were
// removed since it was last updated. Issue files that can't be indexed are reported, but don't stop ish,
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

	return true
}

// plural returns noun, made plural if n isn't 1.
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
