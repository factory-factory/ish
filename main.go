package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/factory-factory/ish/app"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"

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
			var model tea.Model

			if len(args) == 1 {
				issue := parse.InitialModel(issueIdFromPath(args[0]))
				issue.SetSave(func(issue models.Issue) error {
					return parse.SaveIssue(parse.IssuesDir, issue)
				})
				model = issue
			} else {
				issues, err := parse.ListIssues(parse.IssuesDir)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}

				model = app.New(parse.IssuesDir, issues)
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
