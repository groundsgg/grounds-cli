package workspace

import (
	"os"
	"path/filepath"
)

var knownVariants = []string{"paper", "velocity", "minestom"}

func ScanRoots(roots []string) (*Config, error) {
	cfg := &Config{Repos: map[string]Repo{}}
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		children, err := os.ReadDir(absRoot)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			if !child.IsDir() {
				continue
			}
			repoPath := filepath.Join(absRoot, child.Name())
			if !isWorkspaceRepo(repoPath) {
				continue
			}
			cfg.Repos[child.Name()] = scanRepo(repoPath)
		}
	}
	return cfg, nil
}

func scanRepo(path string) Repo {
	repo := Repo{
		Path:     path,
		Enabled:  true,
		Variants: map[string]Variant{},
	}
	for _, variant := range knownVariants {
		if info, err := os.Stat(filepath.Join(path, variant)); err == nil && info.IsDir() {
			repo.Variants[variant] = Variant{
				Artifact: filepath.ToSlash(filepath.Join(variant, "build", "libs", "*.jar")),
				Build:    "./gradlew :" + variant + ":shadowJar",
				Enabled:  true,
			}
		}
	}
	if len(repo.Variants) == 0 {
		repo.Variants = nil
		repo.Artifact = filepath.ToSlash(filepath.Join("build", "libs", "*.jar"))
		repo.Build = "./gradlew build"
	}
	return repo
}

func isWorkspaceRepo(path string) bool {
	for _, marker := range []string{"settings.gradle.kts", "build.gradle.kts", "grounds.yaml"} {
		if info, err := os.Stat(filepath.Join(path, marker)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}
