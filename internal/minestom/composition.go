package minestom

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
)

type ResolveOptions struct {
	LocalIDs  []string
	WithLocal bool
}

type LocalPlan struct {
	LocalModules           []LocalModule
	EffectivePluginSources []internalworkspace.EffectiveSource
}

type LocalModule struct {
	ID      string
	Variant string
	Path    string
	Module  string
	Project string
}

type compositeSubstitution struct {
	module  string
	project string
}

type launcherCandidate struct {
	name string
	path string
}

func ResolveLocalModules(ctx context.Context, manifest PushManifest, cfg *internalworkspace.Config, opts ResolveOptions) (*LocalPlan, error) {
	localIDs := internalworkspace.NormalizeLocalIDs(opts.LocalIDs)
	explicitLocal := map[string]bool{}
	for _, id := range localIDs {
		explicitLocal[id] = true
		if !manifestContainsModule(manifest.Runtime.Modules, id) {
			return nil, fmt.Errorf("--local module %q not found in grounds.yaml", id)
		}
	}

	plan := &LocalPlan{}
	for _, module := range manifest.Runtime.Modules {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entry, ok := cfg.EntryForVariant(module.ID, module.Variant)
		selected := explicitLocal[module.ID] || opts.WithLocal && ok && entry.Enabled
		if selected {
			if !ok {
				return nil, fmt.Errorf("local workspace entry for %q variant %q not found", module.ID, module.Variant)
			}
			if entry.Path == "" {
				return nil, fmt.Errorf("local workspace entry for %q has no path", module.ID)
			}
			plan.LocalModules = append(plan.LocalModules, LocalModule{
				ID:      module.ID,
				Variant: module.Variant,
				Path:    entry.Path,
				Module:  entry.Module,
				Project: entry.Project,
			})
			plan.EffectivePluginSources = append(plan.EffectivePluginSources, internalworkspace.EffectiveSource{
				ID:            module.ID,
				Variant:       module.Variant,
				Effective:     "local",
				DefaultSource: module.Source,
			})
			continue
		}
		plan.EffectivePluginSources = append(plan.EffectivePluginSources, internalworkspace.EffectiveSource{
			ID:        module.ID,
			Variant:   module.Variant,
			Effective: "release",
			Source:    module.Source,
		})
	}
	return plan, nil
}

func WriteCompositeInitScript(plan *LocalPlan) (string, error) {
	includes := map[string][]compositeSubstitution{}
	if plan != nil {
		for _, module := range plan.LocalModules {
			if module.Path == "" {
				return "", fmt.Errorf("local module %q has no path", module.ID)
			}
			if (module.Module == "") != (module.Project == "") {
				return "", fmt.Errorf("local module %q requires both module and project for dependency substitution", module.ID)
			}
			absolute, err := filepath.Abs(module.Path)
			if err != nil {
				return "", err
			}
			if _, ok := includes[absolute]; !ok {
				includes[absolute] = nil
			}
			if module.Module != "" {
				includes[absolute] = append(includes[absolute], compositeSubstitution{
					module:  module.Module,
					project: module.Project,
				})
			}
		}
	}

	sorted := make([]string, 0, len(includes))
	for includePath := range includes {
		sorted = append(sorted, includePath)
	}
	sort.Strings(sorted)

	var builder strings.Builder
	builder.WriteString("settingsEvaluated {\n")
	for _, includePath := range sorted {
		builder.WriteString("\tincludeBuild(\"")
		builder.WriteString(escapeKotlinString(filepath.ToSlash(includePath)))
		substitutions := uniqueSubstitutions(includes[includePath])
		if len(substitutions) == 0 {
			builder.WriteString("\")\n")
			continue
		}
		builder.WriteString("\") {\n")
		builder.WriteString("\t\tdependencySubstitution {\n")
		for _, substitution := range substitutions {
			builder.WriteString("\t\t\tsubstitute(module(\"")
			builder.WriteString(escapeKotlinString(substitution.module))
			builder.WriteString("\")).using(project(\"")
			builder.WriteString(escapeKotlinString(substitution.project))
			builder.WriteString("\"))\n")
		}
		builder.WriteString("\t\t}\n")
		builder.WriteString("\t}\n")
	}
	builder.WriteString("}\n")

	file, err := os.CreateTemp("", "grounds-minestom-composite-*.gradle.kts")
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(builder.String()); err != nil {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
		return "", err
	}
	if err := file.Close(); err != nil {
		name := file.Name()
		_ = os.Remove(name)
		return "", err
	}
	return file.Name(), nil
}

func uniqueSubstitutions(values []compositeSubstitution) []compositeSubstitution {
	seen := map[string]bool{}
	unique := make([]compositeSubstitution, 0, len(values))
	for _, value := range values {
		key := value.module + "\x00" + value.project
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, value)
	}
	sort.SliceStable(unique, func(i, j int) bool {
		if unique[i].module == unique[j].module {
			return unique[i].project < unique[j].project
		}
		return unique[i].module < unique[j].module
	})
	return unique
}

func ResolveDistributionArtifact(projectRoot, pattern string) (string, error) {
	if strings.TrimSpace(pattern) == "" {
		return "", fmt.Errorf("distribution artifact glob is empty")
	}
	glob := filepath.FromSlash(pattern)
	if !filepath.IsAbs(glob) {
		glob = filepath.Join(projectRoot, glob)
	}
	matches, err := filepath.Glob(glob)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if isTarArtifact(match) {
			candidates = append(candidates, match)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("expected at least one distribution tar for %s, found 0", glob)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		leftInfo, leftErr := os.Stat(left)
		rightInfo, rightErr := os.Stat(right)
		if leftErr == nil && rightErr == nil && !leftInfo.ModTime().Equal(rightInfo.ModTime()) {
			return leftInfo.ModTime().After(rightInfo.ModTime())
		}
		return left < right
	})
	return candidates[0], nil
}

func NormalizeDistributionArtifact(sourcePath string) (string, func(), error) {
	if !isTarArtifact(sourcePath) {
		return "", nil, fmt.Errorf("unsupported distribution artifact %q", sourcePath)
	}
	tempDir, err := os.MkdirTemp("", "grounds-minestom-distribution-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	stageRoot := filepath.Join(tempDir, "app")
	if err := extractGradleDistribution(sourcePath, stageRoot); err != nil {
		cleanup()
		return "", nil, err
	}

	normalizedPath := filepath.Join(tempDir, "app.tar.gz")
	if err := writeNormalizedTarGz(stageRoot, normalizedPath); err != nil {
		cleanup()
		return "", nil, err
	}
	return normalizedPath, cleanup, nil
}

func ArtifactSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func manifestContainsModule(modules []Module, id string) bool {
	for _, module := range modules {
		if module.ID == id {
			return true
		}
	}
	return false
}

func escapeKotlinString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`)
	return replacer.Replace(value)
}

func isTarArtifact(filePath string) bool {
	name := strings.ToLower(filePath)
	return strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")
}

func extractGradleDistribution(sourcePath, stageRoot string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	var reader io.Reader = source
	var gz *gzip.Reader
	if isGzipTar(sourcePath) {
		gz, err = gzip.NewReader(source)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	}

	if err := ensureDir(stageRoot, 0o755); err != nil {
		return err
	}

	tr := tar.NewReader(reader)
	var archiveRoot string
	var launcherCandidates []launcherCandidate
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cleanName, err := cleanArchivePath(header.Name)
		if err != nil {
			return err
		}
		root, relativePath := splitArchiveRoot(cleanName)
		if archiveRoot == "" {
			archiveRoot = root
		} else if root != archiveRoot {
			return fmt.Errorf("distribution archive contains multiple roots: %q and %q", archiveRoot, root)
		}
		if relativePath == "" {
			if header.Typeflag != tar.TypeDir {
				return fmt.Errorf("distribution archive root %q is not a directory", root)
			}
			continue
		}

		outputPath := relativePath
		target, err := safeJoin(stageRoot, outputPath)
		if err != nil {
			return err
		}
		directBinName, isDirectBin := directBinFileName(outputPath)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := ensureDir(target, modeOrDefault(header.Mode, 0o755)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if isDirectBin && strings.EqualFold(path.Ext(directBinName), ".bat") {
				continue
			}
			if err := ensureDir(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := modeOrDefault(header.Mode, 0o644)
			if err := writeExtractedFile(target, tr, mode); err != nil {
				return err
			}
			if isDirectBin && isExecutableMode(mode) {
				launcherCandidates = append(launcherCandidates, launcherCandidate{
					name: directBinName,
					path: target,
				})
			}
		default:
			return fmt.Errorf("unsupported distribution archive entry %q", header.Name)
		}
	}
	if archiveRoot == "" {
		return fmt.Errorf("distribution archive is empty")
	}
	return finalizeLauncher(stageRoot, archiveRoot, launcherCandidates)
}

func isGzipTar(filePath string) bool {
	name := strings.ToLower(filePath)
	return strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")
}

func cleanArchivePath(name string) (string, error) {
	if strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("unsafe archive path %q", name)
		}
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return cleaned, nil
}

func splitArchiveRoot(cleanName string) (string, string) {
	root, rest, ok := strings.Cut(cleanName, "/")
	if !ok {
		return cleanName, ""
	}
	return root, rest
}

func safeJoin(base, relativePath string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(relativePath))
	if cleaned == "." || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("unsafe archive path %q", relativePath)
	}
	target := filepath.Join(base, cleaned)
	relative, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe archive path %q", relativePath)
	}
	return target, nil
}

func modeOrDefault(mode, fallback int64) os.FileMode {
	permissions := os.FileMode(mode).Perm()
	if permissions == 0 {
		return os.FileMode(fallback)
	}
	return permissions
}

func ensureDir(dir string, mode os.FileMode) error {
	if err := os.MkdirAll(dir, mode); err != nil {
		return err
	}
	return os.Chmod(dir, mode)
}

func writeExtractedFile(target string, reader io.Reader, mode os.FileMode) error {
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(target, mode)
}

func directBinFileName(relativePath string) (string, bool) {
	dir, name := path.Split(relativePath)
	if dir != "bin/" || name == "" {
		return "", false
	}
	return name, true
}

func isExecutableMode(mode os.FileMode) bool {
	return mode.Perm()&0o111 != 0
}

func finalizeLauncher(stageRoot, archiveRoot string, candidates []launcherCandidate) error {
	selected, err := selectLauncherCandidate(archiveRoot, candidates)
	if err != nil {
		return err
	}

	finalPath := filepath.Join(stageRoot, "bin", "app")
	for _, candidate := range candidates {
		if candidate.path == selected.path {
			continue
		}
		if err := os.Remove(candidate.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	if selected.path != finalPath {
		if _, err := os.Stat(finalPath); err == nil {
			return fmt.Errorf("launcher output path %q conflicts with archive entry", "bin/app")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := ensureDir(filepath.Dir(finalPath), 0o755); err != nil {
			return err
		}
		if err := os.Rename(selected.path, finalPath); err != nil {
			return err
		}
	}
	return os.Chmod(finalPath, 0o755)
}

func selectLauncherCandidate(archiveRoot string, candidates []launcherCandidate) (launcherCandidate, error) {
	if len(candidates) == 0 {
		return launcherCandidate{}, fmt.Errorf("distribution archive missing unix launcher under bin/")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	expected := rootDerivedLauncherName(archiveRoot)
	var matches []launcherCandidate
	for _, candidate := range candidates {
		if candidate.name == expected {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.name)
	}
	sort.Strings(names)
	return launcherCandidate{}, fmt.Errorf("ambiguous unix launcher candidates under bin/ for %q: %s", archiveRoot, strings.Join(names, ", "))
}

func rootDerivedLauncherName(archiveRoot string) string {
	root := path.Base(archiveRoot)
	for _, suffix := range []string{"-local-SNAPSHOT", "-SNAPSHOT"} {
		if stripped, ok := trimSuffixFold(root, suffix); ok {
			root = stripped
			break
		}
	}
	if stripped, ok := stripNumericVersionSuffix(root); ok {
		return stripped
	}
	return root
}

func trimSuffixFold(value, suffix string) (string, bool) {
	if len(value) <= len(suffix) || !strings.HasSuffix(strings.ToLower(value), strings.ToLower(suffix)) {
		return "", false
	}
	return value[:len(value)-len(suffix)], true
}

func stripNumericVersionSuffix(value string) (string, bool) {
	index := strings.LastIndex(value, "-")
	if index <= 0 || index == len(value)-1 {
		return "", false
	}
	suffix := value[index+1:]
	if suffix[0] < '0' || suffix[0] > '9' {
		return "", false
	}
	for _, r := range suffix {
		if r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			continue
		}
		return "", false
	}
	return value[:index], true
}

func writeNormalizedTarGz(stageRoot, outputPath string) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(output)
	gz.Name = "app.tar"
	tw := tar.NewWriter(gz)

	walkErr := filepath.WalkDir(stageRoot, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(filepath.Dir(stageRoot), filePath)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		if entry.IsDir() {
			header.Name += "/"
		}
		header.ModTime = time.Unix(0, 0)
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		input, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})

	closeTarErr := tw.Close()
	closeGzipErr := gz.Close()
	closeOutputErr := output.Close()
	if walkErr != nil {
		return walkErr
	}
	if closeTarErr != nil {
		return closeTarErr
	}
	if closeGzipErr != nil {
		return closeGzipErr
	}
	return closeOutputErr
}
