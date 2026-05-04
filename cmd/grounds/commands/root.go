package commands

import (
	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "grounds",
		Short:         "Grounds developer platform CLI",
		Long:          "Build, deploy, inspect, and troubleshoot Grounds projects from the terminal.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			disableColor, err := cmd.Flags().GetBool("no-color")
			if err != nil {
				return err
			}
			render.SetEnabled(disableColor)
			if cmd != nil && cmd.HasParent() && cmd.Parent().PersistentPreRunE != nil && cmd.Parent() != cmd.Root() {
				return cmd.Parent().PersistentPreRunE(cmd, args)
			}
			return nil
		},
	}
	cmd.PersistentFlags().String("api-url", "", "override API endpoint (also GROUNDS_API_URL)")
	cmd.PersistentFlags().String("output", "table", "output format: table | json | yaml")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "debug logging to stderr")
	cmd.PersistentFlags().Bool("no-color", false, "disable colour output")
	cmd.PersistentFlags().String("config", "", "alternative config directory (also GROUNDS_CONFIG_DIR)")
	cmd.PersistentFlags().String("project", "", "project id to scope this call (also GROUNDS_PROJECT)")
	return cmd
}
