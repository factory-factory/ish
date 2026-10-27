package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/factory-factory/ish/parse"
	"github.com/factory-factory/ish/store"
	"github.com/spf13/cobra"
)

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

	if result.Modifications > 0 {
		fmt.Fprintf(stdout, "Indexed %d %s from the git history\n", result.Modifications, plural(result.Modifications, "change"))
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

func init() {
	rootCmd.AddCommand(initCmd)
}
