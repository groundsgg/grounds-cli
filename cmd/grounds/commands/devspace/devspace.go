package devspace

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/internal/projectscope"
	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/project"
)

func NewDevspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "devspace",
		Short:   "DevSpace integration helpers",
		Example: "  grounds devspace generate plugin-social --bundle main\n  grounds devspace generate plugin-social --override ./me.yaml",
	}
	cmd.AddCommand(newGenerate())
	return cmd
}

// Mirrors cluster/cluster.go — same auth + config resolution, same
// shape so callers don't need to know which command-group they're in.
func buildClient(ctx context.Context, cmd *cobra.Command) (*api.Client, *config.Config, project.Selection, error) {
	return projectscope.BuildClient(ctx, cmd)
}
