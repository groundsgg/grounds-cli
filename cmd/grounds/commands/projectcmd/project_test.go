package projectcmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/project"
	"github.com/groundsgg/grounds-cli/internal/workspace"
)

func TestProjectListPrintsAvailableProjects(t *testing.T) {
	withProjectAPITest(t, func(serverURL, configDir string) {
		t.Setenv("GROUNDS_API_URL", serverURL)
		t.Setenv("GROUNDS_CONFIG_DIR", configDir)
		t.Setenv("GROUNDS_TOKEN", "token")

		var out bytes.Buffer
		cmd := NewProjectCommand()
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		got := out.String()
		for _, want := range []string{"main", "team", "owner", "editor"} {
			if !strings.Contains(got, want) {
				t.Fatalf("output = %q, want it to contain %q", got, want)
			}
		}
	})
}

func TestProjectUseStoresGlobalDefaultByID(t *testing.T) {
	withProjectAPITest(t, func(serverURL, configDir string) {
		t.Setenv("GROUNDS_API_URL", serverURL)
		t.Setenv("GROUNDS_CONFIG_DIR", configDir)
		t.Setenv("GROUNDS_TOKEN", "token")

		cmd := NewProjectCommand()
		cmd.SetArgs([]string{"use", "team"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		cfg, err := config.Load(configDir)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.DefaultProjectID != "p-team" {
			t.Fatalf("DefaultProjectID = %q", cfg.DefaultProjectID)
		}
	})
}

func TestProjectUseStoresLocalDefaultByWorkspaceRoot(t *testing.T) {
	withProjectAPITest(t, func(serverURL, configDir string) {
		t.Setenv("GROUNDS_API_URL", serverURL)
		t.Setenv("GROUNDS_CONFIG_DIR", configDir)
		t.Setenv("GROUNDS_TOKEN", "token")
		repo := t.TempDir()
		if err := os.WriteFile(filepath.Join(repo, "grounds.yaml"), []byte("name: plugin-chat\n"), 0o600); err != nil {
			t.Fatalf("write grounds.yaml: %v", err)
		}
		t.Chdir(repo)

		cmd := NewProjectCommand()
		cmd.SetArgs([]string{"use", "team", "--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		cfg, err := workspace.Load("")
		if err != nil {
			t.Fatalf("workspace.Load: %v", err)
		}
		if cfg.ProjectDefaults[project.WorkspaceRoot(repo)] != "p-team" {
			t.Fatalf("ProjectDefaults = %#v", cfg.ProjectDefaults)
		}
	})
}

func TestProjectClearRemovesGlobalDefault(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{APIURL: "https://api.grounds.gg", DefaultTarget: "dev", Output: "table", Color: "auto", DefaultProjectID: "p-team"}
	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv("GROUNDS_CONFIG_DIR", dir)

	cmd := NewProjectCommand()
	cmd.SetArgs([]string{"clear"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DefaultProjectID != "" {
		t.Fatalf("DefaultProjectID = %q", got.DefaultProjectID)
	}
}

func withProjectAPITest(t *testing.T, run func(serverURL, configDir string)) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects" {
			t.Fatalf("path = %q, want /v1/projects", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"p-main","slug":"main","name":"Main","role":"owner","createdAt":"2026-07-07T10:00:00.000Z"},{"id":"p-team","slug":"team","name":"Team","role":"editor","createdAt":"2026-07-07T10:00:00.000Z"}]}`))
	}))
	t.Cleanup(server.Close)
	run(server.URL, t.TempDir())
}
