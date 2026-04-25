package main

import (
	"fmt"
	"os"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/cluster"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/logs"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/push"
)

func main() {
	root := commands.NewRootCommand()
	root.AddCommand(commands.NewVersionCommand())
	root.AddCommand(commands.NewCompletionCommand())
	root.AddCommand(commands.NewLoginCommand())
	root.AddCommand(commands.NewLogoutCommand())
	root.AddCommand(commands.NewDoctorCommand())
	root.AddCommand(commands.NewInitCommand())
	root.AddCommand(cluster.NewClusterCommand())
	root.AddCommand(logs.NewLogsCommand())
	root.AddCommand(push.NewPushCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
