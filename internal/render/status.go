package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/groundsgg/grounds-cli/internal/api"
)

// Status renders a ClusterStatus as a coloured 2-column table.
func Status(w io.Writer, s *api.ClusterStatus) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleRounded)

	state := s.State
	switch s.State {
	case "active":
		state = Green(state)
	case "paused":
		state = Yellow(state)
	case "deleted":
		state = Red(state)
	}

	t.AppendRow(table.Row{"namespace", s.Namespace})
	t.AppendRow(table.Row{"state", state})
	t.AppendRow(table.Row{"profile", s.Profile})

	switch s.State {
	case "active":
		if s.AutoPauseAt != nil {
			t.AppendRow(table.Row{"auto-pause at", fmtUTC(s.AutoPauseAt)})
		}
	case "paused":
		if s.AutoDeleteAt != nil {
			t.AppendRow(table.Row{"auto-delete at", fmtUTC(s.AutoDeleteAt)})
		}
	}

	t.AppendRow(table.Row{"deployments", fmt.Sprintf("%d ready", s.DeploymentsReady)})

	if h := s.BundleHealth; h != nil {
		switch h.Status {
		case "degraded":
			t.AppendRow(table.Row{"bundle", Red(fmt.Sprintf("degraded — %d missing", len(h.Missing)))})
		case "unreachable":
			t.AppendRow(table.Row{"bundle", Yellow("unreachable")})
		}
	}

	if s.Quota != nil {
		cpu, mem, storage := s.Quota["cpu"], s.Quota["memory"], s.Quota["storage"]
		t.AppendRow(table.Row{"quota", fmt.Sprintf("%s CPU / %s / %s", cpu, mem, storage)})
	}
	t.Render()

	if s.State == "paused" {
		fmt.Fprintln(w)
		StatusLine(w, StatusWarn, "Workspace", "Paused")
		DetailLine(w, StatusWarn, "Next push or "+Command("grounds cluster up")+" resumes it.")
	}

	// The workspace says `active`, but components forge installed are gone. This
	// is what an emptied vCluster looks like — previously it just showed up as
	// "0 deployments" and nothing said anything was wrong.
	if h := s.BundleHealth; h != nil && h.Status == "degraded" {
		fmt.Fprintln(w)
		StatusLine(w, StatusError, "Workspace", "Bundle components missing")
		DetailLine(w, StatusError, strings.Join(h.Missing, ", "))
		DetailLine(w, StatusError, "The workspace is running but its bundle base is gone. "+
			Command("grounds cluster reset")+" rebuilds it.")
	}
}

func fmtUTC(t *time.Time) string {
	if t == nil {
		return ""
	}
	d := time.Until(*t)
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%s (%s)", t.UTC().Format("2006-01-02 15:04 MST"), shortDuration(d))
}

func shortDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) - 60*h
	switch {
	case h == 0:
		return fmt.Sprintf("%dm", m)
	case h < 24:
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	}
}
