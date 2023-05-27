package main

import (
	"os"
	"fmt"

	// tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:	          "ish",
		Short:            "A file-based distributed issue tracker",
		SilenceErrors:    false,
		SilenceUsage:     false,
		TraverseChildren: true,
		Run: func (cmd *cobra.Command, args []string) {
			fmt.Println("Hello, world!")
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

