package workspace

import (
	"strings"
	"testing"
)

func TestWorkspaceCommandDefinesSubcommands(t *testing.T) {
	cmd := NewWorkspaceCommand()

	for _, name := range []string{"add", "list", "enable", "doctor", "scan"} {
		sub, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("Find(%q) error = %v", name, err)
		}
		if sub.Name() != name {
			t.Fatalf("Find(%q) = %q, want %q", name, sub.Name(), name)
		}
	}
}

func TestWorkspaceScanRequiresRoot(t *testing.T) {
	cmd := NewWorkspaceCommand()
	cmd.SetArgs([]string{"scan"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing root error")
	}
	if !strings.Contains(err.Error(), "at least one root") {
		t.Fatalf("error = %q, want root requirement", err)
	}
}
