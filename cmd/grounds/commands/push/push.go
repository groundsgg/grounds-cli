package push

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/internal/projectscope"
	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/gradle"
	"github.com/groundsgg/grounds-cli/internal/minestom"
	"github.com/groundsgg/grounds-cli/internal/render"
	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
)

var (
	findGradleWrapper = gradle.FindWrapper
	runGradleWrapper  = gradle.Run
	runGradleWithEnv  = gradle.RunWithEnv
	newAPIClient      = api.New
)

func NewPushCommand() *cobra.Command {
	cmd := newPush()
	cmd.Example = "  grounds push\n  grounds push --target=staging\n  grounds push list --mine"
	cmd.AddCommand(newRetry(), newList())
	return cmd
}

func newPush() *cobra.Command {
	var target string
	var flavor string
	var force bool
	var local []string
	var withLocal bool
	cmd := &cobra.Command{
		Use:     "push [--target=dev|staging] [--flavor=<key>] [--force] [--local=<id>[,<id>]] [--with-local]",
		Short:   "Build via Gradle plugin and deploy to a target",
		Example: "  grounds push\n  grounds push --flavor=velocity\n  grounds push --target=staging\n  grounds push --force\n  grounds push --local=plugin-chat\n  grounds push --with-local",
		Long: `Build the current project with the grounds-push Gradle plugin and deploy it.

Targets:
  dev     — long-lived, lands in your personal namespace (user-<handle>).
  staging — ephemeral preview env, fresh namespace per push, auto-deleted after 7 days.
            Public URL pattern: <name>-pr<id>.dev.grnds.io.

--force re-runs the build pipeline even when forge's contentHash dedup
would have reused an existing image. Useful when the upstream base
image moved under a stable tag, or to re-observe the build flow.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if target != "dev" && target != "staging" {
				return fmt.Errorf("invalid --target %q: must be \"dev\" or \"staging\"", target)
			}
			flavor = strings.TrimSpace(flavor)
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			wrapper, err := findGradleWrapper(cwd)
			if err != nil {
				return fmt.Errorf("%w\n    ! Not a Gradle project? Run %s to scaffold, or cd to your project root.", err, render.Command("grounds init"))
			}
			ctx := context.Background()

			// Pre-refresh: the Gradle plugin reads credentials.json
			// directly via CredentialResolver and rejects expired
			// access tokens (Keycloak default lifetime is ~5min).
			// Force a refresh here so the file is fresh before
			// Gradle reads it. Skip when GROUNDS_TOKEN is set —
			// the plugin uses the env var directly, no file involved.
			if os.Getenv("GROUNDS_TOKEN") == "" {
				cfg, err := config.Load("")
				if err != nil {
					return err
				}
				src := &auth.FileTokenSource{
					Store:  auth.NewStore(cfg.Dir),
					Device: defaultDevice(),
				}
				if _, err := src.Token(ctx); err != nil {
					return authRefreshError(err)
				}
			}

			manifestPath := filepath.Join(filepath.Dir(wrapper), "grounds.yaml")
			pushManifest, err := minestom.LoadPushManifest(manifestPath, flavor)
			if err == nil && pushManifest.IsMinestomServer() {
				return runMinestomPush(ctx, cmd, wrapper, pushManifest, target, flavor, force, local, withLocal)
			}
			if err != nil && (flavor == "minestom" || minestom.IsRuntimeValidationError(err)) {
				return err
			}
			return runGradlePush(ctx, cmd, wrapper, target, flavor, force, local, withLocal)
		},
	}
	cmd.Flags().StringVar(&target, "target", "dev", "deploy target: dev (persistent personal ns) or staging (ephemeral preview env, 7d TTL)")
	_ = cmd.RegisterFlagCompletionFunc("target", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "staging"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&flavor, "flavor", "", "app flavor from grounds.yaml flavors (for example paper or velocity)")
	_ = cmd.RegisterFlagCompletionFunc("flavor", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&force, "force", false, "skip contentHash dedup and force a fresh build")
	cmd.Flags().StringArrayVar(&local, "local", nil, "use local workspace override for plugin id (repeatable, comma-separated)")
	cmd.Flags().BoolVar(&withLocal, "with-local", false, "use all enabled local workspace overrides present in grounds.yaml")
	return cmd
}

func runGradlePush(ctx context.Context, cmd *cobra.Command, wrapper, target, flavor string, force bool, local []string, withLocal bool) error {
	args := []string{"groundsPush", "--target=" + target}
	if flavor != "" {
		args = append(args, "--flavor="+flavor)
	}
	if force {
		args = append(args, "--force")
	}
	if withLocal || len(internalworkspace.NormalizeLocalIDs(local)) > 0 {
		workspaceConfig, err := internalworkspace.Load("")
		if err != nil {
			return err
		}
		manifestPath := filepath.Join(filepath.Dir(wrapper), "grounds.yaml")
		plan, err := internalworkspace.Resolve(ctx, manifestPath, workspaceConfig, internalworkspace.ResolveOptions{
			LocalIDs:  local,
			WithLocal: withLocal,
			Flavor:    flavor,
			Stdout:    cmd.OutOrStdout(),
			Stderr:    cmd.ErrOrStderr(),
		})
		if err != nil {
			return err
		}
		renderBundleSources(cmd.OutOrStdout(), plan)
		file, err := os.CreateTemp("", "grounds-resolved-plugins-*.json")
		if err != nil {
			return err
		}
		resolvedPath := file.Name()
		if err := file.Close(); err != nil {
			return err
		}
		defer os.Remove(resolvedPath)
		if err := internalworkspace.WritePlanFile(resolvedPath, plan); err != nil {
			return err
		}
		args = append(args, "--resolved-plugins-file="+resolvedPath)
	}
	_, _, selected, err := projectscope.BuildClient(ctx, cmd)
	if err != nil {
		return err
	}
	return runGradleWithEnv(ctx, wrapper, args, []string{"GROUNDS_PROJECT=" + selected.ID}, cmd.OutOrStdout(), cmd.ErrOrStderr(), 0)
}

func runMinestomPush(ctx context.Context, cmd *cobra.Command, wrapper string, pushManifest *minestom.PushManifest, target, flavor string, force bool, local []string, withLocal bool) error {
	projectRoot := filepath.Dir(wrapper)
	hasFlavors := false
	if pushManifest.Full != nil {
		_, hasFlavors = pushManifest.Full["flavors"]
	}
	if !hasFlavors && flavor != "" && flavor != pushManifest.FlavorKey {
		return fmt.Errorf("grounds.yaml: unknown flavor %q for top-level minestom runtime", flavor)
	}

	var workspaceConfig *internalworkspace.Config
	if withLocal || len(internalworkspace.NormalizeLocalIDs(local)) > 0 {
		var err error
		workspaceConfig, err = internalworkspace.Load("")
		if err != nil {
			return err
		}
	}
	localPlan, err := minestom.ResolveLocalModules(ctx, *pushManifest, workspaceConfig, minestom.ResolveOptions{LocalIDs: local, WithLocal: withLocal})
	if err != nil {
		return err
	}
	if len(localPlan.EffectivePluginSources) > 0 {
		renderMinestomBundleSources(cmd.OutOrStdout(), localPlan)
	}

	args := []string{pushManifest.Runtime.Build.Task}
	var initScript string
	if len(localPlan.LocalModules) > 0 {
		initScript, err = minestom.WriteCompositeInitScript(localPlan)
		if err != nil {
			return err
		}
		defer os.Remove(initScript)
		args = append(args, "-I", initScript)
	}
	if err := runGradleWrapper(ctx, wrapper, args, cmd.OutOrStdout(), cmd.ErrOrStderr(), 0); err != nil {
		return err
	}
	distributionArtifact, err := minestom.ResolveDistributionArtifact(projectRoot, pushManifest.Runtime.Build.Artifact)
	if err != nil {
		return err
	}
	artifact, cleanupArtifact, err := minestom.NormalizeDistributionArtifact(distributionArtifact)
	if err != nil {
		return err
	}
	defer cleanupArtifact()

	client, _, _, err := projectscope.BuildClient(ctx, cmd)
	if err != nil {
		return err
	}
	manifestPayload := any(map[string]any{
		"name":       pushManifest.Name,
		"type":       pushManifest.Runtime.Type,
		"publicType": "minestom",
		"baseImage":  pushManifest.Runtime.BaseImage,
	})
	uploadFlavor := ""
	if pushManifest.Full != nil {
		manifestPayload = pushManifest.Full
		if hasFlavors {
			uploadFlavor = pushManifest.FlavorKey
		}
	}
	response, err := client.CreatePush(ctx, api.CreatePushRequest{
		Target:                 target,
		Flavor:                 uploadFlavor,
		Force:                  force,
		Manifest:               manifestPayload,
		EffectivePluginSources: localPlan.EffectivePluginSources,
		ArtifactPath:           artifact,
	})
	if err != nil {
		return err
	}
	render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Push", "Submitted "+response.PushID)
	render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Status: "+response.Status)
	return nil
}

func renderBundleSources(out io.Writer, plan *internalworkspace.Plan) {
	fmt.Fprintln(out, "Bundle sources:")
	rows := make([][]any, 0, len(plan.EffectivePluginSources))
	localPaths := map[string]string{}
	for _, plugin := range plan.Plugins {
		if plugin.LocalPath != "" {
			localPaths[plugin.ID+"\x00"+plugin.Variant] = plugin.LocalPath
		}
	}
	for _, source := range plan.EffectivePluginSources {
		value := source.Source
		if source.Effective == "local" {
			value = localPaths[source.ID+"\x00"+source.Variant]
		}
		rows = append(rows, []any{source.ID, source.Variant, source.Effective, value})
	}
	render.Table(out, []string{"ID", "Variant", "Effective", "Value"}, rows)
}

func renderMinestomBundleSources(out io.Writer, plan *minestom.LocalPlan) {
	workspacePlan := &internalworkspace.Plan{EffectivePluginSources: plan.EffectivePluginSources}
	for _, module := range plan.LocalModules {
		workspacePlan.Plugins = append(workspacePlan.Plugins, internalworkspace.PlanPlugin{
			ID:        module.ID,
			Variant:   module.Variant,
			LocalPath: module.Path,
		})
	}
	renderBundleSources(out, workspacePlan)
}

func authRefreshError(err error) error {
	return fmt.Errorf("auth refresh failed: %w\n    ! Run %s to re-authenticate.", err, render.Command("grounds login"))
}

// defaultDevice mirrors login.go (same issuer, same client ID).
// Lifted here so subcommands don't depend on the commands package.
func defaultDevice() *auth.DeviceClient {
	// Avoid pkg-level constants from sibling pkg; hardcode same values
	return &auth.DeviceClient{
		Issuer:   "https://account.grounds.gg/realms/grounds",
		ClientID: "grounds-cli",
		HTTP:     defaultHTTP(),
	}
}

func defaultHTTP() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}
