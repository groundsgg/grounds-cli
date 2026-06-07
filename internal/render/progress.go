package render

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/groundsgg/grounds-cli/internal/api"
	"golang.org/x/term"
)

var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type SpinnerLine struct {
	w           io.Writer
	frame       int
	shown       bool
	interactive bool
	width       int
}

func NewSpinnerLine(w io.Writer) *SpinnerLine {
	file, ok := w.(*os.File)
	if !ok {
		return NewSpinnerLineForTerminal(w, false, 0)
	}
	fd := int(file.Fd())
	if !term.IsTerminal(fd) {
		return NewSpinnerLineForTerminal(w, false, 0)
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		width = 80
	}
	return NewSpinnerLineForTerminal(w, true, width)
}

func NewSpinnerLineForTerminal(w io.Writer, interactive bool, width int) *SpinnerLine {
	if width <= 0 {
		width = 120
	}
	return &SpinnerLine{w: w, interactive: interactive, width: width}
}

func (s *SpinnerLine) Update(summary string, elapsed, nextPoll time.Duration) {
	if !s.interactive {
		return
	}
	if s.shown {
		s.Clear()
	}
	frame := SpinnerFrames[s.frame%len(SpinnerFrames)]
	s.frame++
	suffix := fmt.Sprintf(" (elapsed %s, next check in %s)", formatClock(elapsed), formatSeconds(nextPoll))
	contentWidth := s.width - visibleWidth("    ")
	if contentWidth <= 0 {
		contentWidth = 1
	}
	line := truncateForWidth(fmt.Sprintf("%s %s%s", frame, summary, suffix), contentWidth)
	rest := ""
	lineRunes := []rune(line)
	if len(lineRunes) > 1 {
		rest = string(lineRunes[1:])
	}
	fmt.Fprintf(
		s.w,
		"    %s%s\n",
		Yellow(string(lineRunes[0])),
		rest,
	)
	s.shown = true
}

func (s *SpinnerLine) Clear() {
	if !s.interactive || !s.shown {
		return
	}
	fmt.Fprint(s.w, "\033[1A\r\033[K")
	s.shown = false
}

func BundleProgressSummary(progress *api.BundleProgress) string {
	if progress == nil {
		return ""
	}
	label := phaseLabel(progress.Phase)
	if progress.ComponentsTotal > 0 {
		label = fmt.Sprintf("%s %d/%d", label, progress.ComponentsDone, progress.ComponentsTotal)
	}
	if progress.CurrentComponent != "" {
		label += ": " + progress.CurrentComponent
	}
	parts := make([]string, 0, 2)
	if progress.CurrentComponentType != "" {
		parts = append(parts, progress.CurrentComponentType)
	}
	if progress.CurrentComponentMode != "" {
		parts = append(parts, progress.CurrentComponentMode)
	}
	if len(parts) > 0 {
		label += " (" + strings.Join(parts, ", ") + ")"
	}
	return label
}

func phaseLabel(phase string) string {
	switch phase {
	case "initializing":
		return "preparing bundle workspace"
	case "ensuring_namespace":
		return "ensuring namespace"
	case "installing_vcluster":
		return "installing vCluster"
	case "waiting_for_vcluster":
		return "waiting for vCluster API"
	case "provisioning_pull_secret":
		return "provisioning pull secret"
	case "provisioning_forwarding_secret":
		return "provisioning forwarding secret"
	case "installing_nats":
		return "installing shared NATS"
	case "installing_postgres":
		return "installing shared Postgres"
	case "loading_bundle":
		return "loading bundle"
	case "deploying_components":
		return "deploying components"
	case "finalizing":
		return "finalizing"
	case "active":
		return "bundle provisioning completed"
	case "failed":
		return "bundle provisioning failed"
	default:
		return strings.ReplaceAll(phase, "_", " ")
	}
}

func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Round(time.Second).Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatSeconds(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
}

func truncateForWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if visibleWidth(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	runes := []rune(s)
	for visibleWidth(string(runes)) > width-3 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

func visibleWidth(s string) int {
	return len([]rune(s))
}
