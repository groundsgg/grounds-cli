package push

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
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
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			ts := api.NewEnvTokenSource()
			if ts == nil {
				ts = &auth.FileTokenSource{Store: auth.NewStore(cfg.Dir), Device: defaultDevice()}
			}
			c := api.New(cfg.APIURL, ts)
			c.ProjectID = projectIDFrom(cmd)
			p, err := c.RetryPush(ctx, args[0])
			if err != nil {
				return err
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Push", "Retry triggered for "+p.ID)
			render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Status: "+p.Status)
			if !follow {
				return nil
			}
			tok, err := ts.Token(ctx)
			if err != nil {
				return err
			}
			stream := &sse.Stream{
				URL:    cfg.APIURL + "/v1/pushes/" + p.ID + "/logs",
				Token:  tok,
				Client: defaultHTTP(),
			}
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
