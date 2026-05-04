package push

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/gradle"
	"github.com/groundsgg/grounds-cli/internal/render"
)

func NewPushCommand() *cobra.Command {
	cmd := newPush()
	cmd.Example = "  grounds push\n  grounds push --target=staging\n  grounds push list --mine"
	cmd.AddCommand(newRetry(), newList())
	return cmd
}

func newPush() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:     "push [--target=dev|staging]",
		Short:   "Build via Gradle plugin and deploy to a target",
		Example: "  grounds push\n  grounds push --target=staging",
		Long: `Build the current project with the grounds-push Gradle plugin and deploy it.

Targets:
  dev     — long-lived, lands in your personal namespace (user-<handle>).
  staging — ephemeral preview env, fresh namespace per push, auto-deleted after 7 days.
            Public URL pattern: <name>-pr<id>.dev.grnds.io.`,
		Args: cobra.NoArgs,
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
				return fmt.Errorf("%w\n    ! Not a Gradle project? Run %s to scaffold, or cd to your project root.", err, render.Command("grounds init"))
			}
			ctx := context.Background()

			// Pre-refresh: the Gradle plugin reads credentials.json
			// directly via CredentialResolver and rejects expired
			// access tokens (Keycloak default lifetime is ~5min).
			// Force a refresh here so the file is fresh before
			// Gradle reads it. Skip when GROUNDS_TOKEN is set —
			// the plugin uses the env var directly, no file involved.
			if os.Getenv("GROUNDS_TOKEN") == "" {
				cfg, err := config.Load("")
				if err != nil {
					return err
				}
				src := &auth.FileTokenSource{
					Store:  auth.NewStore(cfg.Dir),
					Device: defaultDevice(),
				}
				if _, err := src.Token(ctx); err != nil {
					return authRefreshError(err)
				}
			}

			args := []string{"groundsPush", "--target=" + target}
			return gradle.Run(ctx, wrapper, args, cmd.OutOrStdout(), cmd.ErrOrStderr(), 0)
		},
	}
	cmd.Flags().StringVar(&target, "target", "dev", "deploy target: dev (persistent personal ns) or staging (ephemeral preview env, 7d TTL)")
	_ = cmd.RegisterFlagCompletionFunc("target", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "staging"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func authRefreshError(err error) error {
	return fmt.Errorf("auth refresh failed: %w\n    ! Run %s to re-authenticate.", err, render.Command("grounds login"))
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
