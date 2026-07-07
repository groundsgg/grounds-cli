package project

import (
	"context"
	"strings"
	"testing"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/workspace"
)

type fakeProjectsClient struct {
	projects []api.Project
}

func (f fakeProjectsClient) ListProjects(context.Context) (*api.ProjectList, error) {
	return &api.ProjectList{Items: f.projects}, nil
}

func TestResolveUsesExplicitProjectBeforeDefaults(t *testing.T) {
	got, err := Resolve(context.Background(), ResolveOptions{
		Explicit:        "team",
		Config:          &config.Config{DefaultProjectID: "personal-id"},
		WorkspaceConfig: &workspace.Config{ProjectDefaults: map[string]string{"/repo": "local-id"}},
		WorkDir:         "/repo",
		Client: fakeProjectsClient{projects: []api.Project{
			{ID: "team-id", Slug: "team", Name: "Team"},
			{ID: "personal-id", Slug: "personal", Name: "Personal"},
			{ID: "local-id", Slug: "local", Name: "Local"},
		}},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.ID != "team-id" || got.Source != SourceExplicit {
		t.Fatalf("selection = %#v", got)
	}
}

func TestResolveUsesEnvBeforeLocalDefault(t *testing.T) {
	got, err := Resolve(context.Background(), ResolveOptions{
		EnvProject:      "team-id",
		WorkspaceConfig: &workspace.Config{ProjectDefaults: map[string]string{"/repo": "local-id"}},
		WorkDir:         "/repo",
		Client: fakeProjectsClient{projects: []api.Project{
			{ID: "team-id", Slug: "team", Name: "Team"},
			{ID: "local-id", Slug: "local", Name: "Local"},
		}},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.ID != "team-id" || got.Source != SourceEnv {
		t.Fatalf("selection = %#v", got)
	}
}

func TestResolveUsesLocalDefaultBeforeGlobalDefault(t *testing.T) {
	repo := t.TempDir()
	root := WorkspaceRoot(repo)
	got, err := Resolve(context.Background(), ResolveOptions{
		Config:          &config.Config{DefaultProjectID: "global-id"},
		WorkspaceConfig: &workspace.Config{ProjectDefaults: map[string]string{root: "local-id"}},
		WorkDir:         repo,
		Client: fakeProjectsClient{projects: []api.Project{
			{ID: "global-id", Slug: "global", Name: "Global"},
			{ID: "local-id", Slug: "local", Name: "Local"},
		}},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.ID != "local-id" || got.Source != SourceLocalDefault {
		t.Fatalf("selection = %#v", got)
	}
}

func TestResolveUsesSingleProjectWhenNoDefaultExists(t *testing.T) {
	got, err := Resolve(context.Background(), ResolveOptions{
		Config:          &config.Config{},
		WorkspaceConfig: &workspace.Config{},
		WorkDir:         "/repo",
		Client:          fakeProjectsClient{projects: []api.Project{{ID: "only-id", Slug: "only", Name: "Only"}}},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.ID != "only-id" || got.Source != SourceSingleProject {
		t.Fatalf("selection = %#v", got)
	}
}

func TestResolveFailsWhenMultipleProjectsHaveNoDefault(t *testing.T) {
	_, err := Resolve(context.Background(), ResolveOptions{
		Config:          &config.Config{},
		WorkspaceConfig: &workspace.Config{},
		WorkDir:         "/repo",
		Client: fakeProjectsClient{projects: []api.Project{
			{ID: "one", Slug: "one", Name: "One"},
			{ID: "two", Slug: "two", Name: "Two"},
		}},
	})
	if err == nil {
		t.Fatal("Resolve error = nil, want multiple-project guidance")
	}
	if !strings.Contains(err.Error(), "multiple projects") || !strings.Contains(err.Error(), "grounds project use") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestResolveRejectsUnknownExplicitProject(t *testing.T) {
	_, err := Resolve(context.Background(), ResolveOptions{
		Explicit: "missing",
		Client:   fakeProjectsClient{projects: []api.Project{{ID: "p1", Slug: "main", Name: "Main"}}},
	})
	if err == nil {
		t.Fatal("Resolve error = nil, want project not found")
	}
	if !strings.Contains(err.Error(), "project \"missing\" not found") {
		t.Fatalf("error = %q", err.Error())
	}
}
