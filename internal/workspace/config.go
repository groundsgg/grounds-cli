package workspace

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/groundsgg/grounds-cli/internal/config"
	"gopkg.in/yaml.v3"
)

const FileName = "workspace.yaml"

type Config struct {
	Repos map[string]Repo `yaml:"repos,omitempty"`
}

type Repo struct {
	Path     string             `yaml:"path"`
	Artifact string             `yaml:"artifact,omitempty"`
	Build    string             `yaml:"build,omitempty"`
	Enabled  bool               `yaml:"enabled"`
	Variants map[string]Variant `yaml:"variants,omitempty"`
}

type Variant struct {
	Artifact string `yaml:"artifact,omitempty"`
	Build    string `yaml:"build,omitempty"`
	Enabled  bool   `yaml:"enabled"`
}

type ResolvedEntry struct {
	Path     string
	Artifact string
	Build    string
	Enabled  bool
}

func DefaultPath() (string, error) {
	dir, err := config.ResolveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Repos: map[string]Repo{}}, nil
		}
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, err
	}
	if cfg.Repos == nil {
		cfg.Repos = map[string]Repo{}
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Repos == nil {
		cfg.Repos = map[string]Repo{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func (c *Config) EntryForVariant(id, variant string) (ResolvedEntry, bool) {
	if c == nil {
		return ResolvedEntry{}, false
	}
	repo, ok := c.Repos[id]
	if !ok {
		return ResolvedEntry{}, false
	}
	if variant != "" && repo.Variants != nil {
		if v, ok := repo.Variants[variant]; ok {
			return ResolvedEntry{
				Path:     repo.Path,
				Artifact: firstNonEmpty(v.Artifact, repo.Artifact),
				Build:    firstNonEmpty(v.Build, repo.Build),
				Enabled:  v.Enabled,
			}, true
		}
	}
	if repo.Artifact == "" {
		return ResolvedEntry{}, false
	}
	return ResolvedEntry{
		Path:     repo.Path,
		Artifact: repo.Artifact,
		Build:    repo.Build,
		Enabled:  repo.Enabled,
	}, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
