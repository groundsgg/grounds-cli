package devspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSuccessSummary(t *testing.T) {
	if got := generateSuccessSummary("./devspace.yaml"); got != "Wrote ./devspace.yaml" {
		t.Fatalf("generateSuccessSummary = %q", got)
	}
}

func TestLoadGenerateInputs(t *testing.T) {
	t.Run("bundle flag only, no override", func(t *testing.T) {
		bundle, override, err := loadGenerateInputs("0.4.0", "", "plugin-social")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bundle != "0.4.0" {
			t.Errorf("bundle = %q, want 0.4.0", bundle)
		}
		if override != nil {
			t.Errorf("override should be nil, got %v", override)
		}
	})

	t.Run("override file with bundle field, no flag", func(t *testing.T) {
		path := writeTempYAML(t, `
bundle: 1.5.0
overrides:
  plugin-social:
    mode: gradle-local
    project: ./plugin-social
    artifact: build/libs/*-all.jar
`)
		bundle, override, err := loadGenerateInputs("", path, "plugin-social")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bundle != "1.5.0" {
			t.Errorf("bundle = %q, want 1.5.0", bundle)
		}
		if override["mode"] != "gradle-local" {
			t.Errorf("override.mode = %v, want gradle-local", override["mode"])
		}
		if override["project"] != "./plugin-social" {
			t.Errorf("override.project = %v", override["project"])
		}
	})

	t.Run("flag wins over file's bundle field", func(t *testing.T) {
		path := writeTempYAML(t, `
bundle: 1.5.0
overrides: {}
`)
		bundle, _, err := loadGenerateInputs("0.4.0", path, "plugin-social")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bundle != "0.4.0" {
			t.Errorf("bundle = %q, want 0.4.0 (flag wins)", bundle)
		}
	})

	t.Run("missing bundle ref errors", func(t *testing.T) {
		path := writeTempYAML(t, `overrides: {}`)
		if _, _, err := loadGenerateInputs("", path, "plugin-social"); err == nil {
			t.Error("expected error when no bundle ref provided")
		}
	})

	t.Run("missing both flag and file errors", func(t *testing.T) {
		if _, _, err := loadGenerateInputs("", "", "plugin-social"); err == nil {
			t.Error("expected error when neither --bundle nor --override given")
		}
	})

	t.Run("component missing from override file uses nil override", func(t *testing.T) {
		path := writeTempYAML(t, `
bundle: 0.4.0
overrides:
  plugin-chat:
    mode: gradle-local
    project: ./plugin-chat
    artifact: build/libs/*-all.jar
`)
		_, override, err := loadGenerateInputs("", path, "plugin-social")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if override != nil {
			t.Errorf("expected nil override for component not in file, got %v", override)
		}
	})
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "override.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return p
}
