package push

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
)

// Validates the --target flag's allow-list before grounds-push gets
// invoked. The Gradle plugin already rejects bogus targets but
// surfacing the error here gives a faster fail-loud signal and a
// cleaner CLI error.
func TestPushRejectsInvalidTarget(t *testing.T) {
	cmd := newPush()
	cmd.SetArgs([]string{"--target=production"})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetOut(&stderr)
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error for invalid target, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --target") {
		t.Errorf("expected 'invalid --target' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "dev") || !strings.Contains(err.Error(), "staging") {
		t.Errorf("expected error to mention 'dev' and 'staging', got: %v", err)
	}
}

// Default value should match the help text + Gradle plugin default.
func TestPushDefaultTargetIsDev(t *testing.T) {
	cmd := newPush()
	flag := cmd.Flag("target")
	if flag == nil {
		t.Fatal("expected --target flag to exist")
	}
	if flag.DefValue != "dev" {
		t.Errorf("expected default --target=dev, got %q", flag.DefValue)
	}
}

func TestPushTargetCompletion(t *testing.T) {
	cmd := newPush()
	completion, ok := cmd.GetFlagCompletionFunc("target")
	if !ok {
		t.Fatal("expected --target completion function")
	}

	got, directive := completion(cmd, nil, "")
	want := []string{"dev", "staging"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("target completions = %v, want %v", got, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion directive = %v, want %v", directive, cobra.ShellCompDirectiveNoFileComp)
	}
}

func TestPushFlavorFlagIsForwardedToGradle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX shell gradle wrapper")
	}
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("Chdir(%q) error = %v", cwd, err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	if err := os.WriteFile("grounds.yaml", []byte("name: plugin-config\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(grounds.yaml) error = %v", err)
	}
	argsPath := filepath.Join(dir, "args.txt")
	wrapper := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(argsPath) + "\n"
	if err := os.WriteFile("gradlew", []byte(wrapper), 0o755); err != nil {
		t.Fatalf("WriteFile(gradlew) error = %v", err)
	}
	t.Setenv("GROUNDS_TOKEN", "test-token")

	cmd := newPush()
	cmd.SetArgs([]string{"--flavor= velocity "})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	raw, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("ReadFile(args) error = %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, "groundsPush\n") || !strings.Contains(got, "--flavor=velocity\n") {
		t.Fatalf("gradle args = %q, want groundsPush and --flavor=velocity", got)
	}
	if strings.Contains(got, "--flavor= velocity ") {
		t.Fatalf("gradle args = %q, want normalized flavor value", got)
	}
}

func TestPushMinestomFlavorBuildsDistributionAndUploads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell gradle wrapper")
	}
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("Chdir(%q) error = %v", cwd, err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	writePushTestFile(t, "grounds.yaml", `
name: minestom-demo
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
    build:
      task: :server:distTar
      artifact: server/build/distributions/*.tar
    modules:
      - id: plugin-agones
        variant: minestom
        source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
`)
	if err := os.MkdirAll("server/build/distributions", 0o755); err != nil {
		t.Fatalf("MkdirAll(distributions) error = %v", err)
	}
	writePushGradleTar(t, "server/build/distributions/minestom-demo.tar", "minestom-demo-local-SNAPSHOT")
	writePushTestFile(t, "gradlew", "#!/bin/sh\nprintf '%s\\n' \"$@\" > args.txt\n")
	if err := os.Chmod("gradlew", 0o755); err != nil {
		t.Fatalf("Chmod(gradlew) error = %v", err)
	}

	workspaceDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(workspaceDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(config) error = %v", err)
	}
	t.Setenv("GROUNDS_CONFIG_DIR", workspaceDir)
	t.Setenv("GROUNDS_TOKEN", "test-token")
	writePushTestFile(t, filepath.Join(workspaceDir, "workspace.yaml"), `
repos:
  plugin-agones:
    path: `+filepath.ToSlash(filepath.Join(dir, "plugin-agones"))+`
    variants:
      minestom:
        enabled: true
`)

	var uploaded bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploaded = true
		if r.Method != http.MethodPost || r.URL.Path != "/v1/pushes" {
			t.Errorf("request = %s %s, want POST /v1/pushes", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm() error = %v", err)
			return
		}
		if got := r.FormValue("flavor"); got != "minestom" {
			t.Errorf("form flavor = %q, want minestom", got)
		}
		manifest := r.FormValue("manifest")
		if !strings.Contains(manifest, `"type":"minestom-server"`) {
			t.Errorf("manifest = %q, want minestom-server type", manifest)
		}
		if !strings.Contains(manifest, `"flavors"`) {
			t.Errorf("manifest = %q, want flavored manifest payload", manifest)
		}
		file, header, err := r.FormFile("jar")
		if err != nil {
			t.Errorf("FormFile(jar) error = %v", err)
			return
		}
		defer file.Close()
		if header.Filename != "app.tar.gz" {
			t.Errorf("jar filename = %q, want app.tar.gz", header.Filename)
		}
		magic := make([]byte, 2)
		if _, err := file.Read(magic); err != nil {
			t.Errorf("Read(jar magic) error = %v", err)
			return
		}
		if magic[0] != 0x1f || magic[1] != 0x8b {
			t.Errorf("jar magic = %#v, want gzip magic", magic)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"pushId": "push-1", "status": "building"})
	}))
	defer srv.Close()
	t.Setenv("GROUNDS_API_URL", srv.URL)

	cmd := newPush()
	cmd.SetArgs([]string{"--flavor=minestom", "--with-local"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !uploaded {
		t.Fatal("expected upload request")
	}
	raw, err := os.ReadFile("args.txt")
	if err != nil {
		t.Fatalf("ReadFile(args.txt) error = %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, ":server:distTar\n") || !strings.Contains(got, "-I\n") {
		t.Fatalf("gradle args = %q, want :server:distTar and -I", got)
	}
}

func TestPushTopLevelMinestomIgnoresWorkspaceWithoutLocalFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell gradle wrapper")
	}
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("Chdir(%q) error = %v", cwd, err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	writePushTestFile(t, "grounds.yaml", `
name: minestom-demo
type: minestom-server
baseImage: minestom
build:
  task: :server:distTar
  artifact: server/build/distributions/*.tar
modules:
  - id: plugin-agones
    variant: minestom
    source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
`)
	if err := os.MkdirAll("server/build/distributions", 0o755); err != nil {
		t.Fatalf("MkdirAll(distributions) error = %v", err)
	}
	writePushGradleTar(t, "server/build/distributions/minestom-demo.tar", "minestom-demo-local-SNAPSHOT")
	writePushTestFile(t, "gradlew", "#!/bin/sh\nprintf '%s\\n' \"$@\" > args.txt\n")
	if err := os.Chmod("gradlew", 0o755); err != nil {
		t.Fatalf("Chmod(gradlew) error = %v", err)
	}

	workspaceDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(workspaceDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(config) error = %v", err)
	}
	t.Setenv("GROUNDS_CONFIG_DIR", workspaceDir)
	t.Setenv("GROUNDS_TOKEN", "test-token")
	writePushTestFile(t, filepath.Join(workspaceDir, "workspace.yaml"), "repos: [\n")

	var uploaded bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploaded = true
		if got := r.FormValue("flavor"); got != "" {
			t.Errorf("form flavor = %q, want empty for top-level manifest", got)
		}
		if manifest := r.FormValue("manifest"); !strings.Contains(manifest, `"type":"minestom-server"`) || strings.Contains(manifest, `"flavors"`) {
			t.Errorf("manifest = %q, want top-level minestom manifest", manifest)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"pushId": "push-1", "status": "building"})
	}))
	defer srv.Close()
	t.Setenv("GROUNDS_API_URL", srv.URL)

	cmd := newPush()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !uploaded {
		t.Fatal("expected upload request")
	}
	raw, err := os.ReadFile("args.txt")
	if err != nil {
		t.Fatalf("ReadFile(args.txt) error = %v", err)
	}
	if got := string(raw); strings.Contains(got, "-I\n") {
		t.Fatalf("gradle args = %q, did not expect composite init script", got)
	}
}

func TestPushTopLevelMinestomRejectsUnexpectedFlavor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell gradle wrapper")
	}
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("Chdir(%q) error = %v", cwd, err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	writePushTestFile(t, "grounds.yaml", `
name: minestom-demo
type: minestom-server
baseImage: minestom
build:
  task: :server:distTar
  artifact: server/build/distributions/*.tar
modules:
  - id: plugin-agones
    variant: minestom
    source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
`)
	writePushTestFile(t, "gradlew", "#!/bin/sh\nprintf '%s\\n' \"$@\" > args.txt\n")
	if err := os.Chmod("gradlew", 0o755); err != nil {
		t.Fatalf("Chmod(gradlew) error = %v", err)
	}
	t.Setenv("GROUNDS_TOKEN", "test-token")

	cmd := newPush()
	cmd.SetArgs([]string{"--flavor=paper"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `unknown flavor "paper"`) {
		t.Fatalf("Execute() error = %v, want unknown flavor", err)
	}
	if _, err := os.Stat("args.txt"); err == nil {
		t.Fatal("gradle should not run for invalid top-level minestom flavor")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(args.txt) error = %v", err)
	}
}

func TestPushRootOwnsDeployFlagsAndSubcommands(t *testing.T) {
	cmd := NewPushCommand()

	if flag := cmd.Flag("target"); flag == nil {
		t.Fatal("expected root push command to define --target")
	} else if flag.DefValue != "dev" {
		t.Fatalf("default --target = %q, want %q", flag.DefValue, "dev")
	}

	for _, name := range []string{"list", "retry"} {
		if sub, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("Find(%q) error = %v", name, err)
		} else if sub.Name() != name {
			t.Fatalf("Find(%q) = %q, want %q", name, sub.Name(), name)
		}
	}

	if sub, _, err := cmd.Find([]string{"push"}); err == nil && sub.Name() == "push" && sub != cmd {
		t.Fatalf("unexpected nested push subcommand found")
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func writePushTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimPrefix(body, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writePushGradleTar(t *testing.T, path, root string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer file.Close()
	tw := tar.NewWriter(file)
	defer tw.Close()

	entries := []struct {
		name string
		mode int64
		body string
		dir  bool
	}{
		{root + "/", 0o755, "", true},
		{root + "/bin/", 0o755, "", true},
		{root + "/bin/minestom-demo", 0o755, "#!/bin/sh\n", false},
		{root + "/bin/minestom-demo.bat", 0o644, "@echo off\r\n", false},
		{root + "/lib/", 0o755, "", true},
		{root + "/lib/minestom-demo.jar", 0o644, "jar", false},
	}
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Mode: entry.mode}
		if entry.dir {
			header.Typeflag = tar.TypeDir
		} else {
			header.Typeflag = tar.TypeReg
			header.Size = int64(len(entry.body))
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%q) error = %v", entry.name, err)
		}
		if entry.body != "" {
			if _, err := tw.Write([]byte(entry.body)); err != nil {
				t.Fatalf("Write(%q) error = %v", entry.name, err)
			}
		}
	}
}

func TestPushRootRejectsUnexpectedArgsBeforeDeployWork(t *testing.T) {
	for _, args := range [][]string{
		{"definitely-not-a-command"},
		{"push"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := NewPushCommand()
			cmd.SetArgs(args)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected unexpected argument error")
			}
			got := err.Error()
			if !strings.Contains(got, "unknown command") {
				t.Fatalf("error = %q, want argument validation error", got)
			}
			if strings.Contains(got, "Run `grounds init`") || strings.Contains(got, "Not a Gradle project") {
				t.Fatalf("error = %q, should not enter deploy path", got)
			}
		})
	}
}

func TestPushListRejectsUnexpectedArgsBeforeAPIWork(t *testing.T) {
	cmd := NewPushCommand()
	cmd.SetArgs([]string{"list", "unexpected"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unexpected argument error")
	}
	got := err.Error()
	if !strings.Contains(got, "unknown command") && !strings.Contains(got, "arg(s)") {
		t.Fatalf("error = %q, want argument validation error", got)
	}
	if strings.Contains(got, "credentials") || strings.Contains(got, "GROUNDS_TOKEN") {
		t.Fatalf("error = %q, should not enter auth/API path", got)
	}
}

func TestPushDeployCommandRejectsUnexpectedArgsBeforeDeployWork(t *testing.T) {
	cmd := newPush()
	cmd.SetArgs([]string{"definitely-not-a-command"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unexpected argument error")
	}
	got := err.Error()
	if !strings.Contains(got, "unknown command") && !strings.Contains(got, "arg(s)") {
		t.Fatalf("error = %q, want argument validation error", got)
	}
	if strings.Contains(got, "Run `grounds init`") || strings.Contains(got, "Not a Gradle project") {
		t.Fatalf("error = %q, should not enter deploy path", got)
	}
}

func TestPushMissingGradleWrapperSuggestsCommand(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("Chdir(%q) error = %v", cwd, err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	cmd := newPush()
	cmd.SetArgs([]string{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected missing Gradle wrapper error")
	}
	got := err.Error()
	if !strings.Contains(got, "Run `grounds init`") {
		t.Fatalf("error = %q, want command suggestion", got)
	}
	if strings.Contains(got, oldArrow()) || strings.Contains(got, singleQuotedCommand("grounds init")) {
		t.Fatalf("error = %q, should not use arrows or single-quoted commands", got)
	}
}

func TestPushAuthRefreshErrorSuggestsLoginCommand(t *testing.T) {
	err := authRefreshError(errors.New("token expired"))

	got := err.Error()
	if !strings.Contains(got, "Run `grounds login`") {
		t.Fatalf("error = %q, want login command suggestion", got)
	}
	if strings.Contains(got, oldArrow()) || strings.Contains(got, singleQuotedCommand("grounds login")) {
		t.Fatalf("error = %q, should not use arrows or single-quoted commands", got)
	}
}

func oldArrow() string {
	return string(rune(0x2192))
}

func singleQuotedCommand(command string) string {
	return "'" + command + "'"
}

func TestRenderRetryTriggered(t *testing.T) {
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = false })

	var buf bytes.Buffer
	renderRetryTriggered(&buf, &api.Push{ID: "push-123", Status: "queued"})

	want := "[✓] Push - Retry triggered for push-123\n    • Status: queued\n"
	if got := buf.String(); got != want {
		t.Fatalf("retry output = %q, want %q", got, want)
	}
}

func TestRenderPushPaginationNote(t *testing.T) {
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = false })

	var buf bytes.Buffer
	renderPushPaginationNote(&buf)

	want := "[!] Push - More results are available\n    ! Pagination is not available in this CLI version.\n"
	if got := buf.String(); got != want {
		t.Fatalf("pagination output = %q, want %q", got, want)
	}
}
