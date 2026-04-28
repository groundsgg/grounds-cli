package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

// NewPreviewCommand returns the `grounds preview` subtree:
//
//	grounds preview list   — show active preview envs in this project
//	grounds preview show   — print one preview env in detail
//	grounds preview pin    — keep an env alive past its 7d TTL
//	grounds preview unpin  — re-enable TTL sweep
func NewPreviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Manage preview environments (target=staging deploys)",
	}
	cmd.AddCommand(newList(), newShow(), newPin(true), newPin(false))
	return cmd
}

// ----------------------------------------------------------------------------
// helpers — duplicated from push package (small enough not to warrant a
// shared internal/cmd helper yet; lift if a third subtree needs it).
// ----------------------------------------------------------------------------

func projectIDFrom(cmd *cobra.Command) string {
	if cmd != nil {
		if p, _ := cmd.Flags().GetString("project"); p != "" {
			return p
		}
	}
	return os.Getenv("GROUNDS_PROJECT")
}

func defaultDevice() *auth.DeviceClient {
	return &auth.DeviceClient{
		Issuer:   "https://account.grounds.gg/realms/grounds",
		ClientID: "grounds-cli",
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

func makeClient(cmd *cobra.Command) (*api.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{Store: auth.NewStore(cfg.Dir), Device: defaultDevice()}
	}
	c := api.New(cfg.APIURL, ts)
	c.ProjectID = projectIDFrom(cmd)
	return c, nil
}

// ----------------------------------------------------------------------------
// list
// ----------------------------------------------------------------------------

func newList() *cobra.Command {
	var includeDeleted bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List preview environments in the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			c, err := makeClient(cmd)
			if err != nil {
				return err
			}
			res, err := c.ListPreviewEnvs(ctx, includeDeleted)
			if err != nil {
				return err
			}
			if len(res.Items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no preview environments")
				return nil
			}
			header := []string{"ID", "PUSH", "NAME", "TYPE", "STATUS", "PINNED", "EXPIRES", "URL"}
			rows := make([][]any, 0, len(res.Items))
			for _, p := range res.Items {
				exp := "—"
				if p.ExpiresAt != nil {
					exp = p.ExpiresAt.Format("2006-01-02")
				}
				pin := "no"
				if p.Pinned {
					pin = "yes"
				}
				rows = append(rows, []any{
					shortID(p.ID),
					shortID(p.PushID),
					p.Push.ManifestName,
					p.Push.ManifestType,
					p.Push.Status,
					pin,
					exp,
					p.PublicURL,
				})
			}
			render.Table(cmd.OutOrStdout(), header, rows)
			return nil
		},
	}
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "show envs already swept by the janitor")
	return cmd
}

// ----------------------------------------------------------------------------
// show
// ----------------------------------------------------------------------------

func newShow() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show one preview environment in detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := makeClient(cmd)
			if err != nil {
				return err
			}
			p, err := c.GetPreviewEnv(ctx, args[0])
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(p)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID:        %s\nPushID:    %s\nNamespace: %s\nName:      %s (%s)\nStatus:    %s\nPinned:    %t\nExpires:   %s\nURL:       %s\n",
				p.ID, p.PushID, p.Namespace,
				p.Push.ManifestName, p.Push.ManifestType,
				p.Push.Status, p.Pinned,
				formatTime(p.ExpiresAt), p.PublicURL)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON instead of human-readable text")
	return cmd
}

// ----------------------------------------------------------------------------
// pin / unpin (single factory; flag boolean differs)
// ----------------------------------------------------------------------------

func newPin(pin bool) *cobra.Command {
	use := "pin <id>"
	short := "Pin a preview env so the TTL janitor skips it"
	if !pin {
		use = "unpin <id>"
		short = "Un-pin a preview env so the TTL janitor can sweep it"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := makeClient(cmd)
			if err != nil {
				return err
			}
			p, err := c.SetPreviewPin(ctx, args[0], pin)
			if err != nil {
				return err
			}
			verb := "pinned"
			if !pin {
				verb = "unpinned"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (%s)\n", verb, shortID(p.ID), p.Push.ManifestName)
			return nil
		},
	}
}

// ----------------------------------------------------------------------------

func shortID(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format(time.RFC3339)
}
