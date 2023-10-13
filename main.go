package main

import (
	"fmt"
	"os"

	"github.com/iridescent-software/ish/parse"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:              "ish",
		Short:            "A file-based distributed issue tracker",
		SilenceErrors:    false,
		SilenceUsage:     false,
		TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			p := tea.NewProgram(parse.InitialModel())
			if _, err := p.Run(); err != nil {
				fmt.Printf("Oh no, an error:\n%v", err)
				os.Exit(1)
			}
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
