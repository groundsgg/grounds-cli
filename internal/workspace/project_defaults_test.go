package workspace

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadProjectDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.yaml")
	cfg := &Config{
		ProjectDefaults: map[string]string{
			"/repo": "project-id",
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ProjectDefaults["/repo"] != "project-id" {
		t.Fatalf("ProjectDefaults = %#v", got.ProjectDefaults)
	}
}
