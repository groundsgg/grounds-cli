package cluster

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
)

func newUp() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Resume the paused workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			s, err := c.ClusterUp(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✔ Active.")
			render.Status(cmd.OutOrStdout(), s)
			return nil
		},
	}
}
