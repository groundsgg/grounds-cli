package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/render"
	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
)

func NewWorkspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage local plugin workspace overrides",
	}
	cmd.AddCommand(newAdd(), newList(), newEnable(), newDoctor(), newScan())
	return cmd
}

func newAdd() *cobra.Command {
	var artifact string
	var build string
	var variant string
	var disabled bool
	cmd := &cobra.Command{
		Use:   "add <id> <path>",
		Short: "Add a local plugin repository",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internalworkspace.Load("")
			if err != nil {
				return err
			}
			abs, err := filepath.Abs(args[1])
			if err != nil {
				return err
			}
			repo := cfg.Repos[args[0]]
			repo.Path = abs
			if variant == "" {
				repo.Artifact = firstNonEmpty(artifact, "build/libs/*.jar")
				repo.Build = firstNonEmpty(build, "./gradlew build")
				repo.Enabled = !disabled
			} else {
				if repo.Variants == nil {
					repo.Variants = map[string]internalworkspace.Variant{}
				}
				repo.Variants[variant] = internalworkspace.Variant{
					Artifact: firstNonEmpty(artifact, filepath.ToSlash(filepath.Join(variant, "build", "libs", "*.jar"))),
					Build:    firstNonEmpty(build, "./gradlew :"+variant+":shadowJar"),
					Enabled:  !disabled,
				}
			}
			if cfg.Repos == nil {
				cfg.Repos = map[string]internalworkspace.Repo{}
			}
			cfg.Repos[args[0]] = repo
			if err := internalworkspace.Save("", cfg); err != nil {
				return err
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Workspace", "Added "+args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&artifact, "artifact", "", "artifact glob relative to the repo")
	cmd.Flags().StringVar(&build, "build", "", "build command run in the repo before resolving artifacts")
	cmd.Flags().StringVar(&variant, "variant", "", "plugin variant: paper, velocity, or minestom")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "add the mapping disabled")
	return cmd
}

func newList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List local plugin workspace mappings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := internalworkspace.Load("")
			if err != nil {
				return err
			}
			renderMappings(cmd.OutOrStdout(), cfg)
			return nil
		},
	}
}

func newEnable() *cobra.Command {
	var disabled bool
	cmd := &cobra.Command{
		Use:   "enable <id> [variant]",
		Short: "Enable a local plugin workspace mapping",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internalworkspace.Load("")
			if err != nil {
				return err
			}
			repo, ok := cfg.Repos[args[0]]
			if !ok {
				return fmt.Errorf("workspace repo %q not found", args[0])
			}
			enabled := !disabled
			if len(args) == 2 {
				variant, ok := repo.Variants[args[1]]
				if !ok {
					return fmt.Errorf("workspace repo %q variant %q not found", args[0], args[1])
				}
				variant.Enabled = enabled
				repo.Variants[args[1]] = variant
			} else {
				repo.Enabled = enabled
			}
			cfg.Repos[args[0]] = repo
			if err := internalworkspace.Save("", cfg); err != nil {
				return err
			}
			summary := "Enabled " + args[0]
			if disabled {
				summary = "Disabled " + args[0]
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Workspace", summary)
			return nil
		},
	}
	cmd.Flags().BoolVar(&disabled, "disable", false, "disable instead of enabling")
	return cmd
}

func newDoctor() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configured local plugin workspace mappings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := internalworkspace.Load("")
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, id := range sortedRepoIDs(cfg) {
				repo := cfg.Repos[id]
				if _, err := os.Stat(repo.Path); err != nil {
					render.StatusLine(out, render.StatusWarn, "Workspace", fmt.Sprintf("Repo path missing (id=%s, path=%s)", id, repo.Path))
					continue
				}
				render.StatusLine(out, render.StatusOK, "Workspace", fmt.Sprintf("Repo path exists (id=%s, path=%s)", id, repo.Path))
			}
			return nil
		},
	}
}

func newScan() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "scan <root> [root...]",
		Short: "Scan directories for local plugin repositories",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("scan requires at least one root")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			proposed, err := internalworkspace.ScanRoots(args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Proposed workspace mappings:")
			renderMappings(out, proposed)
			if !yes {
				ok, err := confirm(cmd.InOrStdin(), out, "Write these mappings? [y/N] ")
				if err != nil {
					return err
				}
				if !ok {
					render.StatusLine(out, render.StatusWarn, "Workspace", "Scan cancelled")
					return nil
				}
			}
			cfg, err := internalworkspace.Load("")
			if err != nil {
				return err
			}
			if cfg.Repos == nil {
				cfg.Repos = map[string]internalworkspace.Repo{}
			}
			for id, repo := range proposed.Repos {
				cfg.Repos[id] = repo
			}
			if err := internalworkspace.Save("", cfg); err != nil {
				return err
			}
			render.StatusLine(out, render.StatusOK, "Workspace", "Saved scanned mappings")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "write scanned mappings without prompting")
	return cmd
}

func renderMappings(w io.Writer, cfg *internalworkspace.Config) {
	rows := [][]any{}
	for _, id := range sortedRepoIDs(cfg) {
		repo := cfg.Repos[id]
		if repo.Artifact != "" {
			rows = append(rows, []any{id, "", repo.Enabled, repo.Path, repo.Artifact, repo.Build})
		}
		for _, variant := range sortedVariantIDs(repo) {
			v := repo.Variants[variant]
			rows = append(rows, []any{id, variant, v.Enabled, repo.Path, v.Artifact, v.Build})
		}
	}
	render.Table(w, []string{"ID", "Variant", "Enabled", "Path", "Artifact", "Build"}, rows)
}

func sortedRepoIDs(cfg *internalworkspace.Config) []string {
	if cfg == nil {
		return nil
	}
	ids := make([]string, 0, len(cfg.Repos))
	for id := range cfg.Repos {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedVariantIDs(repo internalworkspace.Repo) []string {
	variants := make([]string, 0, len(repo.Variants))
	for variant := range repo.Variants {
		variants = append(variants, variant)
	}
	sort.Strings(variants)
	return variants
}

func confirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	fmt.Fprint(out, prompt)
	var answer string
	if _, err := fmt.Fscan(in, &answer); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
