package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
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

func TestWorkspaceScanPreservesExistingMappings(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("GROUNDS_CONFIG_DIR", configDir)
	existingPath := filepath.Join(t.TempDir(), "plugin-chat")
	if err := internalworkspace.Save("", &internalworkspace.Config{Repos: map[string]internalworkspace.Repo{
		"plugin-chat": {
			Path:     existingPath,
			Artifact: "custom/build/*.jar",
			Build:    "./custom build",
			Enabled:  true,
		},
	}}); err != nil {
		t.Fatalf("Save(existing workspace) error = %v", err)
	}

	root := t.TempDir()
	repo := filepath.Join(root, "plugin-chat")
	if err := os.MkdirAll(filepath.Join(repo, "paper"), 0o755); err != nil {
		t.Fatalf("MkdirAll(repo) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "settings.gradle.kts"), []byte(`rootProject.name = "plugin-chat"`), 0o644); err != nil {
		t.Fatalf("WriteFile(settings.gradle.kts) error = %v", err)
	}

	cmd := NewWorkspaceCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"scan", "--yes", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := internalworkspace.Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := cfg.Repos["plugin-chat"]
	if got.Path != existingPath || got.Artifact != "custom/build/*.jar" || got.Build != "./custom build" || !got.Enabled {
		t.Fatalf("existing mapping was overwritten: %#v", got)
	}
	if !strings.Contains(out.String(), "skipped existing=1") {
		t.Fatalf("output = %q, want skipped existing count", out.String())
	}
}
