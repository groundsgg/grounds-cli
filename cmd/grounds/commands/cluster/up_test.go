package cluster

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/render"
)

func TestLoadBundleRequest(t *testing.T) {
	t.Run("bundle flag only", func(t *testing.T) {
		body, err := loadBundleRequest("0.4.0", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body.Bundle != "0.4.0" {
			t.Errorf("Bundle = %q, want 0.4.0", body.Bundle)
		}
		if body.Overrides != nil {
			t.Errorf("Overrides should be nil, got %v", body.Overrides)
		}
	})

	t.Run("override file with bundle field", func(t *testing.T) {
		path := writeTempYAML(t, `
bundle: 1.5.0
overrides:
  plugin-social:
    mode: gradle-local
    project: ./plugin-social
`)
		body, err := loadBundleRequest("", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body.Bundle != "1.5.0" {
			t.Errorf("Bundle = %q, want 1.5.0 (read from file)", body.Bundle)
		}
		if body.Overrides["plugin-social"] == nil {
			t.Error("expected plugin-social override to be set")
		}
	})

	t.Run("flag wins over file bundle field", func(t *testing.T) {
		path := writeTempYAML(t, `
bundle: 1.5.0
overrides:
  velocity:
    mode: image
    tag: dev-fix-routing
`)
		body, err := loadBundleRequest("0.4.0", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body.Bundle != "0.4.0" {
			t.Errorf("Bundle = %q, want 0.4.0 (flag wins)", body.Bundle)
		}
	})

	t.Run("missing bundle ref defaults to main", func(t *testing.T) {
		// platform-bundle is the default profile: no ref on the flag or in the
		// override file → track the bundle's main branch (not an error).
		path := writeTempYAML(t, `
overrides:
  velocity: {mode: image}
`)
		body, err := loadBundleRequest("", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body.Bundle != "main" {
			t.Errorf("Bundle = %q, want main (default)", body.Bundle)
		}
		if body.Overrides["velocity"] == nil {
			t.Error("expected velocity override to survive the ref default")
		}
	})

	t.Run("bare up (no flag, no file) defaults to main", func(t *testing.T) {
		body, err := loadBundleRequest("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body.Bundle != "main" {
			t.Errorf("Bundle = %q, want main (default)", body.Bundle)
		}
	})

	t.Run("missing file errors", func(t *testing.T) {
		if _, err := loadBundleRequest("0.4.0", "/nonexistent/path.yaml"); err == nil {
			t.Error("expected error for nonexistent override file")
		}
	})

	t.Run("invalid yaml errors", func(t *testing.T) {
		path := writeTempYAML(t, "this: is: not: valid: yaml::")
		if _, err := loadBundleRequest("0.4.0", path); err == nil {
			t.Error("expected error for malformed yaml")
		}
	})
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "override.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

func TestRenderBundleWaitProgressWritesSparseNonTTYProgressOnce(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	state := &bundleWaitRenderState{}
	status := &api.ClusterStatus{
		State: "creating",
		BundleProgress: &api.BundleProgress{
			Phase:            "deploying_components",
			CurrentComponent: "plugin-config",
			ComponentsTotal:  14,
			ComponentsDone:   7,
		},
	}

	renderBundleWaitProgress(&buf, state, status, 0, 0)
	renderBundleWaitProgress(&buf, state, status, 0, 0)

	want := "    • phase: deploying components 7/14: plugin-config\n"
	if got := buf.String(); got != want {
		t.Fatalf("renderBundleWaitProgress output = %q, want %q", got, want)
	}
}

func TestWaitForBundleRefreshesInteractiveSpinnerBetweenPolls(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	calls := 0
	getCluster := func(context.Context) (*api.ClusterStatus, error) {
		calls++
		if calls == 1 {
			return &api.ClusterStatus{
				State: "creating",
				BundleProgress: &api.BundleProgress{
					Phase:            "deploying_components",
					CurrentComponent: "plugin-config",
					ComponentsTotal:  14,
					ComponentsDone:   7,
				},
			}, nil
		}
		return &api.ClusterStatus{State: "active"}, nil
	}
	interactive := true

	final, err := waitForBundleWithOptions(context.Background(), getCluster, &buf, bundleWaitOptions{
		pollInterval:    50 * time.Millisecond,
		spinnerInterval: 10 * time.Millisecond,
		overall:         time.Second,
		rowGrace:        30 * time.Millisecond,
		interactive:     &interactive,
		spinner:         render.NewSpinnerLineForTerminal(&buf, true, 120),
	})
	if err != nil {
		t.Fatalf("waitForBundleWithOptions returned error: %v", err)
	}
	if final.State != "active" {
		t.Fatalf("final state = %q, want active", final.State)
	}
	if calls != 2 {
		t.Fatalf("GetCluster calls = %d, want 2", calls)
	}
	if got := strings.Count(buf.String(), "phase: deploying components 7/14: plugin-config"); got < 2 {
		t.Fatalf("spinner redraws = %d, want at least 2; output = %q", got, buf.String())
	}
}

func TestRenderBundleStatus(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	s := &api.ClusterStatus{State: "active", Namespace: "dev-lukas"}
	s.BundleResult = &api.BundleResult{Succeeded: []string{"api"}}
	s.BundleResult.Failed = append(s.BundleResult.Failed, struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	}{Name: "worker", Error: "image pull failed"})

	var buf bytes.Buffer
	renderBundleStatus(&buf, s)

	want := "[!] Workspace - active in namespace dev-lukas\n" +
		"    ! Components: 1 succeeded, 1 failed\n" +
		"    ✗ worker: image pull failed\n"
	if got := buf.String(); got != want {
		t.Fatalf("renderBundleStatus output = %q, want %q", got, want)
	}
}

func TestRenderBundleStatusFailed(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	s := &api.ClusterStatus{State: "failed", FailureReason: "vcluster kubeconfig secret did not appear"}
	var buf bytes.Buffer
	renderBundleStatus(&buf, s)

	want := "[✗] Workspace - bundle provisioning failed\n" +
		"    ✗ vcluster kubeconfig secret did not appear\n"
	if got := buf.String(); got != want {
		t.Fatalf("renderBundleStatus(failed) output = %q, want %q", got, want)
	}
}
