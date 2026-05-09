package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
)

func TestInit_NonInteractive(t *testing.T) {
	stubInitCatalog(t, fallbackBaseImageCatalog(), nil)
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--app-name=my-arena", "--type=gamemode", "--base-image=paper-gamemode"})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	body, err := os.ReadFile("grounds.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Contains(body, []byte("name: my-arena")) {
		t.Errorf("body = %s", body)
	}
	if !bytes.Contains(body, []byte("baseImage: paper-gamemode")) {
		t.Errorf("body = %s", body)
	}
	if got := buf.String(); got != "[✓] Init - Wrote grounds.yaml\n    • Next: run `grounds push`.\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestInit_NonInteractiveRejectsMismatchedCatalogBaseImage(t *testing.T) {
	stubInitCatalog(t, &api.BaseImageCatalog{Items: []api.BaseImageSource{{
		Key: "velocity", DisplayName: "Velocity", ManifestType: "plugin-velocity",
	}}}, nil)
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--app-name=my-plugin", "--type=plugin-paper", "--base-image=velocity"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "requires type plugin-velocity") {
		t.Fatalf("error = %v", err)
	}
}

func TestInit_NonInteractiveUsesFallbackCatalogWhenFetchFails(t *testing.T) {
	stubInitCatalog(t, nil, errors.New("network down"))
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--app-name=my-arena", "--type=gamemode", "--base-image=paper-gamemode"})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	body, _ := os.ReadFile("grounds.yaml")
	if !bytes.Contains(body, []byte("baseImage: paper-gamemode")) {
		t.Fatalf("body = %s", body)
	}
	if !strings.Contains(buf.String(), "Using built-in base image defaults") {
		t.Fatalf("output = %q", buf.String())
	}
}

func stubInitCatalog(t *testing.T, catalog *api.BaseImageCatalog, err error) {
	t.Helper()
	old := loadInitBaseImageCatalog
	loadInitBaseImageCatalog = func(context.Context, *cobra.Command) (*api.BaseImageCatalog, error) {
		return catalog, err
	}
	t.Cleanup(func() { loadInitBaseImageCatalog = old })
}
