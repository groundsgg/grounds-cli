package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/workspace"
)

type Source string

const (
	SourceExplicit      Source = "explicit"
	SourceEnv           Source = "env"
	SourceLocalDefault  Source = "local-default"
	SourceGlobalDefault Source = "global-default"
	SourceSingleProject Source = "single-project"
)

type Lister interface {
	ListProjects(context.Context) (*api.ProjectList, error)
}

type ResolveOptions struct {
	Explicit        string
	EnvProject      string
	Config          *config.Config
	WorkspaceConfig *workspace.Config
	WorkDir         string
	Client          Lister
}

type Selection struct {
	ID      string
	Project api.Project
	Source  Source
}

func Resolve(ctx context.Context, opts ResolveOptions) (Selection, error) {
	if opts.Client == nil {
		return Selection{}, fmt.Errorf("project resolver requires a projects client")
	}
	list, err := opts.Client.ListProjects(ctx)
	if err != nil {
		return Selection{}, err
	}
	projects := list.Items

	if value := strings.TrimSpace(opts.Explicit); value != "" {
		return selectNamed(projects, value, SourceExplicit)
	}
	if value := strings.TrimSpace(opts.EnvProject); value != "" {
		return selectNamed(projects, value, SourceEnv)
	}
	if value := localDefault(opts.WorkspaceConfig, opts.WorkDir); value != "" {
		return selectNamed(projects, value, SourceLocalDefault)
	}
	if opts.Config != nil && strings.TrimSpace(opts.Config.DefaultProjectID) != "" {
		return selectNamed(projects, opts.Config.DefaultProjectID, SourceGlobalDefault)
	}
	if len(projects) == 1 {
		return Selection{ID: projects[0].ID, Project: projects[0], Source: SourceSingleProject}, nil
	}
	if len(projects) == 0 {
		return Selection{}, fmt.Errorf("no projects available; open the portal or run a project-scoped command again after account setup")
	}
	return Selection{}, fmt.Errorf("multiple projects available; select one with `grounds project use <project>` or pass `--project <project>`")
}

func selectNamed(projects []api.Project, value string, source Source) (Selection, error) {
	for _, p := range projects {
		if p.ID == value || p.Slug == value {
			return Selection{ID: p.ID, Project: p, Source: source}, nil
		}
	}
	return Selection{}, fmt.Errorf("project %q not found; run `grounds project list` to see available projects", value)
}

func localDefault(cfg *workspace.Config, workDir string) string {
	if cfg == nil || len(cfg.ProjectDefaults) == 0 {
		return ""
	}
	root := WorkspaceRoot(workDir)
	return strings.TrimSpace(cfg.ProjectDefaults[root])
}

func WorkspaceRoot(workDir string) string {
	if workDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workDir = wd
		}
	}
	if workDir == "" {
		return ""
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		abs = workDir
	}
	current := abs
	for {
		if exists(filepath.Join(current, "grounds.yaml")) || exists(filepath.Join(current, ".git")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return abs
		}
		current = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
