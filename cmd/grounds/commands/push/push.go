package push

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/gradle"
)

func NewPushCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "push", Short: "Build and deploy the current project"}
	cmd.AddCommand(newPush(), newRetry(), newList())
	return cmd
}

func newPush() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "push [--target=dev]",
		Short: "Build via Gradle plugin and deploy to a target",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			wrapper, err := gradle.FindWrapper(cwd)
			if err != nil {
				return fmt.Errorf("%w\n  → not a Gradle project? Run 'grounds init' to scaffold, or cd to your project root", err)
			}
			ctx := context.Background()
			args := []string{"groundsPush", "--target=" + target}
			return gradle.Run(ctx, wrapper, args, cmd.OutOrStdout(), cmd.ErrOrStderr(), 0)
		},
	}
	cmd.Flags().StringVar(&target, "target", "dev", "deploy target: dev")
	return cmd
}

// defaultDevice mirrors login.go (same issuer, same client ID).
// Lifted here so subcommands don't depend on the commands package.
func defaultDevice() *auth.DeviceClient {
	// Avoid pkg-level constants from sibling pkg; hardcode same values
	return &auth.DeviceClient{
		Issuer:   "https://account.grounds.gg/realms/grounds",
		ClientID: "grounds-cli",
		HTTP:     defaultHTTP(),
	}
}

func defaultHTTP() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
