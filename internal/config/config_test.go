package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.APIURL != "https://api.grounds.gg" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	if cfg.DefaultTarget != "dev" {
		t.Errorf("DefaultTarget = %q", cfg.DefaultTarget)
	}
	if cfg.Output != "table" {
		t.Errorf("Output = %q", cfg.Output)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("apiUrl: https://example.test\noutput: json\ndefaultProjectId: project-id\n"), 0600)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.APIURL != "https://example.test" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	if cfg.Output != "json" {
		t.Errorf("Output = %q", cfg.Output)
	}
	if cfg.DefaultTarget != "dev" {
		t.Errorf("DefaultTarget = %q (expected default)", cfg.DefaultTarget)
	}
	if cfg.DefaultProjectID != "project-id" {
		t.Errorf("DefaultProjectID = %q", cfg.DefaultProjectID)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("GROUNDS_API_URL", "https://override.test")
	dir := t.TempDir()
	cfg, _ := Load(dir)
	if cfg.APIURL != "https://override.test" {
		t.Errorf("env override failed: %q", cfg.APIURL)
	}
}

func TestSaveProjectDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.DefaultProjectID = "project-id"
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if got.DefaultProjectID != "project-id" {
		t.Fatalf("DefaultProjectID = %q", got.DefaultProjectID)
	}
}
