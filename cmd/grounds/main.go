package main

import (
	"fmt"
	"os"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands"
)

func main() {
	root := commands.NewRootCommand()
	root.AddCommand(commands.NewVersionCommand())
	root.AddCommand(commands.NewCompletionCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
