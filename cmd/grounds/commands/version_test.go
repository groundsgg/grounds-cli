package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/groundsgg/grounds-cli/internal/version"
)

func TestVersionCommand(t *testing.T) {
	version.Version = "0.1.0"
	version.Commit = "abc123"
	version.BuildAt = "2026-04-25T00:00:00Z"

	cmd := NewVersionCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"0.1.0", "abc123", "2026-04-25"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}
