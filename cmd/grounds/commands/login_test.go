package commands

import (
	"bytes"
	"errors"
	"testing"

	"github.com/fatih/color"

	"github.com/groundsgg/grounds-cli/internal/auth"
)

func TestLoginSubject(t *testing.T) {
	tests := []struct {
		name      string
		preferred string
		email     string
		want      string
	}{
		{
			name:      "preferred username",
			preferred: "player-one",
			email:     "player@example.com",
			want:      "player-one",
		},
		{
			name:  "email",
			email: "player@example.com",
			want:  "player@example.com",
		},
		{
			name: "current user",
			want: "current user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := loginSubject(tt.preferred, tt.email); got != tt.want {
				t.Fatalf("loginSubject(%q, %q) = %q, want %q", tt.preferred, tt.email, got, tt.want)
			}
		})
	}
}

func TestPrintDeviceLoginInstructionsOpenedBrowser(t *testing.T) {
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = false })

	var buf bytes.Buffer
	printDeviceLoginInstructions(&buf, &auth.DeviceCodeResponse{
		UserCode:        "ABCD-EFGH",
		VerificationURI: "https://example.test/device",
	}, nil)

	want := "[✓] Browser - Opened device login page\n" +
		"    • Code: ABCD-EFGH\n"
	if got := buf.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPrintDeviceLoginInstructionsBrowserOpenFailed(t *testing.T) {
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = false })

	var buf bytes.Buffer
	printDeviceLoginInstructions(&buf, &auth.DeviceCodeResponse{
		UserCode:        "ABCD-EFGH",
		VerificationURI: "https://example.test/device",
	}, errors.New("no opener"))

	want := "[!] Browser - Could not open device login page automatically\n" +
		"    ! URL: https://example.test/device\n" +
		"    ! Code: ABCD-EFGH\n"
	if got := buf.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
