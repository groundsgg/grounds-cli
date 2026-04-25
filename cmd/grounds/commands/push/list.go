package push

import "github.com/spf13/cobra"

func newList() *cobra.Command {
	return &cobra.Command{Use: "list", Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return nil }}
}
