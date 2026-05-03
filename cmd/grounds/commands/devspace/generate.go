package devspace

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/groundsgg/grounds-cli/internal/api"
)

func newGenerate() *cobra.Command {
	var bundleRef string
	var overridePath string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "generate <component-name>",
		Short: "Render a component-specific devspace.yaml from the bundle's workflow template",
		Long: `Asks forge to substitute the engineer's override values into the
matching devspace template (jar-sync-pod-restart or quarkus-dev) and
writes the result to ./devspace.yaml (or --output).

The component name is the release-name from bundle.yaml (e.g.
plugin-social, service-config). The workflow type is read from the
bundle.

Examples:

  # Pin against a bundle release, override file describes plugin-social.
  grounds devspace generate plugin-social --bundle 0.4.0 --override ./me.yaml

  # Override file alone — bundle ref read from its 'bundle:' field.
  grounds devspace generate plugin-social --override ./me.yaml

  # Defaults — no override, just template + bundle's component-name.
  grounds devspace generate plugin-social --bundle main`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			componentName := args[0]
			bundle, override, err := loadGenerateInputs(bundleRef, overridePath, componentName)
			if err != nil {
				return err
			}

			ctx := context.Background()
			c, _, err := buildClient(ctx, cmd)
			if err != nil {
				return err
			}

			yaml, err := c.DevspaceGenerate(ctx, &api.DevspaceGenerateRequest{
				Bundle:        bundle,
				ComponentName: componentName,
				Override:      override,
			})
			if err != nil {
				return err
			}

			if outputPath == "-" {
				cmd.OutOrStdout().Write(yaml)
				return nil
			}
			if err := os.WriteFile(outputPath, yaml, 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", outputPath, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ Wrote %d bytes to %s\n", len(yaml), outputPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&bundleRef, "bundle", "", "PlatformBundle ref (e.g. 0.4.0, main); inherits from --override file when not set")
	cmd.Flags().StringVar(&overridePath, "override", "", "path to an Engineer-Override-File (YAML)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "./devspace.yaml", "output path (use '-' for stdout)")
	return cmd
}

// loadGenerateInputs picks the bundle-ref + per-component override
// out of the override file (when present), with --bundle on the
// command line winning over the file's `bundle:` field.
func loadGenerateInputs(bundleRef, overridePath, componentName string) (string, map[string]any, error) {
	if overridePath == "" {
		if bundleRef == "" {
			return "", nil, fmt.Errorf("either --bundle or --override must be set")
		}
		return bundleRef, nil, nil
	}
	raw, err := os.ReadFile(overridePath)
	if err != nil {
		return "", nil, fmt.Errorf("reading override file: %w", err)
	}
	var parsed struct {
		Bundle    string                    `yaml:"bundle"`
		Overrides map[string]map[string]any `yaml:"overrides"`
	}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return "", nil, fmt.Errorf("parsing override file: %w", err)
	}
	bundle := bundleRef
	if bundle == "" {
		bundle = parsed.Bundle
	}
	if bundle == "" {
		return "", nil, fmt.Errorf("--bundle not set and override file has no `bundle:` field")
	}
	override := parsed.Overrides[componentName]
	// Convert to map[string]any so api.DevspaceGenerateRequest's field
	// type matches.
	var overrideAny map[string]any
	if override != nil {
		overrideAny = make(map[string]any, len(override))
		for k, v := range override {
			overrideAny[k] = v
		}
	}
	return bundle, overrideAny, nil
}
