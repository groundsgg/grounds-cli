package bundle

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
)

func newList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List released platform-bundle versions",
		Long: `Lists released library-platform-bundle versions, newest first.
Drafts and prereleases are filtered out. The version with (latest) is
the same one ` + "`grounds cluster up --bundle main`" + ` would track today.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			releases, err := c.ListBundleReleases(ctx)
			if err != nil {
				return err
			}
			if len(releases) == 0 {
				render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Bundle", "No released bundles found")
				render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Try "+render.Command("grounds bundle show main")+" to inspect the current bundle.")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "VERSION\tPUBLISHED\tURL")
			for _, r := range releases {
				marker := ""
				if r.IsLatest {
					marker = " (latest)"
				}
				fmt.Fprintf(w, "%s%s\t%s\t%s\n",
					r.Version, marker,
					r.PublishedAt.Format("2006-01-02"),
					r.HtmlURL,
				)
			}
			return w.Flush()
		},
	}
	return cmd
}
