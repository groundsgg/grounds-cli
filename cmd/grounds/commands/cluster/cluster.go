package cluster

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/internal/projectscope"
	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/project"
)

func NewClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "Manage your dev workspace lifecycle",
		Example: "  grounds cluster status\n  grounds cluster up\n  grounds cluster reset\n  grounds cluster down\n  grounds cluster delete",
	}
	cmd.AddCommand(newUp(), newDown(), newReset(), newDelete(), newStatus())
	return cmd
}

// buildClient is the shared helper every cluster subcommand uses to
// resolve config + auth + API client. The cobra command is passed in
// so we can read the global --project flag.
func buildClient(ctx context.Context, cmd *cobra.Command) (*api.Client, *config.Config, project.Selection, error) {
	return projectscope.BuildClient(ctx, cmd)
}
