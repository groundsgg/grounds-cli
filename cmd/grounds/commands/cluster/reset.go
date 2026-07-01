package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/render"
	"github.com/groundsgg/grounds-cli/internal/ui"
)

func newReset() *cobra.Command {
	var yes string
	var bundleRef string
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Wipe the workspace back to a clean bundle base",
		Long: `Reset a platform-bundle workspace back to a clean bundle base:
services + lobby + velocity + infra, discarding pushed apps and any drift.

Forge tears down the vCluster namespace, then re-provisions from the bundle
ref (--bundle, default main) with no engineer overrides. The workspace keeps
its namespace and profile — only the contents are reset to zero.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			// Need the namespace (for the confirm) and profile (only
			// platform-bundle can be reset — forge enforces this too).
			s, err := c.GetCluster(ctx)
			if err != nil {
				return err
			}
			if s.Profile != "platform-bundle" {
				return fmt.Errorf(
					"reset only supports platform-bundle workspaces (this is %q); use `grounds cluster delete` then `grounds cluster up`",
					s.Profile,
				)
			}

			// Destructive: pushed apps + everything in the vCluster are discarded.
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				if yes != s.Namespace {
					return errors.New("non-interactive reset requires --yes <namespace>")
				}
			} else {
				render.StatusLine(w, render.StatusWarn, "Workspace",
					"This will wipe "+s.Namespace+" back to a clean bundle base (pushed apps + data discarded)")
				if err := ui.AskTypeName(os.Stdin, w, s.Namespace, s.Namespace); err != nil {
					return err
				}
			}

			if _, err := c.ClusterReset(ctx, &api.ClusterResetRequest{Bundle: bundleRef}); err != nil {
				return err
			}
			render.StatusLine(w, render.StatusOK, "Workspace",
				"resetting — tearing down and re-provisioning the bundle base…")
			// Forge holds the row at `creating` across teardown + re-provision,
			// so this is the same single creating→active poll as `cluster up`.
			final, err := waitForBundle(ctx, c, w)
			if err != nil {
				return err
			}
			renderBundleStatus(w, final)
			if final.State == "failed" {
				return fmt.Errorf("reset failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&yes, "yes", "", "namespace to reset (required in non-interactive mode)")
	cmd.Flags().StringVar(&bundleRef, "bundle", "", "PlatformBundle ref to reset to (default main)")
	return cmd
}
