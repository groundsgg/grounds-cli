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
		Use:   "push [--target=dev|staging]",
		Short: "Build via Gradle plugin and deploy to a target",
		Long: `Build the current project with the grounds-push Gradle plugin and deploy it.

Targets:
  dev     — long-lived, lands in your personal namespace (user-<handle>).
  staging — ephemeral preview env, fresh namespace per push, auto-deleted after 7 days.
            Public URL pattern: <name>-pr<id>.dev.grnds.io.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if target != "dev" && target != "staging" {
				return fmt.Errorf("invalid --target %q: must be \"dev\" or \"staging\"", target)
			}
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
	cmd.Flags().StringVar(&target, "target", "dev", "deploy target: dev (persistent personal ns) or staging (ephemeral preview env, 7d TTL)")
	return cmd
}

// projectIDFrom resolves the global --project flag, falling back to
// the GROUNDS_PROJECT env var. Empty string when neither is set —
// forge then uses the caller's default project.
func projectIDFrom(cmd *cobra.Command) string {
	if cmd != nil {
		if p, _ := cmd.Flags().GetString("project"); p != "" {
			return p
		}
	}
	return os.Getenv("GROUNDS_PROJECT")
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
