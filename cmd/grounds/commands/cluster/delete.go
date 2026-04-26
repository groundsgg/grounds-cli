package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/groundsgg/grounds-cli/internal/ui"
)

func newDelete() *cobra.Command {
	var yes string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Permanently delete the workspace and all its data",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}

			// Need the namespace name to send X-Confirm-Delete.
			s, err := c.GetCluster(ctx)
			if err != nil {
				return err
			}

			// Non-TTY: require --yes <namespace>
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				if yes != s.Namespace {
					return errors.New("non-interactive delete requires --yes <namespace>")
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "⚠  This will permanently delete", s.Namespace, "and all its data.")
				if err := ui.AskTypeName(os.Stdin, cmd.OutOrStdout(), s.Namespace, s.Namespace); err != nil {
					return err
				}
			}

			res, err := c.ClusterDelete(ctx, s.Namespace)
			if err != nil {
				return err
			}
			switch res.State {
			case "deleted":
				fmt.Fprintln(cmd.OutOrStdout(), "✔ Deleted.")
			case "deleting":
				fmt.Fprintln(cmd.OutOrStdout(), "→ Stuck Terminating; will be cleaned up by the janitor on next run.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&yes, "yes", "", "namespace to delete (required in non-interactive mode)")
	return cmd
}
