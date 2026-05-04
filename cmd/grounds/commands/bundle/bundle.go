package bundle

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

func NewBundleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bundle",
		Short:   "Inspect available platform-test bundles",
		Example: "  grounds bundle list\n  grounds bundle show main\n  grounds bundle show 0.4.0",
	}
	cmd.AddCommand(newList(), newShow())
	return cmd
}

// Mirrors cluster/cluster.go's buildClient.
func buildClient(_ context.Context, cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{
			Store:  auth.NewStore(cfg.Dir),
			Device: defaultDeviceClient(),
		}
	}
	c := api.New(cfg.APIURL, ts)
	if p := projectIDFlag(cmd); p != "" {
		c.ProjectID = p
	}
	return c, nil
}

func projectIDFlag(cmd *cobra.Command) string {
	if cmd != nil {
		if p, _ := cmd.Flags().GetString("project"); p != "" {
			return p
		}
	}
	return os.Getenv("GROUNDS_PROJECT")
}

func defaultDeviceClient() *auth.DeviceClient {
	return &auth.DeviceClient{
		Issuer:   "https://account.grounds.gg/realms/grounds",
		ClientID: "grounds-cli",
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}
