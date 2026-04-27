package push

import (
	"bytes"
	"strings"
	"testing"
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
