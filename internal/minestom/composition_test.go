package minestom

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
)

func TestResolveLocalModulesSelectsEnabledMinestomEntries(t *testing.T) {
	manifest := PushManifest{Runtime: Runtime{Modules: []Module{
		{
			ID:      "plugin-agones",
			Variant: "minestom",
			Source:  "github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar",
		},
		{
			ID:      "plugin-config",
			Variant: "minestom",
			Source:  "github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar",
		},
	}}}
	cfg := &internalworkspace.Config{Repos: map[string]internalworkspace.Repo{
		"plugin-agones": {
			Path: t.TempDir(),
			Variants: map[string]internalworkspace.Variant{
				"minestom": {
					Enabled: true,
					Module:  "gg.grounds:plugin-agones-minestom",
					Project: ":minestom",
				},
			},
		},
		"plugin-config": {
			Path: t.TempDir(),
			Variants: map[string]internalworkspace.Variant{
				"minestom": {Enabled: false},
			},
		},
	}}

	plan, err := ResolveLocalModules(context.Background(), manifest, cfg, ResolveOptions{WithLocal: true})
	if err != nil {
		t.Fatalf("ResolveLocalModules() error = %v", err)
	}
	if len(plan.LocalModules) != 1 {
		t.Fatalf("LocalModules len = %d, want 1", len(plan.LocalModules))
	}
	if plan.LocalModules[0].ID != "plugin-agones" {
		t.Fatalf("LocalModules[0].ID = %q", plan.LocalModules[0].ID)
	}
	if len(plan.EffectivePluginSources) != 2 {
		t.Fatalf("EffectivePluginSources len = %d, want 2", len(plan.EffectivePluginSources))
	}
	if plan.EffectivePluginSources[0].Effective != "local" {
		t.Fatalf("EffectivePluginSources[0].Effective = %q", plan.EffectivePluginSources[0].Effective)
	}
	if plan.EffectivePluginSources[1].Effective != "release" {
		t.Fatalf("EffectivePluginSources[1].Effective = %q", plan.EffectivePluginSources[1].Effective)
	}
}

func TestResolveLocalModulesHonorsExplicitLocalIDs(t *testing.T) {
	manifest := PushManifest{Runtime: Runtime{Modules: []Module{
		{
			ID:      "plugin-agones",
			Variant: "minestom",
			Source:  "github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar",
		},
	}}}
	cfg := &internalworkspace.Config{Repos: map[string]internalworkspace.Repo{
		"plugin-agones": {
			Path: t.TempDir(),
			Variants: map[string]internalworkspace.Variant{
				"minestom": {Enabled: false},
			},
		},
	}}

	plan, err := ResolveLocalModules(context.Background(), manifest, cfg, ResolveOptions{LocalIDs: []string{"plugin-agones"}})
	if err != nil {
		t.Fatalf("ResolveLocalModules() error = %v", err)
	}
	if len(plan.LocalModules) != 1 {
		t.Fatalf("LocalModules len = %d, want 1", len(plan.LocalModules))
	}
	if plan.LocalModules[0].ID != "plugin-agones" {
		t.Fatalf("LocalModules[0].ID = %q", plan.LocalModules[0].ID)
	}
}

func TestWriteCompositeInitScriptIsDeterministicAndUnique(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	plan := &LocalPlan{LocalModules: []LocalModule{
		{ID: "b", Path: b},
		{ID: "a", Path: a},
		{ID: "duplicate-a", Path: a},
	}}

	path, err := WriteCompositeInitScript(plan)
	if err != nil {
		t.Fatalf("WriteCompositeInitScript() error = %v", err)
	}
	defer os.Remove(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(init script) error = %v", err)
	}
	expected := strings.Join([]string{
		"settingsEvaluated {",
		"\tincludeBuild(\"" + filepath.ToSlash(a) + "\")",
		"\tincludeBuild(\"" + filepath.ToSlash(b) + "\")",
		"}",
		"",
	}, "\n")
	if string(raw) != expected {
		t.Fatalf("init script =\n%s\nwant:\n%s", raw, expected)
	}
}

func TestResolveDistributionArtifactPicksNewestTar(t *testing.T) {
	root := t.TempDir()
	distributions := filepath.Join(root, "build", "distributions")
	if err := os.MkdirAll(distributions, 0o755); err != nil {
		t.Fatalf("MkdirAll(distributions) error = %v", err)
	}
	oldPath := filepath.Join(distributions, "minigame-old.tar")
	newPath := filepath.Join(distributions, "minigame-new.tar")
	writeFile(t, oldPath, "old")
	writeFile(t, newPath, "new")
	baseTime := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldPath, baseTime, baseTime); err != nil {
		t.Fatalf("Chtimes(oldPath) error = %v", err)
	}
	if err := os.Chtimes(newPath, baseTime.Add(time.Hour), baseTime.Add(time.Hour)); err != nil {
		t.Fatalf("Chtimes(newPath) error = %v", err)
	}

	got, err := ResolveDistributionArtifact(root, "build/distributions/*.tar")
	if err != nil {
		t.Fatalf("ResolveDistributionArtifact() error = %v", err)
	}
	if got != newPath {
		t.Fatalf("ResolveDistributionArtifact() = %q, want %q", got, newPath)
	}
}

func TestResolveDistributionArtifactIgnoresNonRegularTarMatches(t *testing.T) {
	root := t.TempDir()
	distributions := filepath.Join(root, "build", "distributions")
	if err := os.MkdirAll(distributions, 0o755); err != nil {
		t.Fatalf("MkdirAll(distributions) error = %v", err)
	}
	filePath := filepath.Join(distributions, "minigame-file.tar")
	dirPath := filepath.Join(distributions, "minigame-dir.tar")
	writeFile(t, filePath, "file")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", dirPath, err)
	}
	baseTime := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filePath, baseTime, baseTime); err != nil {
		t.Fatalf("Chtimes(filePath) error = %v", err)
	}
	if err := os.Chtimes(dirPath, baseTime.Add(time.Hour), baseTime.Add(time.Hour)); err != nil {
		t.Fatalf("Chtimes(dirPath) error = %v", err)
	}

	got, err := ResolveDistributionArtifact(root, "build/distributions/*.tar")
	if err != nil {
		t.Fatalf("ResolveDistributionArtifact() error = %v", err)
	}
	if got != filePath {
		t.Fatalf("ResolveDistributionArtifact() = %q, want %q", got, filePath)
	}
}

func TestNormalizeDistributionArtifactRepackagesGradleTar(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "minigame-local-SNAPSHOT.tar")
	createGradleDistributionTar(t, sourcePath)

	normalizedPath, cleanup, err := NormalizeDistributionArtifact(sourcePath)
	if err != nil {
		t.Fatalf("NormalizeDistributionArtifact() error = %v", err)
	}
	defer cleanup()

	entries := readTarGzEntries(t, normalizedPath)
	want := []tarEntry{
		{Name: "app/", Mode: 0o755},
		{Name: "app/bin/", Mode: 0o755},
		{Name: "app/bin/app", Mode: 0o755},
		{Name: "app/lib/", Mode: 0o755},
		{Name: "app/lib/minigame.jar", Mode: 0o644},
	}
	if len(entries) != len(want) {
		t.Fatalf("entries = %#v, want %#v", entries, want)
	}
	for i := range want {
		if entries[i].Name != want[i].Name {
			t.Fatalf("entry[%d].Name = %q, want %q", i, entries[i].Name, want[i].Name)
		}
		if entries[i].Mode != want[i].Mode {
			t.Fatalf("entry[%d].Mode = %o, want %o", i, entries[i].Mode, want[i].Mode)
		}
	}
}

func TestNormalizeDistributionArtifactRejectsUnsafeTraversalPath(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "unsafe.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/../evil", Typeflag: tar.TypeReg, Mode: 0o644, Body: "evil"},
	})

	assertNormalizeDistributionArtifactErrorContains(t, sourcePath, "unsafe archive path")
}

func TestNormalizeDistributionArtifactRejectsAbsolutePath(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "absolute.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "/minigame-local-SNAPSHOT/bin/minigame", Typeflag: tar.TypeReg, Mode: 0o755, Body: "launcher"},
	})

	assertNormalizeDistributionArtifactErrorContains(t, sourcePath, "unsafe archive path")
}

func TestNormalizeDistributionArtifactRejectsMultipleRoots(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "multiple-roots.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "other-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
	})

	assertNormalizeDistributionArtifactErrorContains(t, sourcePath, "multiple roots")
}

func TestNormalizeDistributionArtifactRejectsMissingLauncher(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "missing-launcher.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/app/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/minigame.bat", Typeflag: tar.TypeReg, Mode: 0o644, Body: "@echo off\r\n"},
		{Name: "minigame-local-SNAPSHOT/lib/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/lib/minigame.jar", Typeflag: tar.TypeReg, Mode: 0o644, Body: "jar"},
	})

	assertNormalizeDistributionArtifactErrorContains(t, sourcePath, "missing unix launcher")
}

func TestNormalizeDistributionArtifactRejectsUnsupportedLinks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typeflag byte
	}{
		{name: "symlink", typeflag: tar.TypeSymlink},
		{name: "hardlink", typeflag: tar.TypeLink},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourcePath := filepath.Join(t.TempDir(), tc.name+".tar")
			createTarArchive(t, sourcePath, []archiveEntry{
				{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
				{Name: "minigame-local-SNAPSHOT/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
				{Name: "minigame-local-SNAPSHOT/bin/minigame", Typeflag: tar.TypeReg, Mode: 0o755, Body: "launcher"},
				{Name: "minigame-local-SNAPSHOT/lib/", Typeflag: tar.TypeDir, Mode: 0o755},
				{Name: "minigame-local-SNAPSHOT/lib/linked.jar", Typeflag: tc.typeflag, Mode: 0o644, Linkname: "minigame.jar"},
			})

			assertNormalizeDistributionArtifactErrorContains(t, sourcePath, "unsupported distribution archive entry")
		})
	}
}

func TestNormalizeDistributionArtifactRejectsAmbiguousUnixLauncherCandidates(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "ambiguous-launchers.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/helper", Typeflag: tar.TypeReg, Mode: 0o755, Body: "helper"},
		{Name: "minigame-local-SNAPSHOT/bin/other", Typeflag: tar.TypeReg, Mode: 0o755, Body: "other"},
		{Name: "minigame-local-SNAPSHOT/lib/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/lib/minigame.jar", Typeflag: tar.TypeReg, Mode: 0o644, Body: "jar"},
	})

	assertNormalizeDistributionArtifactErrorContains(t, sourcePath, "ambiguous unix launcher")
}

func TestNormalizeDistributionArtifactUsesRootDerivedLauncherWhenMultipleBinFilesExist(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "root-derived-launcher.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/minigame", Typeflag: tar.TypeReg, Mode: 0o755, Body: "intended launcher\n"},
		{Name: "minigame-local-SNAPSHOT/bin/helper", Typeflag: tar.TypeReg, Mode: 0o755, Body: "helper launcher\n"},
		{Name: "minigame-local-SNAPSHOT/lib/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/lib/minigame.jar", Typeflag: tar.TypeReg, Mode: 0o644, Body: "jar"},
	})

	normalizedPath, cleanup, err := NormalizeDistributionArtifact(sourcePath)
	if err != nil {
		t.Fatalf("NormalizeDistributionArtifact() error = %v", err)
	}
	defer cleanup()

	if got := readTarGzFile(t, normalizedPath, "app/bin/app"); got != "intended launcher\n" {
		t.Fatalf("app/bin/app content = %q, want intended launcher", got)
	}
}

func TestNormalizeDistributionArtifactUsesVersionedSnapshotRootDerivedLauncher(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "versioned-snapshot-launcher.tar")
	createTarArchive(t, sourcePath, []archiveEntry{
		{Name: "minigame-1.2.3-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-1.2.3-SNAPSHOT/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-1.2.3-SNAPSHOT/bin/minigame", Typeflag: tar.TypeReg, Mode: 0o755, Body: "versioned launcher\n"},
		{Name: "minigame-1.2.3-SNAPSHOT/bin/helper", Typeflag: tar.TypeReg, Mode: 0o755, Body: "helper launcher\n"},
		{Name: "minigame-1.2.3-SNAPSHOT/lib/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-1.2.3-SNAPSHOT/lib/minigame.jar", Typeflag: tar.TypeReg, Mode: 0o644, Body: "jar"},
	})

	normalizedPath, cleanup, err := NormalizeDistributionArtifact(sourcePath)
	if err != nil {
		t.Fatalf("NormalizeDistributionArtifact() error = %v", err)
	}
	defer cleanup()

	if got := readTarGzFile(t, normalizedPath, "app/bin/app"); got != "versioned launcher\n" {
		t.Fatalf("app/bin/app content = %q, want versioned launcher", got)
	}
}

type tarEntry struct {
	Name string
	Mode int64
}

type archiveEntry struct {
	Name     string
	Typeflag byte
	Mode     int64
	Body     string
	Linkname string
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func createGradleDistributionTar(t *testing.T, path string) {
	t.Helper()
	createTarArchive(t, path, []archiveEntry{
		{Name: "minigame-local-SNAPSHOT/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/bin/minigame", Typeflag: tar.TypeReg, Mode: 0o755, Body: "#!/bin/sh\n"},
		{Name: "minigame-local-SNAPSHOT/bin/minigame.bat", Typeflag: tar.TypeReg, Mode: 0o644, Body: "@echo off\r\n"},
		{Name: "minigame-local-SNAPSHOT/lib/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "minigame-local-SNAPSHOT/lib/minigame.jar", Typeflag: tar.TypeReg, Mode: 0o644, Body: "jar"},
	})
}

func createTarArchive(t *testing.T, path string, entries []archiveEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer file.Close()
	tw := tar.NewWriter(file)
	defer tw.Close()

	for _, entry := range entries {
		size := int64(0)
		if entry.Typeflag == tar.TypeReg || entry.Typeflag == tar.TypeRegA {
			size = int64(len(entry.Body))
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:     entry.Name,
			Typeflag: entry.Typeflag,
			Mode:     entry.Mode,
			Size:     size,
			Linkname: entry.Linkname,
		}); err != nil {
			t.Fatalf("WriteHeader(%q) error = %v", entry.Name, err)
		}
		if size > 0 {
			if _, err := tw.Write([]byte(entry.Body)); err != nil {
				t.Fatalf("Write(%q) error = %v", entry.Name, err)
			}
		}
	}
}

func readTarGzEntries(t *testing.T, path string) []tarEntry {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", path, err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader(%q) error = %v", path, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	var entries []tarEntry
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		entries = append(entries, tarEntry{Name: header.Name, Mode: header.Mode})
	}
	return entries
}

func readTarGzFile(t *testing.T, path, name string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", path, err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader(%q) error = %v", path, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			t.Fatalf("%q not found in %q", name, path)
		}
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		if header.Name != name {
			continue
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("ReadAll(%q) error = %v", name, err)
		}
		return string(raw)
	}
}

func assertNormalizeDistributionArtifactErrorContains(t *testing.T, sourcePath, want string) {
	t.Helper()
	normalizedPath, cleanup, err := NormalizeDistributionArtifact(sourcePath)
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Fatalf("NormalizeDistributionArtifact() = %q, nil; want error containing %q", normalizedPath, want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("NormalizeDistributionArtifact() error = %v, want containing %q", err, want)
	}
}
