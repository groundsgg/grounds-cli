package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/bundle"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/cluster"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/devspace"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/logs"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/preview"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/push"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/workspace"
	"github.com/groundsgg/grounds-cli/internal/observability"
)

func main() {
	// Init Sentry/GlitchTip first so panics during command setup are
	// captured. No-op when DSN is unset (dev `go build`, smoke tests).
	if err := observability.Init(); err != nil {
		// Don't bail the CLI on observability failure — log + continue.
		fmt.Fprintf(os.Stderr, "warn: sentry init failed: %v\n", err)
	}
	defer observability.Flush()
	defer observability.Recover()

	root := commands.NewRootCommand()
	root.AddCommand(commands.NewVersionCommand())
	root.AddCommand(commands.NewCompletionCommand())
	root.AddCommand(commands.NewLoginCommand())
	root.AddCommand(commands.NewLogoutCommand())
	root.AddCommand(commands.NewDoctorCommand())
	root.AddCommand(commands.NewInitCommand())
	root.AddCommand(bundle.NewBundleCommand())
	root.AddCommand(cluster.NewClusterCommand())
	root.AddCommand(devspace.NewDevspaceCommand())
	root.AddCommand(logs.NewLogsCommand())
	root.AddCommand(push.NewPushCommand())
	root.AddCommand(preview.NewPreviewCommand())
	root.AddCommand(workspace.NewWorkspaceCommand())

	// Execute() returns the error from the matched cobra subcommand.
	// We pass the matched-command path (e.g. "grounds push") into the
	// captured event as a tag so issues group by subcommand.
	executedCmd, err := root.ExecuteC()
	if err != nil {
		if errors.Is(err, commands.ErrDoctorIssuesFound) {
			os.Exit(1)
		}
		commandPath := ""
		if executedCmd != nil {
			commandPath = executedCmd.CommandPath()
		}
		observability.CaptureExecuteError(err, commandPath)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
