package cluster

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
				res, err := c.ClusterUpBundle(ctx, body)
				if err != nil {
					return err
				}
				renderBundleResult(cmd.OutOrStdout(), res)
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

func renderBundleResult(w interface {
	Write(p []byte) (int, error)
}, res *api.BundleUpResult) {
	status := render.StatusOK
	summary := fmt.Sprintf("%s with bundle %s in namespace %s", res.State, res.BundleVersion, res.Namespace)
	if len(res.Components.Failed) > 0 {
		status = render.StatusWarn
	}
	render.StatusLine(w, status, "Workspace", summary)
	render.DetailLine(w, status, fmt.Sprintf("Components: %d resolved, %d succeeded, %d failed",
		res.Components.Resolved, len(res.Components.Succeeded), len(res.Components.Failed)))
	for _, f := range res.Components.Failed {
		render.DetailLine(w, render.StatusError, fmt.Sprintf("%s: %s", f.Name, f.Error))
	}
}
