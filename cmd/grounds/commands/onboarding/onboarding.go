// Package onboarding provides `grounds onboarding`: an interactive,
// explain-then-do walkthrough that takes a new developer from zero to a
// running app. Each step first explains the concept, then (after a
// confirm) runs the real command — reusing the existing subcommands so
// there is no duplicated logic.
package onboarding

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/groundsgg/grounds-cli/cmd/grounds/commands"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/cluster"
	"github.com/groundsgg/grounds-cli/cmd/grounds/commands/push"
	"github.com/groundsgg/grounds-cli/internal/api"
	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

const totalSteps = 5

// NewOnboardingCommand returns the `grounds onboarding` command.
func NewOnboardingCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "onboarding",
		Aliases: []string{"onboard", "quickstart"},
		Short:   "Interactive walkthrough that teaches the Grounds dev flow step by step",
		Long: `A guided walkthrough for new developers. Each step is first explained,
then run after you confirm:

  Login  →  Workspace (Bundle)  →  Create app  →  push  →  play

You can skip any step; re-runnable any time.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd)
		},
	}
}

func run(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()
	ctx := context.Background()

	// huh needs an interactive terminal; fail early with a clear message
	// instead of a cryptic prompt error when piped or in CI.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("onboarding needs an interactive terminal (TTY)")
	}

	cfg, err := config.Load("")
	if err != nil {
		return err
	}

	// ── Welcome ──────────────────────────────────────────────────────
	fmt.Fprintln(w)
	fmt.Fprintln(w, render.Bold("👋 Welcome to Grounds"))
	fmt.Fprintln(w, "This assistant walks you to a running app step by step and explains")
	fmt.Fprintln(w, "what's happening along the way. Each step is explained first, then run")
	fmt.Fprintln(w, "after you confirm — skipping is always fine.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Path: "+render.Bold("Login → Workspace → Create app → push → play"))
	start, err := confirm("Ready to start?", "")
	if err != nil {
		return err
	}
	if !start {
		return nil
	}

	// ── 1/5 Login ────────────────────────────────────────────────────
	section(w, 1, "Login",
		"Grounds uses your account (OAuth device flow): you get a code + link,",
		"confirm in the browser, done. The token is stored locally and refreshed",
		"automatically.")
	if loggedIn(cfg) {
		render.StatusLine(w, render.StatusOK, "Login", "You're already logged in")
	} else {
		do, err := confirm("Log in now?", "Opens the device flow in your browser.")
		if err != nil {
			return err
		}
		if !do {
			render.DetailLine(w, render.StatusWarn,
				"Can't continue without login — aborted. Restart with "+render.Command("grounds onboarding"))
			return nil
		}
		if err := runSub(commands.NewLoginCommand(), nil, w); err != nil {
			return err
		}
	}

	// ── 2/5 Workspace (Bundle) ───────────────────────────────────────
	section(w, 2, "Your workspace",
		"A workspace is your isolated dev cluster — just yours. We spin it up as a",
		"Bundle: the full platform environment (Velocity proxy, Player/Social/NATS",
		"services …) so your plugin has something to talk to right away — not an",
		"empty cluster. One-time ~2-3 min.")
	ref := latestBundleRef(ctx, cfg)
	do, err := confirm(
		fmt.Sprintf("Spin up bundle workspace %q now? (~2-3 min)", ref),
		"A fully-equipped platform environment for your plugin.")
	if err != nil {
		return err
	}
	if do {
		if err := runSub(cluster.NewClusterCommand(), []string{"up", "--bundle", ref}, w); err != nil {
			render.StatusLine(w, render.StatusWarn, "Workspace", "Provisioning hit a problem (details above)")
			cont, _ := confirm("Continue with the onboarding anyway?", "")
			if !cont {
				return nil
			}
		}
	}

	// ── 3/5 Create app ───────────────────────────────────────────────
	section(w, 3, "Create an app",
		"grounds.yaml describes your app(s): type (Paper plugin, Velocity, gamemode,",
		"service …), base image and flavors. grounds init scaffolds it interactively.")
	if hasGroundsYaml() {
		render.StatusLine(w, render.StatusOK, "Scaffold", "grounds.yaml already exists — skipped")
	} else {
		do, err := confirm("Scaffold grounds.yaml now?", "Interactive scaffold (type, base image, flavor).")
		if err != nil {
			return err
		}
		if do {
			if err := runSub(commands.NewInitCommand(), nil, w); err != nil {
				render.DetailLine(w, render.StatusWarn, "init skipped: "+err.Error())
			}
		}
	}

	// ── 4/5 push ─────────────────────────────────────────────────────
	section(w, 4, "Deploy with push",
		"grounds push builds your app (via the Gradle plugin) and deploys it to your",
		"workspace — one command for build + deploy.")
	if do2, err := confirm("Run grounds push now?", "Builds + deploys to your workspace."); err != nil {
		return err
	} else if do2 {
		if err := runSub(push.NewPushCommand(), nil, w); err != nil {
			render.DetailLine(w, render.StatusWarn, "push hit a problem — see above")
		}
	}

	// ── 5/5 Play & observe ───────────────────────────────────────────
	section(w, 5, "Play & observe",
		"Connect to your workspace's Velocity proxy (Minecraft client) and watch",
		"live. Use --target=staging for ephemeral preview environments.")
	render.DetailLine(w, render.StatusOK, render.Command("grounds cluster status")+" — workspace status & endpoints")
	render.DetailLine(w, render.StatusOK, render.Command("grounds logs")+" — live logs of your app")
	render.DetailLine(w, render.StatusOK, render.Command("grounds push --target=staging")+" — ephemeral preview env (7d TTL)")

	// ── Done ─────────────────────────────────────────────────────────
	fmt.Fprintln(w)
	render.StatusLine(w, render.StatusOK, "Onboarding", "Done 🎉")
	render.DetailLine(w, render.StatusOK, "Re-run any time with "+render.Command("grounds onboarding"))
	return nil
}

// section prints a styled step header followed by the explanation lines.
func section(w io.Writer, n int, title string, body ...string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, render.Bold(fmt.Sprintf("▸ Step %d/%d — %s", n, totalSteps, title)))
	for _, line := range body {
		fmt.Fprintln(w, "  "+line)
	}
	fmt.Fprintln(w)
}

// confirm shows a Yes/Skip prompt and returns the choice.
func confirm(title, desc string) (bool, error) {
	var ok bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(title).
			Description(desc).
			Affirmative("Yes").
			Negative("Skip").
			Value(&ok),
	))
	if err := form.Run(); err != nil {
		return false, err
	}
	return ok, nil
}

// runSub executes an existing subcommand as if invoked directly, wiring
// its output to w and swallowing cobra's usage/error printing so the
// onboarding stays in control of presentation.
func runSub(c *cobra.Command, args []string, w io.Writer) error {
	c.SetArgs(args)
	c.SetOut(w)
	c.SetErr(w)
	c.SilenceUsage = true
	c.SilenceErrors = true
	return c.Execute()
}

// loggedIn reports whether a usable (refresh-alive) credential is stored.
func loggedIn(cfg *config.Config) bool {
	creds, err := auth.NewStore(cfg.Dir).Load()
	return err == nil && creds != nil && creds.IsRefreshAlive()
}

// latestBundleRef resolves the bundle ref the onboarding provisions: the
// newest published release, falling back to "main" if the catalog can't
// be reached (so the walkthrough still works offline-ish).
func latestBundleRef(ctx context.Context, cfg *config.Config) string {
	c := newClient(cfg)
	if c == nil {
		return "main"
	}
	releases, err := c.ListBundleReleases(ctx)
	if err == nil {
		for _, r := range releases {
			if r.IsLatest {
				return r.Version
			}
		}
		if len(releases) > 0 {
			return releases[0].Version
		}
	}
	return "main"
}

// newClient mirrors the per-command buildClient helper (cluster.go) — the
// CLI has no shared constructor yet, so each command builds its own.
func newClient(cfg *config.Config) *api.Client {
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{
			Store: auth.NewStore(cfg.Dir),
			Device: &auth.DeviceClient{
				Issuer:   "https://account.grounds.gg/realms/grounds",
				ClientID: "grounds-cli",
				HTTP:     &http.Client{Timeout: 30 * time.Second},
			},
		}
	}
	return api.New(cfg.APIURL, ts)
}

func hasGroundsYaml() bool {
	_, err := os.Stat("grounds.yaml")
	return err == nil
}
