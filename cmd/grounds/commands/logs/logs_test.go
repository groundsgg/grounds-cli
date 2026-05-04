package logs

import (
	"strings"
	"testing"
)

func TestLogsExamplesIncludeRequiredPushID(t *testing.T) {
	cmd := NewLogsCommand()

	for _, example := range []string{
		"grounds logs <pushId>",
		"grounds logs <pushId> --follow",
		"grounds logs deployment <name>",
	} {
		if !strings.Contains(cmd.Example, example) {
			t.Fatalf("logs examples = %q, want %q", cmd.Example, example)
		}
	}

	if strings.Contains(cmd.Example, "grounds logs\n") || strings.Contains(cmd.Example, "grounds logs --follow") {
		t.Fatalf("logs examples = %q, should include required <pushId>", cmd.Example)
	}
}
