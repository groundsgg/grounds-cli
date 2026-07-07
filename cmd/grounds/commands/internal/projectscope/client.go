package projectscope

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/project"
	"github.com/groundsgg/grounds-cli/internal/workspace"
)

func BuildClient(ctx context.Context, cmd *cobra.Command) (*api.Client, *config.Config, project.Selection, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, project.Selection{}, err
	}
	c := api.New(cfg.APIURL, tokenSource(cfg))
	wcfg, err := workspace.Load("")
	if err != nil {
		return nil, nil, project.Selection{}, err
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
		return nil, nil, project.Selection{}, err
	}
	c.ProjectID = selected.ID
	return c, cfg, selected, nil
}

func BaseClient() (*api.Client, *config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}
	return api.New(cfg.APIURL, tokenSource(cfg)), cfg, nil
}

func tokenSource(cfg *config.Config) api.TokenSource {
	ts := api.NewEnvTokenSource()
	if ts != nil {
		return ts
	}
	return &auth.FileTokenSource{
		Store:  auth.NewStore(cfg.Dir),
		Device: defaultDeviceClient(),
	}
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
		if flag := cmd.Flag("project"); flag != nil {
			return flag.Value.String()
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
