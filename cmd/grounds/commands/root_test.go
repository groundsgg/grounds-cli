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

func TestRootOutputFlagMentionsDataCommands(t *testing.T) {
	root := NewRootCommand()
	flag := root.PersistentFlags().Lookup("output")
	if flag == nil {
		t.Fatal("missing output flag")
	}
	if got := flag.Usage; got != "output format for data commands: table | json | yaml" {
		t.Fatalf("output flag usage = %q", got)
	}
}
