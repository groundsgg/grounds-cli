package push

import (
	"bytes"
	"errors"
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
