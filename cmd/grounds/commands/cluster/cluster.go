package cluster

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
)

func NewClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster",
		Short:   "Manage your dev workspace lifecycle",
		Example: "  grounds cluster status\n  grounds cluster up\n  grounds cluster down\n  grounds cluster delete",
	}
	cmd.AddCommand(newUp(), newDown(), newDelete(), newStatus())
	return cmd
}

// buildClient is the shared helper every cluster subcommand uses to
// resolve config + auth + API client. The cobra command is passed in
// so we can read the global --project flag.
func buildClient(_ context.Context, cmd *cobra.Command) (*api.Client, *config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{
			Store:  auth.NewStore(cfg.Dir),
			Device: defaultDeviceClient(),
		}
	}
	c := api.New(cfg.APIURL, ts)
	c.ProjectID = resolveProjectID(cmd)
	return c, cfg, nil
}

// resolveProjectID picks --project, falling back to the GROUNDS_PROJECT
// env var. Empty string when neither is set, in which case forge falls
// back to the caller's default project.
func resolveProjectID(cmd *cobra.Command) string {
	if cmd != nil {
		if p, _ := cmd.Flags().GetString("project"); p != "" {
			return p
		}
	}
	return envOr("GROUNDS_PROJECT", "")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// defaultDeviceClient mirrors login.go (same issuer, same client ID).
// Lifted here so subcommands don't depend on the commands package.
func defaultDeviceClient() *auth.DeviceClient {
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
