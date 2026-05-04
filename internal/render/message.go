package render

import (
	"fmt"
	"io"
)

type StatusKind string

const (
	StatusOK    StatusKind = "ok"
	StatusWarn  StatusKind = "warn"
	StatusError StatusKind = "error"
)

func StatusBadge(status StatusKind) string {
	switch status {
	case StatusWarn:
		return Yellow("[!]")
	case StatusError:
		return Red("[✗]")
	default:
		return Green("[✓]")
	}
}

func DetailIcon(status StatusKind) string {
	switch status {
	case StatusError:
		return Red("✗")
	case StatusWarn:
		return Yellow("!")
	default:
		return "•"
	}
}

func StatusLine(w io.Writer, status StatusKind, subject, summary string) {
	fmt.Fprintf(w, "%s %s - %s\n", StatusBadge(status), subject, summary)
}

func DetailLine(w io.Writer, status StatusKind, detail string) {
	fmt.Fprintf(w, "    %s %s\n", DetailIcon(status), detail)
}

func Command(command string) string {
	return "`" + command + "`"
}
