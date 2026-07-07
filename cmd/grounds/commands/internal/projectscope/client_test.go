package projectscope

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestProjectFlagReadsInheritedPersistentFlag(t *testing.T) {
	root := &cobra.Command{Use: "grounds"}
	root.PersistentFlags().String("project", "", "project id")
	child := &cobra.Command{Use: "child"}
	child.Run = func(*cobra.Command, []string) {}
	root.AddCommand(child)
	root.SetArgs([]string{"--project", "project-1", "child"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := projectFlag(child); got != "project-1" {
		t.Fatalf("projectFlag = %q, want project-1", got)
	}
}
