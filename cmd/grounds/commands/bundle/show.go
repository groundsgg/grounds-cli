package bundle

import (
	"context"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newShow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <ref>",
		Short: "Show the components in a bundle",
		Long: `Fetches the parsed bundle.yaml at the given ref and prints the
component table. <ref> accepts the same shapes as ` + "`grounds cluster up --bundle`" + `:
semver, "v…", the full release tag, or "main" for the latest commit.

Examples:

  grounds bundle show 0.4.0
  grounds bundle show main`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}
			b, err := c.GetBundle(ctx, args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Bundle:    %s\n", b.Metadata.Version)
			if b.Metadata.Description != "" {
				fmt.Fprintf(out, "About:     %s\n", b.Metadata.Description)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Components:")
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  NAME\tTYPE\tIMAGE\tCHART\tDEVSPACE")

			names := make([]string, 0, len(b.Components))
			for k := range b.Components {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, name := range names {
				comp := b.Components[name]
				devspace := "-"
				if comp.Devspace != nil && comp.Devspace.Workflow != "" {
					devspace = comp.Devspace.Workflow
				}
				marker := name
				if comp.Optional {
					marker = name + " (optional)"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s:%s\t%s\t%s\n",
					marker, comp.Type, comp.Image, comp.Version, comp.Chart.Version, devspace,
				)
			}
			return w.Flush()
		},
	}
	return cmd
}
