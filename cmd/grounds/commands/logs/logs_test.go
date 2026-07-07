package logs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/groundsgg/grounds-cli/internal/config"
)

func TestLogsExamplesIncludeRequiredPushID(t *testing.T) {
	cmd := NewLogsCommand()

	for _, example := range []string{
		"grounds logs <pushId>",
		"grounds logs <pushId> --follow",
		"grounds logs deployment <name>",
	} {
		if !strings.Contains(cmd.Example, example) {
			t.Fatalf("logs examples = %q, want %q", cmd.Example, example)
		}
	}

	if strings.Contains(cmd.Example, "grounds logs\n") || strings.Contains(cmd.Example, "grounds logs --follow") {
		t.Fatalf("logs examples = %q, should include required <pushId>", cmd.Example)
	}
}

func TestBuildStreamScopesPushLogsToDefaultProject(t *testing.T) {
	withLogsProjectAPITest(t, func(serverURL, configDir string) {
		t.Setenv("GROUNDS_API_URL", serverURL)
		t.Setenv("GROUNDS_CONFIG_DIR", configDir)
		t.Setenv("GROUNDS_TOKEN", "token")
		if err := config.Save(configDir, &config.Config{
			APIURL:           serverURL,
			DefaultTarget:    "dev",
			Output:           "table",
			Color:            "auto",
			DefaultProjectID: "p-team",
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		stream, err := buildStream(context.Background(), NewLogsCommand(), "push 1", "push")
		if err != nil {
			t.Fatalf("buildStream: %v", err)
		}
		want := serverURL + "/v1/pushes/push%201/logs?projectId=p-team"
		if stream.URL != want {
			t.Fatalf("URL = %q, want %q", stream.URL, want)
		}
		if stream.Token != "token" {
			t.Fatalf("Token = %q", stream.Token)
		}
	})
}

func TestBuildStreamScopesDeploymentLogsToDefaultProject(t *testing.T) {
	withLogsProjectAPITest(t, func(serverURL, configDir string) {
		t.Setenv("GROUNDS_API_URL", serverURL)
		t.Setenv("GROUNDS_CONFIG_DIR", configDir)
		t.Setenv("GROUNDS_TOKEN", "token")
		if err := config.Save(configDir, &config.Config{
			APIURL:           serverURL,
			DefaultTarget:    "dev",
			Output:           "table",
			Color:            "auto",
			DefaultProjectID: "p-team",
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		stream, err := buildStream(context.Background(), NewLogsCommand(), "service/api", "deployment")
		if err != nil {
			t.Fatalf("buildStream: %v", err)
		}
		want := serverURL + "/v1/deployments/service%2Fapi/logs?projectId=p-team"
		if stream.URL != want {
			t.Fatalf("URL = %q, want %q", stream.URL, want)
		}
	})
}

func withLogsProjectAPITest(t *testing.T, run func(serverURL, configDir string)) {
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
