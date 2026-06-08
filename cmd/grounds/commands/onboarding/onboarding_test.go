package onboarding

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewOnboardingCommand_Metadata(t *testing.T) {
	cmd := NewOnboardingCommand()
	if cmd.Use != "onboarding" {
		t.Fatalf("Use = %q, want onboarding", cmd.Use)
	}
	wantAliases := map[string]bool{"onboard": true, "quickstart": true}
	for _, a := range cmd.Aliases {
		delete(wantAliases, a)
	}
	if len(wantAliases) != 0 {
		t.Fatalf("missing aliases: %v", wantAliases)
	}
}

// In a test process stdin is not a TTY, so the command must bail with a
// clear message instead of attempting an interactive prompt.
func TestOnboarding_NonTTYBailsCleanly(t *testing.T) {
	cmd := NewOnboardingCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error in a non-TTY environment, got nil")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("error = %q, want it to mention an interactive terminal", err.Error())
	}
}
