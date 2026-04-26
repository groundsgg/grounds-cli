package cluster

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
)

func newDown() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Pause the workspace immediately",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			s, err := c.ClusterDown(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✔ Paused.")
			render.Status(cmd.OutOrStdout(), s)
			return nil
		},
	}
}
