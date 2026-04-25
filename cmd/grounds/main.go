package main

import (
	"fmt"
	"os"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/cluster"
)

func main() {
	root := commands.NewRootCommand()
	root.AddCommand(commands.NewVersionCommand())
	root.AddCommand(commands.NewCompletionCommand())
	root.AddCommand(commands.NewLoginCommand())
	root.AddCommand(commands.NewLogoutCommand())
	root.AddCommand(cluster.NewClusterCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
