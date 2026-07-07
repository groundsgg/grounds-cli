package projectcmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/project"
	"github.com/groundsgg/grounds-cli/internal/render"
	"github.com/groundsgg/grounds-cli/internal/workspace"
)

func NewProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Short:   "Manage project selection",
		Example: "  grounds project list\n  grounds project use main\n  grounds project use main --local\n  grounds project current\n  grounds project clear",
	}
	cmd.AddCommand(newList(), newCurrent(), newUse(), newClear())
	return cmd
}

func newList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available projects",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, _, err := baseClient()
			if err != nil {
				return err
			}
			projects, err := c.ListProjects(ctx)
			if err != nil {
				return err
			}
			rows := make([][]any, 0, len(projects.Items))
			for _, p := range projects.Items {
				rows = append(rows, []any{p.ID, p.Slug, p.Name, p.Role})
			}
			render.Table(cmd.OutOrStdout(), []string{"ID", "SLUG", "NAME", "ROLE"}, rows)
			return nil
		},
	}
}

func newCurrent() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the selected project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, cfg, err := baseClient()
			if err != nil {
				return err
			}
			wcfg, err := workspace.Load("")
			if err != nil {
				return err
			}
			selected, err := project.Resolve(ctx, project.ResolveOptions{
				Explicit:        projectFlag(cmd),
				EnvProject:      os.Getenv("GROUNDS_PROJECT"),
				Config:          cfg,
				WorkspaceConfig: wcfg,
				WorkDir:         currentDir(),
				Client:          c,
			})
			if err != nil {
				return err
			}
			render.Table(cmd.OutOrStdout(), []string{"ID", "SLUG", "NAME", "ROLE", "SOURCE"}, [][]any{{
				selected.ID,
				selected.Project.Slug,
				selected.Project.Name,
				selected.Project.Role,
				selected.Source,
			}})
			return nil
		},
	}
}

func newUse() *cobra.Command {
	var local bool
	cmd := &cobra.Command{
		Use:   "use <project>",
		Short: "Set the default project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, cfg, err := baseClient()
			if err != nil {
				return err
			}
			selected, err := project.Resolve(ctx, project.ResolveOptions{
				Explicit: args[0],
				Config:   cfg,
				Client:   c,
			})
			if err != nil {
				return err
			}
			if local {
				wcfg, err := workspace.Load("")
				if err != nil {
					return err
				}
				if wcfg.ProjectDefaults == nil {
					wcfg.ProjectDefaults = map[string]string{}
				}
				root := project.WorkspaceRoot(currentDir())
				wcfg.ProjectDefaults[root] = selected.ID
				if err := workspace.Save("", wcfg); err != nil {
					return err
				}
				render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Project", fmt.Sprintf("using %s locally", displayProject(selected.Project)))
				return nil
			}
			fileCfg, err := config.LoadFile(cfg.Dir)
			if err != nil {
				return err
			}
			fileCfg.DefaultProjectID = selected.ID
			if err := config.Save(fileCfg.Dir, fileCfg); err != nil {
				return err
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Project", fmt.Sprintf("using %s globally", displayProject(selected.Project)))
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "store the default for this workspace")
	return cmd
}

func newClear() *cobra.Command {
	var local bool
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the saved default project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if local {
				wcfg, err := workspace.Load("")
				if err != nil {
					return err
				}
				root := project.WorkspaceRoot(currentDir())
				delete(wcfg.ProjectDefaults, root)
				if err := workspace.Save("", wcfg); err != nil {
					return err
				}
				render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Project", "cleared local default project")
				return nil
			}
			cfg, err := config.Load("")
			if err != nil {
				return err
			}
			fileCfg, err := config.LoadFile(cfg.Dir)
			if err != nil {
				return err
			}
			fileCfg.DefaultProjectID = ""
			if err := config.Save(fileCfg.Dir, fileCfg); err != nil {
				return err
			}
			render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Project", "cleared global default project")
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "clear the default for this workspace")
	return cmd
}

func baseClient() (*api.Client, *config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{
			Store:  auth.NewStore(cfg.Dir),
			Device: defaultDeviceClient(),
		}
	}
	return api.New(cfg.APIURL, ts), cfg, nil
}

func defaultDeviceClient() *auth.DeviceClient {
	return &auth.DeviceClient{
		Issuer:   "https://account.grounds.gg/realms/grounds",
		ClientID: "grounds-cli",
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

func projectFlag(cmd *cobra.Command) string {
	if cmd != nil {
		if p, _ := cmd.Flags().GetString("project"); p != "" {
			return p
		}
	}
	return ""
}

func currentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func displayProject(p api.Project) string {
	if p.Slug != "" {
		return p.Slug + " (" + p.ID + ")"
	}
	if p.Name != "" {
		return p.Name + " (" + p.ID + ")"
	}
	return p.ID
}
