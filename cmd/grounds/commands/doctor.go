package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
)

type checkResult struct {
	name string
	ok   bool
	msg  string
}

func NewDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose env, auth, API reachability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			results := runChecks(context.Background())
			anyFail := false
			for _, r := range results {
				if r.ok {
					fmt.Fprintln(out, render.Green("✔"), r.name, " ", r.msg)
				} else {
					anyFail = true
					fmt.Fprintln(out, render.Red("✗"), r.name, " ", r.msg)
				}
			}
			if anyFail {
				return fmt.Errorf("doctor: 1 or more checks failed")
			}
			fmt.Fprintln(out, "Overall:", render.Green("OK"))
			return nil
		},
	}
}

func runChecks(ctx context.Context) []checkResult {
	out := []checkResult{}
	out = append(out, checkConfig())
	out = append(out, checkAuth(ctx))
	out = append(out, checkAPI(ctx))
	out = append(out, checkGradle())
	out = append(out, checkJava())
	return out
}

func checkConfig() checkResult {
	cfg, err := config.Load("")
	if err != nil {
		return checkResult{name: "config", ok: false, msg: err.Error()}
	}
	return checkResult{name: "config", ok: true, msg: cfg.Dir}
}

func checkAuth(ctx context.Context) checkResult {
	cfg, err := config.Load("")
	if err != nil {
		return checkResult{name: "auth", ok: false, msg: err.Error()}
	}
	if t := auth.EnvToken(); t != "" {
		return checkResult{name: "auth", ok: true, msg: "GROUNDS_TOKEN set"}
	}
	store := auth.NewStore(cfg.Dir)
	c, err := store.Load()
	if err != nil {
		return checkResult{name: "auth", ok: false, msg: "not logged in"}
	}
	// The access_token Keycloak issues is short-lived (≈5 min by default)
	// but the refresh_token lives much longer (≈30 d). Real commands go
	// through FileTokenSource.Token which transparently refreshes; doctor
	// must do the same or it reports "expired" while everything else
	// works fine. Drop down to refresh + persist if needed.
	if time.Now().After(c.ExpiresAt.Add(-30 * time.Second)) {
		if time.Now().After(c.RefreshExpiresAt) {
			return checkResult{name: "auth", ok: false, msg: "session expired (run 'grounds login')"}
		}
		device := &auth.DeviceClient{
			Issuer:   defaultIssuer,
			ClientID: defaultClientID,
			HTTP:     defaultHTTP(),
		}
		fresh, err := device.Refresh(ctx, c.RefreshToken)
		if err != nil {
			return checkResult{name: "auth", ok: false, msg: "refresh failed: " + err.Error() + " (run 'grounds login')"}
		}
		c.AccessToken = fresh.AccessToken
		c.RefreshToken = fresh.RefreshToken
		c.ExpiresAt = time.Now().Add(time.Duration(fresh.ExpiresIn) * time.Second)
		c.RefreshExpiresAt = time.Now().Add(time.Duration(fresh.RefreshExpiresIn) * time.Second)
		if err := store.Save(c); err != nil {
			return checkResult{name: "auth", ok: false, msg: "refresh ok but persist failed: " + err.Error()}
		}
		return checkResult{name: "auth", ok: true, msg: c.PreferredUser + " (refreshed; valid " + time.Until(c.ExpiresAt).Round(time.Minute).String() + ")"}
	}
	return checkResult{name: "auth", ok: true, msg: c.PreferredUser + " (valid " + time.Until(c.ExpiresAt).Round(time.Minute).String() + ", refresh in " + time.Until(c.RefreshExpiresAt).Round(time.Hour).String() + ")"}
}

func checkAPI(ctx context.Context) checkResult {
	cfg, err := config.Load("")
	if err != nil {
		return checkResult{name: "api", ok: false, msg: err.Error()}
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx2, "GET", cfg.APIURL+"/healthz", nil)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return checkResult{name: "api", ok: false, msg: err.Error()}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		return checkResult{name: "api", ok: false, msg: fmt.Sprintf("status %d", resp.StatusCode)}
	}
	return checkResult{name: "api", ok: true, msg: cfg.APIURL + " → 200 /healthz"}
}

func checkGradle() checkResult {
	name := "gradlew"
	if runtime.GOOS == "windows" {
		name = "gradlew.bat"
	}
	if _, err := exec.LookPath("./" + name); err != nil {
		return checkResult{name: "gradle", ok: false, msg: "./gradlew not in cwd (run from project root)"}
	}
	return checkResult{name: "gradle", ok: true, msg: "./" + name}
}

func checkJava() checkResult {
	out, err := exec.Command("java", "-version").CombinedOutput()
	if err != nil {
		return checkResult{name: "java", ok: false, msg: "java not in PATH"}
	}
	return checkResult{name: "java", ok: true, msg: extractJavaVersion(string(out))}
}

func extractJavaVersion(s string) string {
	// "openjdk version \"17.0.12\" 2024-07-16"
	for _, line := range []byte(s) {
		_ = line
	}
	// Very forgiving: just return first line
	if i := indexOfNewline(s); i > 0 {
		return s[:i]
	}
	return s
}

func indexOfNewline(s string) int {
	for i, r := range s {
		if r == '\n' {
			return i
		}
	}
	return -1
}
