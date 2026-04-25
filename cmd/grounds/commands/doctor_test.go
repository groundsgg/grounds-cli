package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorRuns(t *testing.T) {
	cmd := NewDoctorCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	// Doctor returns a non-nil error if checks fail; that's fine for the
	// test — we just verify it ran and produced output.
	_ = cmd.Execute()
	out := buf.String()
	for _, want := range []string{"config", "auth", "api", "gradle", "java"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}
