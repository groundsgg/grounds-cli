package commands

import (
	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

func NewLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			if err := auth.NewStore(cfg.Dir).Delete(); err != nil {
				return err
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Auth", "Logged out")
			return nil
		},
	}
}
