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
		Use:     "up [--profile=minigame|platform] [--bundle=<ref> [--override=<file>]]",
		Short:   "Spawn or resume the workspace",
		Example: "  grounds cluster up\n  grounds cluster up --profile=platform\n  grounds cluster up --bundle=0.4.0 --override=./overrides/me.yaml",
		Long: `Create the workspace if it doesn't exist, or resume it from a paused state.

Profiles:
  minigame  — namespace-scoped sandbox (default). Cheap, fast, isolated by RBAC.
  platform  — full per-developer vCluster with the Grounds platform chart
              installed inside. Heavier (one-time ~90s spawn) but lets you
              run platform plugins / agones / mc-router locally.

Bundle mode (` + "`--bundle`" + `):
  Drives a platform-test environment from a versioned bundle in
  groundsgg/library-platform-bundle. Forge spins up your vCluster,
  fetches bundle.yaml @ <ref>, applies the optional override file, and
  helm-installs each component. Implies profile=platform-bundle.

  Examples:
    grounds cluster up --bundle=0.4.0
    grounds cluster up --bundle=0.4.0 --override=./overrides/me.yaml
    grounds cluster up --override=./overrides/me.yaml   # bundle ref read from file

Profile is locked once a workspace exists. To switch, ` + "`grounds cluster delete`" + ` and re-up.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}

			if bundleRef != "" || overridePath != "" {
				if profile != "" {
					return fmt.Errorf("--profile is implicit when using --bundle/--override (always platform-bundle); drop --profile")
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

			if profile != "" && profile != "minigame" && profile != "platform" {
				return fmt.Errorf("invalid --profile %q: must be \"minigame\" or \"platform\"", profile)
			}
			s, err := c.ClusterUp(ctx, profile)
			if err != nil {
				return err
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Workspace", "Active")
			render.Status(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "workspace profile: minigame (default) or platform (vCluster)")
	cmd.Flags().StringVar(&bundleRef, "bundle", "", "PlatformBundle ref (e.g. 0.4.0, main); implies platform-bundle profile")
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
		return nil, fmt.Errorf("--bundle is required (or set `bundle: <ref>` in the override file)")
	}
	return body, nil
}

// waitForBundle polls GET /v1/cluster until the workspace reaches a terminal
// state (active or failed), printing each state transition. It tolerates a
// brief 404 right after the 202 while forge commits the workspace row.
func waitForBundle(ctx context.Context, c *api.Client, w io.Writer) (*api.ClusterStatus, error) {
	const (
		interval = 5 * time.Second
		overall  = 20 * time.Minute
		rowGrace = 30 * time.Second
	)
	startedAt := time.Now()
	deadline := startedAt.Add(overall)
	graceUntil := startedAt.Add(rowGrace)
	lastState := ""
	lastSummary := ""
	spinner := render.NewSpinnerLine(w)
	defer spinner.Clear()
	renderState := &bundleWaitRenderState{
		spinner:     spinner,
		interactive: bundleWaitInteractive(w),
	}
	for {
		s, err := c.GetCluster(ctx)
		if err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) && apiErr.StatusCode == 404 && time.Now().Before(graceUntil) {
				// workspace row not committed yet — keep waiting
			} else {
				return nil, err
			}
		} else {
			renderBundleWaitProgress(w, renderState, s, time.Since(startedAt), interval)
			lastState = renderState.lastState
			lastSummary = renderState.lastSummary
			switch s.State {
			case "active", "failed":
				spinner.Clear()
				return s, nil
			}
		}
		if time.Now().After(deadline) {
			if lastSummary != "" {
				return nil, fmt.Errorf("timed out after %s waiting for the bundle (still %q, phase: %s); check `grounds cluster status`", overall, lastState, lastSummary)
			}
			return nil, fmt.Errorf("timed out after %s waiting for the bundle (still %q); check `grounds cluster status`", overall, lastState)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
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
