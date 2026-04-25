package commands

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grounds",
		Short: "Grounds Internal Developer Platform CLI",
		Long:  "Drives the Grounds platform from the terminal.",
		SilenceUsage: true,
	}
	cmd.PersistentFlags().String("api-url", "", "override API endpoint (also GROUNDS_API_URL)")
	cmd.PersistentFlags().String("output", "table", "output format: table | json | yaml")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "debug logging to stderr")
	cmd.PersistentFlags().Bool("no-color", false, "disable colour output")
	cmd.PersistentFlags().String("config", "", "alternative config directory (also GROUNDS_CONFIG_DIR)")
	return cmd
}
