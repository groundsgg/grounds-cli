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
		Short:   "Interaktiver Einstieg: erklärt den Grounds-Dev-Flow Schritt für Schritt",
		Long: `Ein geführter Walkthrough für neue Entwickler. Jeder Schritt wird erst
erklärt und dann nach deiner Bestätigung ausgeführt:

  Login  →  Workspace (Bundle)  →  App anlegen  →  push  →  spielen

Überspringen ist überall möglich; jederzeit erneut aufrufbar.`,
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
		return fmt.Errorf("onboarding braucht ein interaktives Terminal (TTY)")
	}

	cfg, err := config.Load("")
	if err != nil {
		return err
	}

	// ── Welcome ──────────────────────────────────────────────────────
	fmt.Fprintln(w)
	fmt.Fprintln(w, render.Bold("👋 Willkommen bei Grounds"))
	fmt.Fprintln(w, "Dieser Assistent bringt dich Schritt für Schritt zu einer laufenden App")
	fmt.Fprintln(w, "und erklärt unterwegs, was passiert. Jeder Schritt wird erst erklärt,")
	fmt.Fprintln(w, "dann nach deiner Bestätigung ausgeführt — Überspringen ist immer ok.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Weg: "+render.Bold("Login → Workspace → App anlegen → push → spielen"))
	start, err := confirm("Los geht's?", "")
	if err != nil {
		return err
	}
	if !start {
		return nil
	}

	// ── 1/5 Login ────────────────────────────────────────────────────
	section(w, 1, "Login",
		"Grounds nutzt deinen Account (OAuth Device-Flow): du bekommst einen Code",
		"+ Link, bestätigst im Browser, fertig. Das Token wird lokal gespeichert",
		"und automatisch erneuert.")
	if loggedIn(cfg) {
		render.StatusLine(w, render.StatusOK, "Login", "Du bist bereits eingeloggt")
	} else {
		do, err := confirm("Jetzt einloggen?", "Öffnet den Device-Flow im Browser.")
		if err != nil {
			return err
		}
		if !do {
			render.DetailLine(w, render.StatusWarn,
				"Ohne Login geht's nicht weiter — abgebrochen. Neustart mit "+render.Command("grounds onboarding"))
			return nil
		}
		if err := runSub(commands.NewLoginCommand(), nil, w); err != nil {
			return err
		}
	}

	// ── 2/5 Workspace (Bundle) ───────────────────────────────────────
	section(w, 2, "Dein Workspace",
		"Ein Workspace ist dein isolierter Dev-Cluster — nur deiner. Wir ziehen ihn",
		"als Bundle hoch: die komplette Platform-Umgebung (Velocity-Proxy, Player-,",
		"Social-, NATS-Services …), damit dein Plugin sofort etwas zum Reden hat —",
		"kein leeres Cluster. Einmalig ~2-3 min.")
	ref := latestBundleRef(ctx, cfg)
	do, err := confirm(
		fmt.Sprintf("Bundle-Workspace »%s« jetzt hochziehen? (~2-3 min)", ref),
		"Voll ausgestattete Platform-Umgebung für dein Plugin.")
	if err != nil {
		return err
	}
	if do {
		if err := runSub(cluster.NewClusterCommand(), []string{"up", "--bundle", ref}, w); err != nil {
			render.StatusLine(w, render.StatusWarn, "Workspace", "Provisioning hatte ein Problem (Details oben)")
			cont, _ := confirm("Trotzdem mit dem Onboarding weitermachen?", "")
			if !cont {
				return nil
			}
		}
	}

	// ── 3/5 App anlegen ──────────────────────────────────────────────
	section(w, 3, "Eine App anlegen",
		"grounds.yaml beschreibt deine App(s): Typ (Paper-Plugin, Velocity, Gamemode,",
		"Service …), Base-Image und Flavors. grounds init legt sie interaktiv an.")
	if hasGroundsYaml() {
		render.StatusLine(w, render.StatusOK, "Scaffold", "grounds.yaml existiert bereits — übersprungen")
	} else {
		do, err := confirm("grounds.yaml jetzt anlegen?", "Interaktiver Scaffold (Typ, Base-Image, Flavor).")
		if err != nil {
			return err
		}
		if do {
			if err := runSub(commands.NewInitCommand(), nil, w); err != nil {
				render.DetailLine(w, render.StatusWarn, "init übersprungen: "+err.Error())
			}
		}
	}

	// ── 4/5 push ─────────────────────────────────────────────────────
	section(w, 4, "Deployen mit push",
		"grounds push baut deine App (via Gradle-Plugin) und deployt sie in deinen",
		"Workspace — ein Kommando für Build + Deploy.")
	if do2, err := confirm("Jetzt »grounds push«?", "Baut + deployt in deinen Workspace."); err != nil {
		return err
	} else if do2 {
		if err := runSub(push.NewPushCommand(), nil, w); err != nil {
			render.DetailLine(w, render.StatusWarn, "push hatte ein Problem — siehe oben")
		}
	}

	// ── 5/5 Spielen & beobachten ─────────────────────────────────────
	section(w, 5, "Spielen & beobachten",
		"Verbinde dich mit dem Velocity-Proxy deines Workspaces (Minecraft-Client)",
		"und schau live zu. Mit --target=staging bekommst du ephemere Preview-Envs.")
	render.DetailLine(w, render.StatusOK, render.Command("grounds cluster status")+" — Workspace-Status & Endpunkte")
	render.DetailLine(w, render.StatusOK, render.Command("grounds logs")+" — Live-Logs deiner App")
	render.DetailLine(w, render.StatusOK, render.Command("grounds push --target=staging")+" — ephemere Preview-Env (7d TTL)")

	// ── Done ─────────────────────────────────────────────────────────
	fmt.Fprintln(w)
	render.StatusLine(w, render.StatusOK, "Onboarding", "Geschafft 🎉")
	render.DetailLine(w, render.StatusOK, "Jederzeit wiederholen mit "+render.Command("grounds onboarding"))
	return nil
}

// section prints a styled step header followed by the explanation lines.
func section(w io.Writer, n int, title string, body ...string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, render.Bold(fmt.Sprintf("▸ Schritt %d/%d — %s", n, totalSteps, title)))
	for _, line := range body {
		fmt.Fprintln(w, "  "+line)
	}
	fmt.Fprintln(w)
}

// confirm shows a Ja/Überspringen prompt and returns the choice.
func confirm(title, desc string) (bool, error) {
	var ok bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(title).
			Description(desc).
			Affirmative("Ja").
			Negative("Überspringen").
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
