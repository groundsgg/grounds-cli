package bundle

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/internal/projectscope"
	"github.com/groundsgg/grounds-cli/internal/api"
)

func NewBundleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Inspect available platform-test bundles",
		Example: "  grounds bundle list\n  grounds bundle show main\n  grounds bundle show 0.4.0",
	}
	cmd.AddCommand(newList(), newShow(), newRestore())
	return cmd
}

func buildClient(ctx context.Context, cmd *cobra.Command) (*api.Client, error) {
	c, _, _, err := projectscope.BuildClient(ctx, cmd)
	return c, err
}
