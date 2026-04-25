package logs

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/sse"
)

func NewLogsCommand() *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs <pushId>",
		Short: "Stream push logs (or deployment logs via 'grounds logs deployment <name>')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return streamLogs(cmd.Context(), args[0], "push", follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", true, "follow until terminal status")
	cmd.AddCommand(newDeployment())
	return cmd
}

func newDeployment() *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "deployment <name>",
		Short: "Stream deployment logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return streamLogs(cmd.Context(), args[0], "deployment", follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", true, "follow")
	return cmd
}

func streamLogs(ctx context.Context, target, kind string, follow bool) error {
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{Store: auth.NewStore(cfg.Dir), Device: defaultDevice()}
	}
	tok, err := ts.Token(ctx)
	if err != nil {
		return err
	}
	var path string
	switch kind {
	case "push":
		path = "/v1/pushes/" + target + "/logs"
	case "deployment":
		path = "/v1/deployments/" + target + "/logs"
	}
	stream := &sse.Stream{URL: cfg.APIURL + path, Token: tok, Client: defaultHTTP()}
	if !follow {
		// No-follow: read until first terminal status, then exit.
	}
	return stream.Subscribe(ctx, func(ev *sse.Event) error {
		if sse.Render(os.Stdout, ev) {
			return io.EOF
		}
		return nil
	})
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
