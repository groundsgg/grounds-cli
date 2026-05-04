package commands

import (
	"bytes"
	"testing"
)

func TestLogoutOutput(t *testing.T) {
	t.Setenv("GROUNDS_CONFIG_DIR", t.TempDir())

	cmd := NewLogoutCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := buf.String(); got != "[✓] Auth - Logged out\n" {
		t.Fatalf("output = %q", got)
	}
}
