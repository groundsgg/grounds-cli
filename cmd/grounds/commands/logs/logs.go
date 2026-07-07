package logs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/internal/projectscope"
	"github.com/groundsgg/grounds-cli/internal/sse"
)

func NewLogsCommand() *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:     "logs <pushId>",
		Short:   "Stream push logs, or deployment logs with `grounds logs deployment <name>`",
		Example: "  grounds logs <pushId>\n  grounds logs <pushId> --follow\n  grounds logs deployment <name>",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return streamLogs(cmd.Context(), cmd, args[0], "push", follow)
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
			return streamLogs(cmd.Context(), cmd, args[0], "deployment", follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", true, "follow")
	return cmd
}

func streamLogs(ctx context.Context, cmd *cobra.Command, target, kind string, follow bool) error {
	stream, err := buildStream(ctx, cmd, target, kind)
	if err != nil {
		return err
	}
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

func buildStream(ctx context.Context, cmd *cobra.Command, target, kind string) (*sse.Stream, error) {
	c, _, _, err := projectscope.BuildClient(ctx, cmd)
	if err != nil {
		return nil, err
	}
	tok, err := c.Tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	var path string
	switch kind {
	case "push":
		path = "/v1/pushes/" + url.PathEscape(target) + "/logs"
	case "deployment":
		path = "/v1/deployments/" + url.PathEscape(target) + "/logs"
	default:
		return nil, fmt.Errorf("unknown log target kind: %s", kind)
	}
	return &sse.Stream{URL: c.ScopedURL(path), Token: tok, Client: defaultHTTP()}, nil
}

func defaultHTTP() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
