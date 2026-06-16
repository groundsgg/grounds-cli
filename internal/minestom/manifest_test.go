package minestom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPushManifestSelectsMinestomFlavor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
    build:
      task: :examples:minigame-agones:distTar
      artifact: examples/minigame-agones/build/distributions/*.tar
    modules:
      - id: plugin-config
        variant: minestom
        source: github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar
      - id: plugin-agones
        variant: minestom
        source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
`)

	manifest, err := LoadPushManifest(path, "minestom")
	if err != nil {
		t.Fatalf("LoadPushManifest() error = %v", err)
	}

	if manifest.Name != "minigame-bedwars" {
		t.Fatalf("Name = %q", manifest.Name)
	}
	if manifest.FlavorKey != "minestom" {
		t.Fatalf("FlavorKey = %q", manifest.FlavorKey)
	}
	if manifest.Runtime.Type != "minestom-server" {
		t.Fatalf("Runtime.Type = %q", manifest.Runtime.Type)
	}
	if manifest.Runtime.PublicType != "minestom" {
		t.Fatalf("Runtime.PublicType = %q", manifest.Runtime.PublicType)
	}
	if manifest.Runtime.Build.Task != ":examples:minigame-agones:distTar" {
		t.Fatalf("Build.Task = %q", manifest.Runtime.Build.Task)
	}
	if len(manifest.Runtime.Modules) != 2 {
		t.Fatalf("Modules len = %d", len(manifest.Runtime.Modules))
	}
}

func TestLoadPushManifestNormalizesMinestomFlavorType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
flavors:
  minestom:
    type: minestom
    baseImage: minestom
    build:
      task: :examples:minigame-agones:distTar
      artifact: examples/minigame-agones/build/distributions/*.tar
    modules:
      - id: plugin-config
        variant: minestom
        source: github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar
`)

	manifest, err := LoadPushManifest(path, "minestom")
	if err != nil {
		t.Fatalf("LoadPushManifest() error = %v", err)
	}

	if manifest.Runtime.Type != "minestom-server" {
		t.Fatalf("Runtime.Type = %q", manifest.Runtime.Type)
	}
	if manifest.Runtime.PublicType != "minestom" {
		t.Fatalf("Runtime.PublicType = %q", manifest.Runtime.PublicType)
	}
}

func TestLoadPushManifestRejectsMissingMinestomBuild(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
`)

	_, err := LoadPushManifest(path, "minestom")
	if err == nil || !strings.Contains(err.Error(), "build.task") || !strings.Contains(err.Error(), "build.artifact") {
		t.Fatalf("LoadPushManifest() error = %v, want missing build fields", err)
	}
}

func TestLoadPushManifestParsesTopLevelMinestomRuntime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
type: minestom
baseImage: minestom
build:
  task: :examples:minigame-agones:distTar
  artifact: examples/minigame-agones/build/distributions/*.tar
modules:
  - id: plugin-config
    variant: minestom
    source: github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar
`)

	manifest, err := LoadPushManifest(path, "")
	if err != nil {
		t.Fatalf("LoadPushManifest() error = %v", err)
	}

	if manifest.Name != "minigame-bedwars" {
		t.Fatalf("Name = %q", manifest.Name)
	}
	if manifest.FlavorKey != "minestom" {
		t.Fatalf("FlavorKey = %q", manifest.FlavorKey)
	}
	if manifest.Runtime.Type != "minestom-server" {
		t.Fatalf("Runtime.Type = %q", manifest.Runtime.Type)
	}
	if manifest.Runtime.PublicType != "minestom" {
		t.Fatalf("Runtime.PublicType = %q", manifest.Runtime.PublicType)
	}
	if manifest.Runtime.Build.Artifact != "examples/minigame-agones/build/distributions/*.tar" {
		t.Fatalf("Build.Artifact = %q", manifest.Runtime.Build.Artifact)
	}
	if len(manifest.Runtime.Modules) != 1 {
		t.Fatalf("Modules len = %d", len(manifest.Runtime.Modules))
	}
}

func TestLoadPushManifestRejectsMissingTopLevelMinestomBuildAsRuntime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
type: minestom
baseImage: minestom
`)

	_, err := LoadPushManifest(path, "")
	if err == nil || !strings.Contains(err.Error(), "minestom runtime") || strings.Contains(err.Error(), "flavor") {
		t.Fatalf("LoadPushManifest() error = %v, want top-level runtime missing build fields", err)
	}
}

func TestLoadPushManifestReturnsNonMinestomForPaperFlavor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: plugin-config
flavors:
  paper:
    type: paper
    baseImage: paper
`)

	manifest, err := LoadPushManifest(path, "paper")
	if err != nil {
		t.Fatalf("LoadPushManifest() error = %v", err)
	}
	if manifest.IsMinestomServer() {
		t.Fatalf("paper flavor should not be a Minestom server")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
