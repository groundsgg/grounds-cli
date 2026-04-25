package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

type initFlags struct {
	appName, type_, baseImage, jar string
}

func NewInitCommand() *cobra.Command {
	f := &initFlags{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a grounds.yaml in the current directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Non-interactive when all flags supplied
			if f.appName != "" && f.type_ != "" && f.baseImage != "" {
				return writeGroundsYaml(cmd.OutOrStdout(), f)
			}
			// Interactive
			form := huh.NewForm(huh.NewGroup(
				huh.NewInput().Title("App name").Value(&f.appName),
				huh.NewSelect[string]().Title("Type").
					Options(
						huh.NewOption("gamemode", "gamemode"),
						huh.NewOption("plugin-paper", "plugin-paper"),
						huh.NewOption("plugin-velocity", "plugin-velocity"),
						huh.NewOption("service", "service"),
					).Value(&f.type_),
				huh.NewSelect[string]().Title("Base image").
					Options(
						huh.NewOption("paper", "paper"),
						huh.NewOption("velocity", "velocity"),
					).Value(&f.baseImage),
				huh.NewInput().Title("JAR glob").Placeholder("build/libs/*.jar").Value(&f.jar),
			))
			if err := form.Run(); err != nil {
				return err
			}
			if f.jar == "" {
				f.jar = "build/libs/*.jar"
			}
			return writeGroundsYaml(cmd.OutOrStdout(), f)
		},
	}
	cmd.Flags().StringVar(&f.appName, "app-name", "", "")
	cmd.Flags().StringVar(&f.type_, "type", "", "gamemode | plugin-paper | plugin-velocity | service")
	cmd.Flags().StringVar(&f.baseImage, "base-image", "", "paper | velocity")
	cmd.Flags().StringVar(&f.jar, "jar", "build/libs/*.jar", "")
	return cmd
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
	body := fmt.Sprintf(
		"name: %s\ntype: %s\nbaseImage: %s\njar: %s\n",
		f.appName, f.type_, f.baseImage, f.jar,
	)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, "→ Wrote grounds.yaml")
	fmt.Fprintln(out, "Next: grounds push")
	return nil
}
