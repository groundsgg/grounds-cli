package devspace

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

func NewDevspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devspace",
		Short: "DevSpace integration helpers",
	}
	cmd.AddCommand(newGenerate())
	return cmd
}

// Mirrors cluster/cluster.go — same auth + config resolution, same
// shape so callers don't need to know which command-group they're in.
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

func defaultDeviceClient() *auth.DeviceClient {
	return &auth.DeviceClient{
		Issuer:   "https://account.grounds.gg/realms/grounds",
		ClientID: "grounds-cli",
		HTTP:     defaultHTTP(),
	}
}

func defaultHTTP() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
