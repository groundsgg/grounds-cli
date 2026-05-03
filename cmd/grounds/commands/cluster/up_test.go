package cluster

import (
	"os"
	"path/filepath"
	"testing"
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

	t.Run("missing bundle ref errors", func(t *testing.T) {
		path := writeTempYAML(t, `
overrides:
  velocity: {mode: image}
`)
		if _, err := loadBundleRequest("", path); err == nil {
			t.Error("expected error when no bundle ref provided")
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
