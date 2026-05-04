package commands

import (
	"bytes"
	"os"
	"testing"
)

func TestInit_NonInteractive(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	cmd := NewInitCommand()
	cmd.SetArgs([]string{"--app-name=my-arena", "--type=gamemode", "--base-image=paper"})
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
	if got := buf.String(); got != "[✓] Init - Wrote grounds.yaml\n    • Next: run `grounds push`.\n" {
		t.Fatalf("output = %q", got)
	}
}
