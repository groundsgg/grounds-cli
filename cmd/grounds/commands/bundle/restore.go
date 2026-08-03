package bundle

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
)

func newRestore() *cobra.Command {
	return &cobra.Command{
		Use:     "restore <component>",
		Short:   "Return one component to its managed bundle image",
		Example: "  grounds bundle restore service-config",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := buildClient(context.Background(), cmd)
			if err != nil {
				return err
			}
			result, err := c.RestoreBundleComponent(context.Background(), args[0])
			if err != nil {
				return err
			}
			render.StatusLine(
				cmd.OutOrStdout(),
				render.StatusOK,
				"Bundle",
				fmt.Sprintf("restoring %s to its managed bundle image…", args[0]),
			)
			if result.Poll != "" {
				render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Follow progress with "+render.Command("grounds cluster status")+".")
			}
			return nil
		},
	}
}
