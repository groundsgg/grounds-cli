package cluster

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
)

func newUp() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "up [--profile=minigame|platform]",
		Short: "Spawn or resume the workspace",
		Long: `Create the workspace if it doesn't exist, or resume it from a paused state.

Profiles:
  minigame  — namespace-scoped sandbox (default). Cheap, fast, isolated by RBAC.
  platform  — full per-developer vCluster with the Grounds platform chart
              installed inside. Heavier (one-time ~90s spawn) but lets you
              run platform plugins / agones / mc-router locally.

Profile is locked once a workspace exists. To switch, ` + "`grounds cluster delete`" + ` and re-up.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profile != "" && profile != "minigame" && profile != "platform" {
				return fmt.Errorf("invalid --profile %q: must be \"minigame\" or \"platform\"", profile)
			}
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			s, err := c.ClusterUp(ctx, profile)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✔ Active.")
			render.Status(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "workspace profile: minigame (default) or platform (vCluster)")
	return cmd
}
