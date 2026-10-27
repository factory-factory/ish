package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/factory-factory/ish/app"
	"github.com/factory-factory/ish/config"
	"github.com/factory-factory/ish/filter"
	"github.com/factory-factory/ish/parse"
	"github.com/factory-factory/ish/store"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [-q query]... [-c attribute]...",
	Short: "Print the issues in the project",
	Long: `Print the issues in the project, one per line, ordered by ID. Each line
has the issue's ID, the date it was created in RFC 3339 format, its author's
email, a column for each attribute named with -c, in the order they're
named, and the issue's title, separated by tabs. An attribute with several
values is printed with its values separated by commas, and attributes
missing from a particular issue are printed as empty columns. Naming an
attribute that no issue has is an error. Tabs and newlines in values are
printed as spaces.

Each -q query is a filter expression, like the ones in the issue list's
custom filter. If there's more than one, an issue is printed only if it
matches every one.`,
	Example: `  ish list
  ish list -q '{state != "closed"}'
  ish list -q '{state != "closed"}' -q '{assignees = "alice@example.com"}'
  ish list -c assignees -c state`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		queries, _ := cmd.Flags().GetStringArray("query")
		columns, _ := cmd.Flags().GetStringArray("column")

		os.Exit(runList(parse.IssuesDir, config.UserHome(), queries, columns, cmd.OutOrStdout(), cmd.ErrOrStderr()))
	},
}

// runList prints the issues in the issue directory dir that match every
// query to stdout, with a column for each of the attributes named by
// columns, and reports problems to stderr. The configuration is read from the
// per-user file in home and the project's file in dir, for the queries'
// lookups. It returns the exit status.
func runList(dir, home string, queries, columns []string, stdout, stderr io.Writer) int {
	where, err := parseQueries(queries)
	if err != nil {
		fmt.Fprintf(stderr, "Couldn't parse the %v\n", err)
		return 1
	}

	// Listing issues shouldn't create an issue directory where there isn't
	// one, as ish init does.
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(stderr, "%s doesn't exist; run ish init to create it\n", dir)
		return 1
	} else if err != nil {
		fmt.Fprintf(stderr, "Couldn't read %s: %v\n", dir, err)
		return 1
	}

	st, err := loadState(home, dir)
	if err != nil {
		fmt.Fprintf(stderr, "Couldn't read the configuration: %v\n", err)
		return 1
	}

	// Updates to the database are reported to stderr, so stdout has only
	// the listing.
	if !ensureDatabase(dir, stderr, stderr) {
		return 1
	}

	if where, err = app.ResolveFilter(where, st, dir); err != nil {
		fmt.Fprintf(stderr, "Couldn't apply the query: %v\n", err)
		return 1
	}

	summaries, err := store.ListIssues(dir, where)
	if err != nil {
		fmt.Fprintf(stderr, "Couldn't list the issues: %v\n", err)
		return 1
	}

	attributes, err := store.Attributes(dir, columns)
	if err != nil {
		fmt.Fprintf(stderr, "Couldn't read the issues' attributes: %v\n", err)
		return 1
	}

	if missing := missingColumns(columns, attributes); len(missing) > 0 {
		fmt.Fprintf(stderr, "No issue has the %s %s\n", plural(len(missing), "attribute"), strings.Join(missing, ", "))
		return 1
	}

	for _, s := range summaries {
		fields := []string{s.ID, s.Opened.Format(time.RFC3339), s.Author}

		for _, key := range columns {
			fields = append(fields, strings.Join(attributes[s.ID][key], ","))
		}

		fields = append(fields, s.Title)

		for i, field := range fields {
			fields[i] = blankControls.Replace(field)
		}

		fmt.Fprintln(stdout, strings.Join(fields, "\t"))
	}

	return 0
}

// blankControls replaces the tabs and newlines in a value with spaces, so
// each value stays in its own column and each issue on its own line.
var blankControls = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ")

// missingColumns returns the attribute keys in columns that no issue in
// attributes, by issue ID and then key, has, in the order they're named.
func missingColumns(columns []string, attributes map[string]map[string][]string) []string {
	var missing []string

	for _, key := range columns {
		found := false

		for _, values := range attributes {
			if _, ok := values[key]; ok {
				found = true
				break
			}
		}

		if !found && !slices.Contains(missing, key) {
			missing = append(missing, key)
		}
	}

	return missing
}

// parseQueries parses each of the filter expressions in queries, and returns
// an expression that matches the issues that match all of them, or nil, to
// match every issue, if there are none.
func parseQueries(queries []string) (filter.Node, error) {
	var where filter.Node

	for _, query := range queries {
		node, err := filter.Parse(query)
		if err != nil {
			return nil, fmt.Errorf("query %q: %w", query, err)
		}

		if where == nil {
			where = node
		} else {
			where = &filter.AndExpression{LeftExpression: where, RightExpression: node}
		}
	}

	return where, nil
}

func init() {
	listCmd.Flags().StringArrayP("query", "q", nil, "list only the issues that match the filter `expression`")
	listCmd.Flags().StringArrayP("column", "c", nil, "add a column with the `attribute`'s values")
	rootCmd.AddCommand(listCmd)
}
