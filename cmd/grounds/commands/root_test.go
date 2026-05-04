package commands

import (
	"testing"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func TestRootCommandAppliesNoColorFlag(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = false }()

	root := NewRootCommand()
	root.AddCommand(&cobra.Command{
		Use: "noop",
		Run: func(*cobra.Command, []string) {},
	})
	root.SetArgs([]string{"--no-color", "noop"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !color.NoColor {
		t.Fatal("expected --no-color to disable color output")
	}
}

func TestRootCommandDoesNotAdvertiseUnusedOutputFlag(t *testing.T) {
	root := NewRootCommand()
	if flag := root.PersistentFlags().Lookup("output"); flag != nil {
		t.Fatalf("unexpected unused output flag: %q", flag.Usage)
	}
}
