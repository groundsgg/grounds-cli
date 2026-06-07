package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/groundsgg/grounds-cli/internal/api"
)

func TestBundleProgressSummaryWithComponent(t *testing.T) {
	got := BundleProgressSummary(&api.BundleProgress{
		Phase:                "deploying_components",
		CurrentComponent:     "plugin-config",
		CurrentComponentType: "grpc-service",
		CurrentComponentMode: "gradle-local",
		ComponentsTotal:      14,
		ComponentsDone:       7,
	})
	want := "deploying components 7/14: plugin-config (grpc-service, gradle-local)"
	if got != want {
		t.Fatalf("BundleProgressSummary() = %q, want %q", got, want)
	}
}

func TestBundleProgressSummaryUnknownPhase(t *testing.T) {
	got := BundleProgressSummary(&api.BundleProgress{Phase: "warming_cache"})
	if got != "warming cache" {
		t.Fatalf("BundleProgressSummary() = %q, want %q", got, "warming cache")
	}
}

func TestBundleProgressSummaryNil(t *testing.T) {
	got := BundleProgressSummary(nil)
	if got != "" {
		t.Fatalf("BundleProgressSummary(nil) = %q, want empty", got)
	}
}

func TestSpinnerLineClearAfterUpdate(t *testing.T) {
	SetEnabled(true)
	buf := &bytes.Buffer{}
	spinner := NewSpinnerLineForTerminal(buf, true, 80)

	spinner.Update("deploying components", 75*time.Second, 5*time.Second)
	spinner.Clear()

	if !strings.Contains(buf.String(), "\033[1A\r\033[K") {
		t.Fatalf("SpinnerLine output = %q, want clear sequence", buf.String())
	}
}

func TestSpinnerLineDefaultConstructorDoesNotAnimateNonTerminalWriter(t *testing.T) {
	SetEnabled(true)
	buf := &bytes.Buffer{}
	spinner := NewSpinnerLine(buf)

	spinner.Update("deploying components", 75*time.Second, 5*time.Second)
	spinner.Clear()

	if got := buf.String(); got != "" {
		t.Fatalf("SpinnerLine output = %q, want empty for non-terminal writer", got)
	}
}

func TestSpinnerLineNonInteractiveDoesNotWriteANSI(t *testing.T) {
	SetEnabled(true)
	buf := &bytes.Buffer{}
	spinner := NewSpinnerLineForTerminal(buf, false, 80)

	spinner.Update("deploying components", 75*time.Second, 5*time.Second)
	spinner.Clear()

	if strings.Contains(buf.String(), "\033[") {
		t.Fatalf("SpinnerLine output = %q, want no ANSI", buf.String())
	}
	if buf.String() != "" {
		t.Fatalf("SpinnerLine output = %q, want empty", buf.String())
	}
}

func TestSpinnerLineTruncatesToSingleTerminalRow(t *testing.T) {
	SetEnabled(true)
	buf := &bytes.Buffer{}
	spinner := NewSpinnerLineForTerminal(buf, true, 48)

	spinner.Update("deploying components 7/14: very-long-component-name-that-would-wrap", 75*time.Second, 5*time.Second)

	line := strings.TrimSuffix(buf.String(), "\n")
	if got := len([]rune(line)); got > 48 {
		t.Fatalf("SpinnerLine width = %d, want <= 48; output = %q", got, line)
	}
	if !strings.Contains(line, "...") {
		t.Fatalf("SpinnerLine output = %q, want truncation marker", line)
	}
}

func TestSpinnerLineTruncatesWholeLineForVeryNarrowTerminal(t *testing.T) {
	SetEnabled(true)
	buf := &bytes.Buffer{}
	spinner := NewSpinnerLineForTerminal(buf, true, 20)

	spinner.Update("deploying components", 75*time.Second, 5*time.Second)

	line := strings.TrimSuffix(buf.String(), "\n")
	if got := len([]rune(line)); got > 20 {
		t.Fatalf("SpinnerLine width = %d, want <= 20; output = %q", got, line)
	}
	if !strings.Contains(line, "...") {
		t.Fatalf("SpinnerLine output = %q, want truncation marker", line)
	}
}
