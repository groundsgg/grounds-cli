package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/render"
)

func newUp() *cobra.Command {
	var profile string
	var bundleRef string
	var overridePath string
	cmd := &cobra.Command{
		Use:     "up [--bundle=<ref>] [--override=<file>] [--profile=minigame|platform]",
		Short:   "Spawn or resume the workspace",
		Example: "  grounds cluster up\n  grounds cluster up --bundle=0.6.4\n  grounds cluster up --profile=minigame",
		Long: `Create the workspace if it doesn't exist, or resume it from a paused state.

By default ` + "`up`" + ` provisions a **platform-bundle** workspace: a per-developer
vCluster driven by a versioned bundle in groundsgg/library-platform-bundle
(velocity proxy + lobby + grpc-services + nats/postgres/agones). Forge spins
up the vCluster, fetches bundle.yaml @ <ref> (default ` + "`main`" + `), applies the
optional override file, and helm-installs each component.

  grounds cluster up                       # platform-bundle @ main
  grounds cluster up --bundle=0.6.4         # pin a bundle release
  grounds cluster up --override=./me.yaml   # local overrides (ref from file, else main)

Legacy profiles (opt in with ` + "`--profile`" + `):
  minigame  — namespace-scoped sandbox. Cheap, fast, isolated by RBAC.
  platform  — bare per-developer vCluster with the Grounds platform chart,
              without the bundle components.

Profile is locked once a workspace exists. To switch, ` + "`grounds cluster delete`" + ` and
re-up. To wipe a platform-bundle workspace back to a clean bundle base without
changing profile, use ` + "`grounds cluster reset`" + `.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}

			// platform-bundle is the default: a bare `up`, `--bundle`, or
			// `--override` all drive bundle mode. Opt out to the legacy
			// sandbox / vCluster profiles with --profile=minigame|platform.
			if profile == "" || profile == "platform-bundle" || bundleRef != "" || overridePath != "" {
				if profile != "" && profile != "platform-bundle" {
					return fmt.Errorf("--profile=%s can't be combined with --bundle/--override (those imply platform-bundle); drop --profile, or use --profile=minigame|platform for the legacy profiles", profile)
				}
				body, err := loadBundleRequest(bundleRef, overridePath)
				if err != nil {
					return err
				}
				w := cmd.OutOrStdout()
				// forge runs the reconcile in the background and returns 202;
				// poll GET /v1/cluster until it reaches a terminal state.
				if _, err := c.ClusterUpBundle(ctx, body); err != nil {
					return err
				}
				render.StatusLine(w, render.StatusOK, "Workspace",
					fmt.Sprintf("provisioning bundle %s — this takes a few minutes…", body.Bundle))
				final, err := waitForBundle(ctx, c, w)
				if err != nil {
					return err
				}
				renderBundleStatus(w, final)
				if final.State == "failed" {
					return fmt.Errorf("bundle provisioning failed")
				}
				return nil
			}

			if profile != "minigame" && profile != "platform" {
				return fmt.Errorf("invalid --profile %q: must be \"minigame\" or \"platform\" (omit --profile for the default platform-bundle)", profile)
			}
			w := cmd.OutOrStdout()
			if _, err := c.ClusterUp(ctx, profile); err != nil {
				return err
			}
			// forge enqueues the provision and a worker runs it, so the row
			// comes back `creating`. Poll to a terminal state instead of
			// claiming "Active" the moment the request returns.
			render.StatusLine(w, render.StatusOK, "Workspace",
				"provisioning — this takes a few minutes…")
			final, err := waitForBundle(ctx, c, w)
			if err != nil {
				return err
			}
			render.Status(w, final)
			if final.State == "failed" {
				return fmt.Errorf("workspace provisioning failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "legacy profile: minigame (sandbox) or platform (bare vCluster); default is platform-bundle")
	cmd.Flags().StringVar(&bundleRef, "bundle", "", "PlatformBundle ref (e.g. 0.6.4, main); default main")
	cmd.Flags().StringVar(&overridePath, "override", "", "path to an Engineer-Override-File (YAML); implies platform-bundle profile")
	return cmd
}

// loadBundleRequest builds the POST /v1/cluster/bundle body from
// --bundle and/or --override. If --override is set, the file is parsed
// for its `overrides:` map (and `bundle:` if --bundle wasn't given).
// --bundle wins over the override-file's bundle field when both are set.
func loadBundleRequest(bundleRef, overridePath string) (*api.BundleUpRequest, error) {
	body := &api.BundleUpRequest{Bundle: bundleRef}
	if overridePath != "" {
		raw, err := os.ReadFile(overridePath)
		if err != nil {
			return nil, fmt.Errorf("reading override file: %w", err)
		}
		var parsed struct {
			Bundle    string         `yaml:"bundle"`
			Overrides map[string]any `yaml:"overrides"`
		}
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("parsing override file: %w", err)
		}
		if body.Bundle == "" {
			body.Bundle = parsed.Bundle
		}
		body.Overrides = parsed.Overrides
	}
	if body.Bundle == "" {
		// platform-bundle is the default profile; with no ref given on the flag
		// or in the override file, track the bundle's main branch.
		body.Bundle = "main"
	}
	return body, nil
}

// waitForBundle polls GET /v1/cluster until the workspace reaches a terminal
// state (active or failed), printing each state transition. It tolerates a
// brief 404 right after the 202 while forge commits the workspace row.
func waitForBundle(ctx context.Context, c *api.Client, w io.Writer) (*api.ClusterStatus, error) {
	return waitForBundleWithOptions(ctx, c.GetCluster, w, bundleWaitOptions{})
}

type bundleStatusGetter func(context.Context) (*api.ClusterStatus, error)

type bundleWaitOptions struct {
	pollInterval    time.Duration
	spinnerInterval time.Duration
	overall         time.Duration
	rowGrace        time.Duration
	interactive     *bool
	spinner         *render.SpinnerLine
}

// waitForBundleWithOptions polls Forge at a coarse interval but refreshes the
// interactive spinner between polls so the terminal does not look stalled.
func waitForBundleWithOptions(ctx context.Context, getCluster bundleStatusGetter, w io.Writer, opts bundleWaitOptions) (*api.ClusterStatus, error) {
	const (
		defaultPollInterval    = 5 * time.Second
		defaultSpinnerInterval = 100 * time.Millisecond
		defaultOverall         = 20 * time.Minute
		defaultRowGrace        = 30 * time.Second
	)
	interval := durationOrDefault(opts.pollInterval, defaultPollInterval)
	spinnerInterval := durationOrDefault(opts.spinnerInterval, defaultSpinnerInterval)
	overall := durationOrDefault(opts.overall, defaultOverall)
	rowGrace := durationOrDefault(opts.rowGrace, defaultRowGrace)
	startedAt := time.Now()
	deadline := startedAt.Add(overall)
	graceUntil := startedAt.Add(rowGrace)
	lastState := ""
	lastSummary := ""
	spinner := opts.spinner
	if spinner == nil {
		spinner = render.NewSpinnerLine(w)
	}
	defer spinner.Clear()
	interactive := bundleWaitInteractive(w)
	if opts.interactive != nil {
		interactive = *opts.interactive
	}
	renderState := &bundleWaitRenderState{
		spinner:     spinner,
		interactive: interactive,
	}
	var lastStatus *api.ClusterStatus
	nextPollAt := startedAt
	for {
		now := time.Now()
		if !nextPollAt.After(now) {
			s, err := getCluster(ctx)
			if err != nil {
				var apiErr *api.Error
				if errors.As(err, &apiErr) && apiErr.StatusCode == 404 && now.Before(graceUntil) {
					// workspace row not committed yet — keep waiting
				} else {
					return nil, err
				}
			} else {
				lastStatus = s
				renderBundleWaitProgress(w, renderState, s, time.Since(startedAt), interval)
				lastState = renderState.lastState
				lastSummary = renderState.lastSummary
				switch s.State {
				case "active", "failed":
					spinner.Clear()
					return s, nil
				}
			}
			nextPollAt = time.Now().Add(interval)
			continue
		}
		if now.After(deadline) {
			if lastSummary != "" {
				return nil, fmt.Errorf("timed out after %s waiting for the bundle (still %q, phase: %s); check `grounds cluster status`", overall, lastState, lastSummary)
			}
			return nil, fmt.Errorf("timed out after %s waiting for the bundle (still %q); check `grounds cluster status`", overall, lastState)
		}
		wait := time.Until(nextPollAt)
		if renderState.interactive && lastStatus != nil && wait > spinnerInterval {
			wait = spinnerInterval
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
			if renderState.interactive && lastStatus != nil && time.Now().Before(nextPollAt) {
				renderBundleWaitProgress(w, renderState, lastStatus, time.Since(startedAt), time.Until(nextPollAt))
			}
		}
	}
}

func durationOrDefault(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

type bundleWaitRenderState struct {
	spinner     *render.SpinnerLine
	interactive bool
	lastState   string
	lastSummary string
}

func renderBundleWaitProgress(w io.Writer, state *bundleWaitRenderState, s *api.ClusterStatus, elapsed, nextPoll time.Duration) {
	if state.spinner == nil {
		state.spinner = render.NewSpinnerLine(w)
	}
	summary := render.BundleProgressSummary(s.BundleProgress)
	if summary != "" {
		line := "phase: " + summary
		if state.interactive {
			state.spinner.Update(line, elapsed, nextPoll)
		} else if summary != state.lastSummary {
			render.DetailLine(w, render.StatusOK, line)
		}
		state.lastState = s.State
		state.lastSummary = summary
		return
	}

	line := "state: " + s.State
	if state.interactive {
		state.spinner.Update(line, elapsed, nextPoll)
	} else if s.State != state.lastState {
		render.DetailLine(w, render.StatusOK, line)
	}
	state.lastState = s.State
}

func bundleWaitInteractive(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func renderBundleStatus(w io.Writer, s *api.ClusterStatus) {
	if s.State == "failed" {
		render.StatusLine(w, render.StatusError, "Workspace", "bundle provisioning failed")
		if s.FailureReason != "" {
			render.DetailLine(w, render.StatusError, s.FailureReason)
		}
		return
	}
	status := render.StatusOK
	nFailed := 0
	if s.BundleResult != nil {
		nFailed = len(s.BundleResult.Failed)
	}
	if nFailed > 0 {
		status = render.StatusWarn
	}
	render.StatusLine(w, status, "Workspace", fmt.Sprintf("active in namespace %s", s.Namespace))
	if s.BundleResult != nil {
		render.DetailLine(w, status, fmt.Sprintf("Components: %d succeeded, %d failed",
			len(s.BundleResult.Succeeded), nFailed))
		for _, f := range s.BundleResult.Failed {
			render.DetailLine(w, render.StatusError, fmt.Sprintf("%s: %s", f.Name, f.Error))
		}
	}
}
