package sse

import (
	"fmt"
	"io"

	"github.com/groundsgg/grounds-cli/internal/render"
)

// Render writes a single event as a coloured line. Caller passes the
// destination (typically os.Stdout). Returns true when the event marks
// a terminal status ("ready" or "failed") so the caller can exit.
func Render(w io.Writer, ev *Event) (terminal bool) {
	switch ev.Status {
	case "ready":
		fmt.Fprintln(w, render.Green("✔ ready"))
		if url, ok := ev.Extra["publicUrl"].(string); ok && url != "" {
			fmt.Fprintln(w, "  "+render.Bold(url))
		}
		return true
	case "deploy_failed", "build_failed":
		fmt.Fprintln(w, render.Red("✗ "+ev.Status))
		if r, ok := ev.Extra["failureReason"].(string); ok && r != "" {
			fmt.Fprintln(w, "  "+r)
		}
		return true
	case "":
		if ev.Message != "" {
			fmt.Fprintln(w, ev.Message)
		}
	default:
		fmt.Fprintf(w, "[%s]\n", ev.Status)
	}
	return false
}
