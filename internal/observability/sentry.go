// Package observability wires the CLI into GlitchTip (Sentry-protocol
// error tracker) at glitch.grounds.gg.
//
// Activation: the DSN is baked into release binaries via -ldflags
// in the Makefile. Local `go build` invocations leave DSN empty and
// the package no-ops, so dev work doesn't pollute the prod project.
//
// Override at runtime: GROUNDS_SENTRY_DSN env var wins over the
// build-time default (used by maintainers to point a dev build at
// a personal GlitchTip).
//
// Disable: GROUNDS_SENTRY_DISABLE=1 (used in CI smoke tests so we
// don't ship synthetic events to prod).
//
// What we capture:
//   - panics in the root cobra command (via Recover)
//   - errors returned from the cobra Execute() call
//   - explicit CaptureException() calls from elsewhere in the CLI
//
// We tag every event with version, commit, and the cobra subcommand
// path so the GlitchTip "Issues" view groups failures by command.
package observability

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/groundsgg/grounds-cli/internal/version"
)

// DSN is overridden at build time via -ldflags. Leaving empty in
// source so `go run` / `go test` / un-released builds don't ship
// telemetry.
var DSN string

const flushTimeout = 3 * time.Second

// Init wires Sentry up. Returns nil if disabled (no DSN, or
// GROUNDS_SENTRY_DISABLE=1). Safe to call multiple times — only the
// first wins.
func Init() error {
	if os.Getenv("GROUNDS_SENTRY_DISABLE") == "1" {
		return nil
	}
	dsn := os.Getenv("GROUNDS_SENTRY_DSN")
	if dsn == "" {
		dsn = DSN
	}
	if dsn == "" {
		return nil
	}
	return sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Release:          version.Version,
		Environment:      detectEnvironment(),
		AttachStacktrace: true,
		// CLI events are low-volume; sample everything.
		SampleRate: 1.0,
		// Tracing off for now — CLI command runs are short and we
		// don't get useful spans from them. Enable later if we
		// instrument forge calls.
		EnableTracing: false,
		// Strip the user's home dir from filepath frames in stack
		// traces so the events don't leak local paths to the
		// shared GlitchTip project.
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			redactHomePaths(event)
			return event
		},
	})
}

// Flush blocks up to flushTimeout for in-flight events to land in
// GlitchTip. Call as the final defer in main().
func Flush() {
	sentry.Flush(flushTimeout)
}

// Recover captures panics. Wire this as a deferred call alongside
// Flush — sentry.Recover re-panics so the runtime crash is preserved
// after the event has been queued.
func Recover() {
	if r := recover(); r != nil {
		sentry.CurrentHub().Recover(r)
		panic(r)
	}
}

// CaptureExecuteError is called from main() with the error returned
// by cobra's root Execute. It does NOT capture cobra's "user error"
// class (missing/invalid args, unknown subcommand) — those are
// expected outcomes of a user-invocation and would create issue
// noise. We sniff the error text for cobra's prefixes; not perfect
// but covers the common cases.
func CaptureExecuteError(err error, commandPath string) {
	if err == nil {
		return
	}
	if isUserError(err) {
		return
	}
	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		if commandPath != "" {
			scope.SetTag("command", commandPath)
		}
		scope.SetTag("commit", version.Commit)
		scope.SetTag("os", currentOS())
		scope.SetLevel(sentry.LevelError)
		hub.CaptureException(err)
	})
}

// CaptureException is exported for ad-hoc capture from elsewhere in
// the CLI (e.g. forge HTTP errors). The optional `op` arg ends up as
// a tag for grouping.
func CaptureException(err error, op string) {
	if err == nil {
		return
	}
	hub := sentry.CurrentHub()
	hub.WithScope(func(scope *sentry.Scope) {
		if op != "" {
			scope.SetTag("op", op)
		}
		scope.SetTag("commit", version.Commit)
		hub.CaptureException(err)
	})
}

// isUserError tries to identify cobra's user-error class so we don't
// noise GlitchTip with wrong-arg messages. Heuristic: cobra's user
// errors carry "unknown command", "required flag(s)", or "accepts at
// most/least N arg(s)" prefixes.
func isUserError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, p := range []string{
		"unknown command",
		"unknown flag",
		"required flag",
		"accepts at most",
		"accepts at least",
		"accepts between",
		"requires at least",
		"requires at most",
		"invalid argument",
	} {
		if strings.Contains(msg, p) {
			return true
		}
	}
	// Wrapped: walk the chain.
	if next := errors.Unwrap(err); next != nil {
		return isUserError(next)
	}
	return false
}

func detectEnvironment() string {
	switch {
	case os.Getenv("CI") == "true":
		return "ci"
	case version.Version == "dev" || strings.Contains(version.Version, "dirty"):
		return "dev"
	default:
		return "production"
	}
}

func currentOS() string {
	if runtime := os.Getenv("GOOS"); runtime != "" {
		return runtime
	}
	// On the running binary we get the actual OS via runtime.GOOS,
	// but we don't import runtime here to keep this file dep-light;
	// callers usually only care about the Sentry-side `os` context
	// which sentry-go fills in automatically. This tag is a cheap
	// best-effort.
	return ""
}

// redactHomePaths walks the Sentry event's stack frames and replaces
// the user's home directory prefix with "$HOME" so personal paths
// don't leak. Only runs for events that have stacktraces (i.e. our
// captured errors and panics).
func redactHomePaths(event *sentry.Event) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	for _, ex := range event.Exception {
		if ex.Stacktrace == nil {
			continue
		}
		for i := range ex.Stacktrace.Frames {
			f := &ex.Stacktrace.Frames[i]
			f.Filename = strings.Replace(f.Filename, home, "$HOME", 1)
			f.AbsPath = strings.Replace(f.AbsPath, home, "$HOME", 1)
		}
	}
}
