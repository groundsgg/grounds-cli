package push

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"

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
	if strings.Contains(got, "→") || strings.Contains(got, "'grounds init'") {
		t.Fatalf("error = %q, should not use arrows or single-quoted commands", got)
	}
}

func TestPushAuthRefreshErrorSuggestsLoginCommand(t *testing.T) {
	err := authRefreshError(errors.New("token expired"))

	got := err.Error()
	if !strings.Contains(got, "Run `grounds login`") {
		t.Fatalf("error = %q, want login command suggestion", got)
	}
	if strings.Contains(got, "→") || strings.Contains(got, "'grounds login'") {
		t.Fatalf("error = %q, should not use arrows or single-quoted commands", got)
	}
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
