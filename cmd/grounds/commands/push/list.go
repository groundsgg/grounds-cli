package push

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

func newList() *cobra.Command {
	var mine bool
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pushes",
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			list, err := c.ListPushes(ctx, mine, limit)
			if err != nil {
				return err
			}
			header := []string{"ID", "TARGET", "STATUS", "CREATED"}
			rows := make([][]any, 0, len(list.Items))
			for _, p := range list.Items {
				rows = append(rows, []any{p.ID, p.Target, p.Status, p.CreatedAt.Format("2006-01-02 15:04")})
			}
			render.Table(cmd.OutOrStdout(), header, rows)
			if list.NextCursor != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "(more available; pagination flag TBD)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&mine, "mine", false, "filter to caller's pushes only")
	cmd.Flags().IntVar(&limit, "limit", 20, "page size")
	return cmd
}
