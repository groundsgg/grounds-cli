package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ResolveOptions struct {
	LocalIDs  []string
	WithLocal bool
}

type Plan struct {
	Plugins                []PlanPlugin      `json:"plugins"`
	EffectivePluginSources []EffectiveSource `json:"effectivePluginSources"`
}

type PlanPlugin struct {
	ID        string `json:"id"`
	Variant   string `json:"variant,omitempty"`
	Source    string `json:"source,omitempty"`
	LocalPath string `json:"localPath,omitempty"`
}

type EffectiveSource struct {
	ID             string       `json:"id"`
	Variant        string       `json:"variant,omitempty"`
	Effective      string       `json:"effective"`
	Source         string       `json:"source,omitempty"`
	DefaultSource  string       `json:"defaultSource,omitempty"`
	ArtifactName   string       `json:"artifactName,omitempty"`
	ArtifactSha256 string       `json:"artifactSha256,omitempty"`
	Git            *GitMetadata `json:"git,omitempty"`
}

type GitMetadata struct {
	Remote string `json:"remote"`
	Commit string `json:"commit"`
	Dirty  bool   `json:"dirty"`
}

type manifestPlugin struct {
	ID      string
	Variant string
	Source  string
}

func NormalizeLocalIDs(values []string) []string {
	seen := map[string]bool{}
	var normalized []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			id := strings.TrimSpace(part)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			normalized = append(normalized, id)
		}
	}
	return normalized
}

func Resolve(ctx context.Context, manifestPath string, cfg *Config, opts ResolveOptions) (*Plan, error) {
	plugins, err := loadManifestPlugins(manifestPath)
	if err != nil {
		return nil, err
	}
	localIDs := NormalizeLocalIDs(opts.LocalIDs)
	explicitLocal := map[string]bool{}
	for _, id := range localIDs {
		explicitLocal[id] = true
		if !manifestContains(plugins, id) {
			return nil, fmt.Errorf("--local plugin %q not found in grounds.yaml", id)
		}
	}

	plan := &Plan{}
	for _, plugin := range plugins {
		entry, variant, ok := inferEntryForPlugin(cfg, plugin)
		selected := false
		if explicitLocal[plugin.ID] {
			selected = true
		} else if opts.WithLocal && ok && entry.Enabled {
			selected = true
		}
		if selected {
			if !ok {
				return nil, fmt.Errorf("local workspace entry for %q variant %q not found", plugin.ID, plugin.Variant)
			}
			if variant == "" {
				variant = plugin.Variant
			}
			local, err := resolveLocal(ctx, plugin, variant, entry)
			if err != nil {
				return nil, err
			}
			plan.Plugins = append(plan.Plugins, PlanPlugin{
				ID:        plugin.ID,
				Variant:   variant,
				LocalPath: local.LocalPath,
			})
			plan.EffectivePluginSources = append(plan.EffectivePluginSources, local.Effective)
			continue
		}

		plan.Plugins = append(plan.Plugins, PlanPlugin{
			ID:      plugin.ID,
			Variant: plugin.Variant,
			Source:  plugin.Source,
		})
		plan.EffectivePluginSources = append(plan.EffectivePluginSources, EffectiveSource{
			ID:        plugin.ID,
			Variant:   plugin.Variant,
			Effective: "release",
			Source:    plugin.Source,
		})
	}
	return plan, nil
}

func WritePlanFile(path string, plan *Plan) error {
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func loadManifestPlugins(path string) ([]manifestPlugin, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Plugins []yaml.Node `yaml:"plugins"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var plugins []manifestPlugin
	for i := range doc.Plugins {
		node := doc.Plugins[i]
		switch node.Kind {
		case yaml.ScalarNode:
			source := strings.TrimSpace(node.Value)
			plugins = append(plugins, manifestPlugin{
				ID:      inferIDFromSource(source),
				Source:  source,
				Variant: "",
			})
		case yaml.MappingNode:
			var plugin struct {
				ID      string `yaml:"id"`
				Variant string `yaml:"variant"`
				Source  string `yaml:"source"`
			}
			if err := node.Decode(&plugin); err != nil {
				return nil, err
			}
			if plugin.ID == "" {
				plugin.ID = inferIDFromSource(plugin.Source)
			}
			plugins = append(plugins, manifestPlugin{
				ID:      plugin.ID,
				Variant: plugin.Variant,
				Source:  plugin.Source,
			})
		default:
			return nil, fmt.Errorf("unsupported plugin entry at index %d", i)
		}
	}
	return plugins, nil
}

func manifestContains(plugins []manifestPlugin, id string) bool {
	for _, plugin := range plugins {
		if plugin.ID == id {
			return true
		}
	}
	return false
}

func inferEntryForPlugin(cfg *Config, plugin manifestPlugin) (ResolvedEntry, string, bool) {
	if cfg == nil {
		return ResolvedEntry{}, "", false
	}
	repo, ok := cfg.Repos[plugin.ID]
	if !ok {
		return ResolvedEntry{}, "", false
	}
	if plugin.Variant != "" {
		entry, ok := cfg.EntryForVariant(plugin.ID, plugin.Variant)
		return entry, plugin.Variant, ok
	}
	if repo.Artifact != "" {
		entry, ok := cfg.EntryForVariant(plugin.ID, "")
		return entry, "", ok
	}
	if len(repo.Variants) != 1 {
		return ResolvedEntry{}, "", false
	}
	for variant := range repo.Variants {
		entry, ok := cfg.EntryForVariant(plugin.ID, variant)
		return entry, variant, ok
	}
	return ResolvedEntry{}, "", false
}

type localResolution struct {
	LocalPath string
	Effective EffectiveSource
}

func resolveLocal(ctx context.Context, plugin manifestPlugin, variant string, entry ResolvedEntry) (localResolution, error) {
	if entry.Path == "" {
		return localResolution{}, fmt.Errorf("local workspace entry for %q has no path", plugin.ID)
	}
	if entry.Build != "" {
		if err := runBuild(ctx, entry.Path, entry.Build); err != nil {
			return localResolution{}, fmt.Errorf("failed to build local plugin %q: %w", plugin.ID, err)
		}
	}
	localPath, err := resolveArtifact(entry.Path, entry.Artifact)
	if err != nil {
		return localResolution{}, fmt.Errorf("failed to resolve local artifact for %q: %w", plugin.ID, err)
	}
	sum, err := sha256File(localPath)
	if err != nil {
		return localResolution{}, err
	}
	git, err := collectGitMetadata(ctx, entry.Path)
	if err != nil {
		return localResolution{}, err
	}
	return localResolution{
		LocalPath: localPath,
		Effective: EffectiveSource{
			ID:             plugin.ID,
			Variant:        variant,
			Effective:      "local",
			DefaultSource:  plugin.Source,
			ArtifactName:   filepath.Base(localPath),
			ArtifactSha256: sum,
			Git:            git,
		},
	}, nil
}

func runBuild(ctx context.Context, dir, build string) error {
	parts := strings.Fields(build)
	if len(parts) == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveArtifact(repoPath, pattern string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("artifact glob is empty")
	}
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(repoPath, filepath.FromSlash(pattern))
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	var jars []string
	for _, match := range matches {
		if strings.EqualFold(filepath.Ext(match), ".jar") {
			jars = append(jars, match)
		}
	}
	sort.Strings(jars)
	if len(jars) != 1 {
		return "", fmt.Errorf("expected exactly one .jar for %s, found %d", pattern, len(jars))
	}
	return jars[0], nil
}

func sha256File(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func collectGitMetadata(ctx context.Context, path string) (*GitMetadata, error) {
	root, err := gitOutput(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("failed to read git root for %s: %w", path, err)
	}
	commit, err := gitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to read git commit for %s: %w", root, err)
	}
	remote, _ := gitOutput(ctx, root, "config", "--get", "remote.origin.url")
	status, err := gitOutput(ctx, root, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to read git status for %s: %w", root, err)
	}
	return &GitMetadata{
		Remote: normalizeRemote(remote),
		Commit: strings.TrimSpace(commit),
		Dirty:  strings.TrimSpace(status) != "",
	}, nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	raw, err := cmd.Output()
	return strings.TrimSpace(string(raw)), err
}

func normalizeRemote(remote string) string {
	remote = strings.TrimSpace(remote)
	remote = strings.TrimSuffix(remote, ".git")
	remote = strings.TrimPrefix(remote, "git@github.com:")
	remote = strings.TrimPrefix(remote, "https://github.com/")
	return remote
}

func inferIDFromSource(source string) string {
	source = strings.TrimSpace(source)
	if beforeAt, _, ok := strings.Cut(source, "@"); ok {
		if slash := strings.LastIndex(beforeAt, "/"); slash >= 0 && slash < len(beforeAt)-1 {
			return beforeAt[slash+1:]
		}
	}
	if _, artifact, ok := strings.Cut(source, ":"); ok && artifact != "" {
		base := filepath.Base(artifact)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	base := filepath.Base(source)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
