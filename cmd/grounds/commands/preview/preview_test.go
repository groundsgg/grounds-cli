package preview

import (
	"strings"
	"testing"
	"time"
)

func TestNewPreviewCommandHasFourSubcommands(t *testing.T) {
	cmd := NewPreviewCommand()
	got := map[string]bool{}
	for _, c := range cmd.Commands() {
		got[c.Name()] = true
	}
	for _, name := range []string{"list", "show", "pin", "unpin"} {
		if !got[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestShowRequiresExactlyOneArg(t *testing.T) {
	cmd := newShow()
	cmd.SetArgs([]string{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for 0 args, got nil")
	}
}

func TestPinUseLineDiffersFromUnpin(t *testing.T) {
	pin := newPin(true)
	unpin := newPin(false)
	if !strings.HasPrefix(pin.Use, "pin") {
		t.Errorf("expected pin Use to start with 'pin', got %q", pin.Use)
	}
	if !strings.HasPrefix(unpin.Use, "unpin") {
		t.Errorf("expected unpin Use to start with 'unpin', got %q", unpin.Use)
	}
}

func TestShortIDTruncatesAt8Chars(t *testing.T) {
	cases := map[string]string{
		"abc":                              "abc",
		"abcdefgh":                         "abcdefgh",
		"abcdefghijkl":                     "abcdefgh",
		"1456b204-569c-4403-a648-b1b1401f": "1456b204",
	}
	for in, want := range cases {
		if got := shortID(in); got != want {
			t.Errorf("shortID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatTimeNilGivesDash(t *testing.T) {
	if got := formatTime(nil); got != "—" {
		t.Errorf("formatTime(nil) = %q, want '—'", got)
	}
	now := time.Now()
	got := formatTime(&now)
	if got == "—" || got == "" {
		t.Errorf("formatTime(now) returned %q, expected RFC3339 timestamp", got)
	}
}
