package render

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
)

func TestStatusBadgeNoColor(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	if got := StatusBadge(StatusOK); got != "[✓]" {
		t.Fatalf("StatusBadge(StatusOK) = %q", got)
	}
	if got := StatusBadge(StatusWarn); got != "[!]" {
		t.Fatalf("StatusBadge(StatusWarn) = %q", got)
	}
	if got := StatusBadge(StatusError); got != "[✗]" {
		t.Fatalf("StatusBadge(StatusError) = %q", got)
	}
}

func TestStatusLine(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	StatusLine(&buf, StatusOK, "Init", "Wrote grounds.yaml")

	want := "[✓] Init - Wrote grounds.yaml\n"
	if got := buf.String(); got != want {
		t.Fatalf("StatusLine output = %q, want %q", got, want)
	}
}

func TestDetailLine(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	DetailLine(&buf, StatusWarn, "Run "+Command("grounds push")+" to create one.")

	want := "    ! Run `grounds push` to create one.\n"
	if got := buf.String(); got != want {
		t.Fatalf("DetailLine output = %q, want %q", got, want)
	}
}

func TestCommand(t *testing.T) {
	if got := Command("grounds version --check"); got != "`grounds version --check`" {
		t.Fatalf("Command() = %q", got)
	}
}
