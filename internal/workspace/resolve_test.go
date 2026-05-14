package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeLocalIDsTrimsSplitsAndDedupes(t *testing.T) {
	got := NormalizeLocalIDs([]string{" plugin-chat,plugin-permissions ", "plugin-chat", "", "plugin-economy"})
	want := []string{"plugin-chat", "plugin-permissions", "plugin-economy"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("NormalizeLocalIDs() = %v, want %v", got, want)
	}
}

func TestResolveExplicitLocalBuildsArtifactAndKeepsReleaseMetadata(t *testing.T) {
	app := t.TempDir()
	writeFile(t, filepath.Join(app, "grounds.yaml"), `
plugins:
  - github:groundsgg/plugin-chat@v1.2.3:plugin-chat.jar
  - id: plugin-permissions
    source: github:groundsgg/plugin-permissions@v1.4.0:plugin-permissions.jar
`)
	repo := t.TempDir()
	mkdirAll(t, filepath.Join(repo, "paper", "build", "libs"))
	writeFile(t, filepath.Join(repo, "build.sh"), "#!/bin/sh\nprintf jar > paper/build/libs/plugin-chat.jar\n")
	if err := os.Chmod(filepath.Join(repo, "build.sh"), 0o700); err != nil {
		t.Fatalf("Chmod(build.sh) error = %v", err)
	}
	initGitRepo(t, repo)

	cfg := &Config{Repos: map[string]Repo{
		"plugin-chat": {
			Path: repo,
			Variants: map[string]Variant{
				"paper": {
					Artifact: "paper/build/libs/*.jar",
					Build:    "./build.sh",
					Enabled:  false,
				},
			},
		},
	}}

	plan, err := Resolve(context.Background(), filepath.Join(app, "grounds.yaml"), cfg, ResolveOptions{
		LocalIDs: []string{"plugin-chat"},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(plan.Plugins) != 2 {
		t.Fatalf("Plugins len = %d, want 2", len(plan.Plugins))
	}
	if plan.Plugins[0].LocalPath != filepath.Join(repo, "paper", "build", "libs", "plugin-chat.jar") {
		t.Fatalf("localPath = %q", plan.Plugins[0].LocalPath)
	}
	if plan.Plugins[1].Source != "github:groundsgg/plugin-permissions@v1.4.0:plugin-permissions.jar" {
		t.Fatalf("release source = %q", plan.Plugins[1].Source)
	}
	local := plan.EffectivePluginSources[0]
	if local.Effective != "local" {
		t.Fatalf("effective = %q, want local", local.Effective)
	}
	if local.Variant != "paper" {
		t.Fatalf("variant = %q, want paper", local.Variant)
	}
	if local.DefaultSource != "github:groundsgg/plugin-chat@v1.2.3:plugin-chat.jar" {
		t.Fatalf("defaultSource = %q", local.DefaultSource)
	}
	if local.ArtifactName != "plugin-chat.jar" {
		t.Fatalf("artifactName = %q", local.ArtifactName)
	}
	if len(local.ArtifactSha256) != 64 {
		t.Fatalf("artifactSha256 length = %d, want 64", len(local.ArtifactSha256))
	}
	if local.Git == nil || local.Git.Commit == "" || local.Git.Remote == "" {
		t.Fatalf("git metadata = %#v, want remote and commit", local.Git)
	}
	release := plan.EffectivePluginSources[1]
	if release.Effective != "release" || release.Source == "" {
		t.Fatalf("release metadata = %#v", release)
	}
}

func TestResolveWithLocalSelectsOnlyEnabledWorkspaceEntries(t *testing.T) {
	app := t.TempDir()
	writeFile(t, filepath.Join(app, "grounds.yaml"), `
plugins:
  - id: plugin-chat
    variant: paper
    source: github:groundsgg/plugin-chat@v1.2.3:plugin-chat.jar
  - id: plugin-disabled
    variant: paper
    source: github:groundsgg/plugin-disabled@v1.0.0:plugin-disabled.jar
`)
	enabledRepo := localJarRepo(t, "plugin-chat.jar")
	disabledRepo := localJarRepo(t, "plugin-disabled.jar")

	cfg := &Config{Repos: map[string]Repo{
		"plugin-chat": {
			Path: enabledRepo,
			Variants: map[string]Variant{
				"paper": {Artifact: "paper/build/libs/*.jar", Enabled: true},
			},
		},
		"plugin-disabled": {
			Path: disabledRepo,
			Variants: map[string]Variant{
				"paper": {Artifact: "paper/build/libs/*.jar", Enabled: false},
			},
		},
	}}

	plan, err := Resolve(context.Background(), filepath.Join(app, "grounds.yaml"), cfg, ResolveOptions{WithLocal: true})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Plugins[0].LocalPath == "" {
		t.Fatalf("plugin-chat was not overridden: %#v", plan.Plugins[0])
	}
	if plan.Plugins[1].Source == "" || plan.Plugins[1].LocalPath != "" {
		t.Fatalf("plugin-disabled should stay release: %#v", plan.Plugins[1])
	}
}

func TestResolveWithLocalSelectsEnabledSingleVariantForLegacyPluginString(t *testing.T) {
	app := t.TempDir()
	writeFile(t, filepath.Join(app, "grounds.yaml"), "plugins:\n  - github:groundsgg/plugin-chat@v1.2.3:plugin-chat.jar\n")
	repo := localJarRepo(t, "plugin-chat.jar")

	plan, err := Resolve(context.Background(), filepath.Join(app, "grounds.yaml"), &Config{Repos: map[string]Repo{
		"plugin-chat": {
			Path: repo,
			Variants: map[string]Variant{
				"paper": {Artifact: "paper/build/libs/*.jar", Enabled: true},
			},
		},
	}}, ResolveOptions{WithLocal: true})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Plugins[0].LocalPath == "" || plan.Plugins[0].Variant != "paper" {
		t.Fatalf("legacy string plugin was not resolved to local paper variant: %#v", plan.Plugins[0])
	}
}

func TestResolveRejectsUnknownExplicitLocalID(t *testing.T) {
	app := t.TempDir()
	writeFile(t, filepath.Join(app, "grounds.yaml"), "plugins:\n  - github:groundsgg/plugin-chat@v1.2.3:plugin-chat.jar\n")

	_, err := Resolve(context.Background(), filepath.Join(app, "grounds.yaml"), &Config{}, ResolveOptions{
		LocalIDs: []string{"plugin-missing"},
	})
	if err == nil || !strings.Contains(err.Error(), "not found in grounds.yaml") {
		t.Fatalf("Resolve() error = %v, want manifest missing ID error", err)
	}
}

func TestResolveRejectsAmbiguousLocalArtifactGlob(t *testing.T) {
	app := t.TempDir()
	writeFile(t, filepath.Join(app, "grounds.yaml"), "plugins:\n  - id: plugin-chat\n    variant: paper\n    source: github:groundsgg/plugin-chat@v1.2.3:plugin-chat.jar\n")
	repo := t.TempDir()
	mkdirAll(t, filepath.Join(repo, "paper", "build", "libs"))
	writeFile(t, filepath.Join(repo, "paper", "build", "libs", "one.jar"), "one")
	writeFile(t, filepath.Join(repo, "paper", "build", "libs", "two.jar"), "two")

	_, err := Resolve(context.Background(), filepath.Join(app, "grounds.yaml"), &Config{Repos: map[string]Repo{
		"plugin-chat": {
			Path: repo,
			Variants: map[string]Variant{
				"paper": {Artifact: "paper/build/libs/*.jar", Enabled: true},
			},
		},
	}}, ResolveOptions{LocalIDs: []string{"plugin-chat"}})
	if err == nil || !strings.Contains(err.Error(), "expected exactly one .jar") {
		t.Fatalf("Resolve() error = %v, want ambiguous artifact error", err)
	}
}

func localJarRepo(t *testing.T, jar string) string {
	t.Helper()
	repo := t.TempDir()
	mkdirAll(t, filepath.Join(repo, "paper", "build", "libs"))
	writeFile(t, filepath.Join(repo, "paper", "build", "libs", jar), "jar")
	initGitRepo(t, repo)
	return repo
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "remote", "add", "origin", "git@github.com:groundsgg/plugin-chat.git")
	writeFile(t, filepath.Join(dir, "README.md"), "test")
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "test")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
