package push

import "github.com/spf13/cobra"

func newRetry() *cobra.Command {
	return &cobra.Command{Use: "retry <pushId>", Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error { return nil }}
}
