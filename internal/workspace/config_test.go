package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadMissingWorkspaceConfigReturnsEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Repos) != 0 {
		t.Fatalf("Repos = %v, want empty", cfg.Repos)
	}
}

func TestSaveCreatesPrivateWorkspaceConfig(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "grounds")
	path := filepath.Join(dir, "workspace.yaml")
	cfg := &Config{
		Repos: map[string]Repo{
			"plugin-chat": {
				Path:     "/repos/plugin-chat",
				Artifact: "build/libs/*.jar",
				Build:    "./gradlew build",
				Enabled:  true,
			},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(workspace.yaml) error = %v", err)
		}
		return
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(config dir) error = %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("config dir mode = %o, want 700", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(workspace.yaml) error = %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("workspace.yaml mode = %o, want 600", got)
	}
}

func TestSaveTightensExistingWorkspaceConfigPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not enforced consistently on Windows")
	}

	dir := filepath.Join(t.TempDir(), "grounds")
	path := filepath.Join(dir, "workspace.yaml")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}
	if err := os.WriteFile(path, []byte("repos: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(workspace.yaml) error = %v", err)
	}

	if err := Save(path, &Config{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(config dir) error = %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("config dir mode = %o, want 700", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(workspace.yaml) error = %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("workspace.yaml mode = %o, want 600", got)
	}
}

func TestEntryForVariantUsesVariantSpecificArtifact(t *testing.T) {
	cfg := &Config{Repos: map[string]Repo{
		"plugin-chat": {
			Path:    "/repos/plugin-chat",
			Enabled: true,
			Variants: map[string]Variant{
				"paper": {
					Artifact: "paper/build/libs/*.jar",
					Build:    "./gradlew :paper:shadowJar",
					Enabled:  true,
				},
			},
		},
	}}

	entry, ok := cfg.EntryForVariant("plugin-chat", "paper")
	if !ok {
		t.Fatal("EntryForVariant() ok = false, want true")
	}
	if entry.Path != "/repos/plugin-chat" {
		t.Fatalf("Path = %q", entry.Path)
	}
	if entry.Artifact != "paper/build/libs/*.jar" {
		t.Fatalf("Artifact = %q", entry.Artifact)
	}
	if entry.Build != "./gradlew :paper:shadowJar" {
		t.Fatalf("Build = %q", entry.Build)
	}
	if !entry.Enabled {
		t.Fatal("Enabled = false, want true")
	}
}

func TestEntryForVariantUsesRootArtifactWhenNoVariantRequested(t *testing.T) {
	cfg := &Config{Repos: map[string]Repo{
		"plugin-permissions": {
			Path:     "/repos/plugin-permissions",
			Artifact: "build/libs/*.jar",
			Build:    "./gradlew build",
			Enabled:  true,
		},
	}}

	entry, ok := cfg.EntryForVariant("plugin-permissions", "")
	if !ok {
		t.Fatal("EntryForVariant() ok = false, want true")
	}
	if entry.Artifact != "build/libs/*.jar" {
		t.Fatalf("Artifact = %q", entry.Artifact)
	}
}

func TestEntryForVariantRequiresRequestedVariant(t *testing.T) {
	cfg := &Config{Repos: map[string]Repo{
		"plugin-permissions": {
			Path:     "/repos/plugin-permissions",
			Artifact: "build/libs/*.jar",
			Build:    "./gradlew build",
			Enabled:  true,
		},
	}}

	if entry, ok := cfg.EntryForVariant("plugin-permissions", "paper"); ok {
		t.Fatalf("EntryForVariant() = %#v, true; want missing variant", entry)
	}
}
