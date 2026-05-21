package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

type initFlags struct {
	appName, type_, baseImage, jar, flavor string
}

var loadInitBaseImageCatalog = func(ctx context.Context, cmd *cobra.Command) (*api.BaseImageCatalog, error) {
	configDir, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configDir)
	if err != nil {
		return nil, err
	}
	return api.New(cfg.APIURL, nil).ListBaseImages(ctx)
}

func NewInitCommand() *cobra.Command {
	f := &initFlags{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a grounds.yaml in the current directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			catalog, err := loadInitBaseImageCatalog(cmd.Context(), cmd)
			if err != nil {
				catalog = fallbackBaseImageCatalog()
				render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Init", "Using built-in base image defaults")
				render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, err.Error())
			}
			// Non-interactive when all flags supplied
			if f.appName != "" && f.type_ != "" && f.baseImage != "" {
				if err := validateBaseImageChoice(catalog, f.type_, f.baseImage); err != nil {
					return err
				}
				if err := validateJarPath(f.jar); err != nil {
					return err
				}
				if err := validateFlavorKey(f.flavor); err != nil {
					return err
				}
				return writeGroundsYaml(cmd.OutOrStdout(), f)
			}
			firstStep := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("App name").Value(&f.appName),
				huh.NewSelect[string]().Title("Type").
					Options(typeOptions(catalog)...).Value(&f.type_),
			))
			if err := firstStep.Run(); err != nil {
				return err
			}
			secondStep := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().Title("Base image").
					Options(baseImageOptionsForType(catalog, f.type_)...).Value(&f.baseImage),
				huh.NewInput().Title("JAR path (optional)").Placeholder("build/libs/my-plugin.jar").Value(&f.jar),
				huh.NewInput().Title("App flavor key (optional)").Placeholder("paper").Value(&f.flavor),
			))
			if err := secondStep.Run(); err != nil {
				return err
			}
			if err := validateBaseImageChoice(catalog, f.type_, f.baseImage); err != nil {
				return err
			}
			if err := validateJarPath(f.jar); err != nil {
				return err
			}
			if err := validateFlavorKey(f.flavor); err != nil {
				return err
			}
			return writeGroundsYaml(cmd.OutOrStdout(), f)
		},
	}
	cmd.Flags().StringVar(&f.appName, "app-name", "", "")
	cmd.Flags().StringVar(&f.type_, "type", "", "gamemode | plugin-paper | plugin-velocity | service")
	cmd.Flags().StringVar(&f.baseImage, "base-image", "", "base image catalog key, e.g. paper or paper@0.8.2")
	cmd.Flags().StringVar(&f.jar, "jar", "", "exact JAR path; omit for Gradle auto-detection")
	cmd.Flags().StringVar(&f.flavor, "flavor", "", "app flavor key to scaffold under grounds.yaml flavors")
	return cmd
}

func fallbackBaseImageCatalog() *api.BaseImageCatalog {
	return &api.BaseImageCatalog{Items: []api.BaseImageSource{
		{Key: "paper", DisplayName: "Paper", ManifestType: "plugin-paper"},
		{Key: "paper-gamemode", DisplayName: "Paper gamemode", ManifestType: "gamemode"},
		{Key: "velocity", DisplayName: "Velocity", ManifestType: "plugin-velocity"},
		{Key: "service", DisplayName: "JVM service", ManifestType: "service"},
	}}
}

func typeOptions(catalog *api.BaseImageCatalog) []huh.Option[string] {
	seen := map[string]bool{}
	var options []huh.Option[string]
	for _, source := range catalog.Items {
		if source.ManifestType == "" || seen[source.ManifestType] {
			continue
		}
		seen[source.ManifestType] = true
		options = append(options, huh.NewOption(source.ManifestType, source.ManifestType))
	}
	return options
}

func baseImageOptionsForType(catalog *api.BaseImageCatalog, manifestType string) []huh.Option[string] {
	var options []huh.Option[string]
	for _, source := range catalog.Items {
		if source.ManifestType != manifestType {
			continue
		}
		label := source.Key
		if source.DisplayName != "" {
			label = fmt.Sprintf("%s (%s)", source.DisplayName, source.Key)
		}
		options = append(options, huh.NewOption(label, source.Key))
		for _, version := range source.Versions {
			if version.Selectable {
				value := source.Key + "@" + version.Version
				options = append(options, huh.NewOption(value, value))
			}
		}
	}
	return options
}

func validateBaseImageChoice(catalog *api.BaseImageCatalog, manifestType, baseImage string) error {
	key, version, _ := strings.Cut(baseImage, "@")
	for _, source := range catalog.Items {
		if source.Key != key {
			continue
		}
		if source.ManifestType != manifestType {
			return fmt.Errorf("baseImage %s requires type %s, got %s", key, source.ManifestType, manifestType)
		}
		if version == "" {
			return nil
		}
		for _, candidate := range source.Versions {
			if candidate.Version == version && candidate.Selectable {
				return nil
			}
		}
		return fmt.Errorf("baseImage version %s is not selectable", baseImage)
	}
	return fmt.Errorf("unknown baseImage %s for type %s", key, manifestType)
}

func validateJarPath(jar string) error {
	if strings.ContainsAny(jar, "*?[") {
		return fmt.Errorf("jar must be an exact JAR path; leave it empty for Gradle auto-detection")
	}
	return nil
}

var flavorKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

func validateFlavorKey(flavor string) error {
	if flavor == "" {
		return nil
	}
	if !flavorKeyPattern.MatchString(flavor) {
		return fmt.Errorf("flavor must match %s", flavorKeyPattern.String())
	}
	return nil
}

func writeGroundsYaml(out io.Writer, f *initFlags) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "grounds.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("grounds.yaml already exists at %s", path)
	}
	body := renderGroundsYaml(f)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return err
	}
	render.StatusLine(out, render.StatusOK, "Init", "Wrote grounds.yaml")
	next := "Next: run " + render.Command("grounds push") + "."
	if f.flavor != "" {
		next = "Next: run " + render.Command("grounds push --flavor="+f.flavor) + "."
	}
	render.DetailLine(out, render.StatusOK, next)
	return nil
}

func renderGroundsYaml(f *initFlags) string {
	if f.flavor == "" {
		body := fmt.Sprintf("name: %s\ntype: %s\nbaseImage: %s\n", f.appName, f.type_, f.baseImage)
		if f.jar != "" {
			body += fmt.Sprintf("jar: %s\n", f.jar)
		}
		return body
	}

	body := fmt.Sprintf(
		"name: %s\nflavors:\n  %s:\n    type: %s\n    baseImage: %s\n",
		f.appName,
		f.flavor,
		f.type_,
		f.baseImage,
	)
	if f.jar != "" {
		body += fmt.Sprintf("    jar: %s\n", f.jar)
	}
	return body
}
