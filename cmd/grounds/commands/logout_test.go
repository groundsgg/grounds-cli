package commands

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
)

func TestLogoutOutput(t *testing.T) {
	previous := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = previous })

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
