package cluster

import (
	"context"
	"fmt"

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
					fmt.Fprintln(cmd.OutOrStdout(), "→ no workspace yet. Run 'grounds push' to create one.")
					return nil
				}
				return err
			}
			render.Status(cmd.OutOrStdout(), s)
			return nil
		},
	}
}
