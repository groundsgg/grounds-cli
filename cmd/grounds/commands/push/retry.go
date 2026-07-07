package push

import (
	"context"
	"io"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/internal/projectscope"
	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/render"
	"github.com/groundsgg/grounds-cli/internal/sse"
)

func newRetry() *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "retry <pushId>",
		Short: "Re-run a failed deploy without rebuilding",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, _, _, err := projectscope.BuildClient(ctx, cmd)
			if err != nil {
				return err
			}
			p, err := c.RetryPush(ctx, args[0])
			if err != nil {
				return err
			}
			renderRetryTriggered(cmd.OutOrStdout(), p)
			if !follow {
				return nil
			}
			tok, err := c.Tokens.Token(ctx)
			if err != nil {
				return err
			}
			stream := retryLogStream(c, p.ID, tok)
			return stream.Subscribe(ctx, func(ev *sse.Event) error {
				if sse.Render(os.Stdout, ev) {
					return io.EOF
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&follow, "follow", true, "stream logs after retry")
	return cmd
}

func retryLogStream(c *api.Client, pushID, token string) *sse.Stream {
	return &sse.Stream{
		URL:    c.ScopedURL("/v1/pushes/" + url.PathEscape(pushID) + "/logs"),
		Token:  token,
		Client: defaultHTTP(),
	}
}

func renderRetryTriggered(out io.Writer, p *api.Push) {
	render.StatusLine(out, render.StatusOK, "Push", "Retry triggered for "+p.ID)
	render.DetailLine(out, render.StatusOK, "Status: "+p.Status)
}
