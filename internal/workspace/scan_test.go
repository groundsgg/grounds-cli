package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRootsFindsDirectChildReposWithVariants(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "plugin-chat")
	mkdirAll(t, filepath.Join(repo, "paper", "build", "libs"))
	mkdirAll(t, filepath.Join(repo, "velocity", "build", "libs"))
	writeFile(t, filepath.Join(repo, "settings.gradle.kts"), "")

	cfg, err := ScanRoots([]string{root})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}

	got, ok := cfg.Repos["plugin-chat"]
	if !ok {
		t.Fatalf("missing plugin-chat mapping: %v", cfg.Repos)
	}
	if got.Path != repo {
		t.Fatalf("Path = %q, want %q", got.Path, repo)
	}
	paper := got.Variants["paper"]
	if paper.Artifact != "paper/build/libs/*.jar" {
		t.Fatalf("paper artifact = %q", paper.Artifact)
	}
	if paper.Build != "./gradlew :paper:shadowJar" {
		t.Fatalf("paper build = %q", paper.Build)
	}
	if !paper.Enabled {
		t.Fatal("paper enabled = false, want true")
	}
	if paper.Module != "" {
		t.Fatalf("paper module = %q, want empty until configured explicitly", paper.Module)
	}
	if paper.Project != "" {
		t.Fatalf("paper project = %q, want empty until configured explicitly", paper.Project)
	}
	if _, ok := got.Variants["velocity"]; !ok {
		t.Fatalf("missing velocity variant: %v", got.Variants)
	}
}

func TestScanRootsUsesRootArtifactWhenNoVariantsExist(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "plugin-permissions")
	mkdirAll(t, filepath.Join(repo, "build", "libs"))
	writeFile(t, filepath.Join(repo, "build.gradle.kts"), "")

	cfg, err := ScanRoots([]string{root})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}

	got := cfg.Repos["plugin-permissions"]
	if got.Artifact != "build/libs/*.jar" {
		t.Fatalf("Artifact = %q", got.Artifact)
	}
	if got.Build != "./gradlew build" {
		t.Fatalf("Build = %q", got.Build)
	}
	if !got.Enabled {
		t.Fatal("Enabled = false, want true")
	}
}

func TestScanRootsDoesNotRecurseBelowChildRepos(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "parent", "plugin-nested")
	mkdirAll(t, nested)
	writeFile(t, filepath.Join(nested, "grounds.yaml"), "")

	cfg, err := ScanRoots([]string{root})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}
	if _, ok := cfg.Repos["plugin-nested"]; ok {
		t.Fatalf("unexpected nested repo mapping: %v", cfg.Repos)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
