package cluster

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/render"
)

func newStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show workspace state, deployments, quota",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			s, err := c.GetCluster(ctx)
			if err != nil {
				if apiErr, ok := err.(*api.Error); ok && apiErr.StatusCode == 404 {
					render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Workspace", "No workspace found")
					render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Run "+render.Command("grounds push")+" to create one.")
					return nil
				}
				return err
			}
			render.Status(cmd.OutOrStdout(), s)
			return nil
		},
	}
}
