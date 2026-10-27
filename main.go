package main

import (
	"os"

	"github.com/factory-factory/ish/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
