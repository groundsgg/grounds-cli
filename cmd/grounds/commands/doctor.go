package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/groundsgg/grounds-cli/internal/auth"
	"github.com/groundsgg/grounds-cli/internal/config"
	"github.com/groundsgg/grounds-cli/internal/render"
	"github.com/groundsgg/grounds-cli/internal/version"
)

var ErrDoctorIssuesFound = errors.New("doctor issues found")

type checkStatus string

const (
	statusOK    checkStatus = "ok"
	statusWarn  checkStatus = "warn"
	statusError checkStatus = "error"
)

type checkResult struct {
	name    string
	status  checkStatus
	summary string
	details []string
}

type doctorCheck struct {
	name string
	run  func(context.Context) checkResult
}

var doctorCheckLatest = version.CheckLatest

func NewDoctorCommand() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:          "doctor",
		Short:        "Diagnose env, auth, API reachability",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			return runDoctorChecks(context.Background(), out, doctorChecks(), isTerminal(out), strict)
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero when doctor finds errors")
	return cmd
}

func doctorChecks() []doctorCheck {
	return []doctorCheck{
		{name: "Version", run: checkVersion},
		{name: "Config", run: func(context.Context) checkResult { return checkConfig() }},
		{name: "Auth", run: checkAuth},
		{name: "API", run: checkAPI},
		{name: "Gradle", run: func(context.Context) checkResult { return checkGradle() }},
		{name: "Java", run: func(context.Context) checkResult { return checkJava() }},
	}
}

func runDoctorChecks(ctx context.Context, out io.Writer, checks []doctorCheck, interactive bool, strict bool) error {
	fmt.Fprintln(out, "Doctor summary:")
	fmt.Fprintln(out)

	results := make([]checkResult, 0, len(checks))
	for _, check := range checks {
		var result checkResult
		if interactive {
			result = runDoctorCheckWithSpinner(ctx, out, check)
		} else {
			result = check.run(ctx)
		}
		if result.name == "" {
			result.name = check.name
		}
		results = append(results, result)
		printCheckResult(out, result)
	}
	fmt.Fprintln(out)

	return printDoctorFooter(out, results, strict)
}

func printCheckResult(out io.Writer, r checkResult) {
	render.StatusLine(out, renderStatus(r.status), r.name, r.summary)
	for _, detail := range r.details {
		render.DetailLine(out, renderStatus(r.status), detail)
	}
}

func renderStatus(status checkStatus) render.StatusKind {
	switch status {
	case statusWarn:
		return render.StatusWarn
	case statusError:
		return render.StatusError
	default:
		return render.StatusOK
	}
}

func printDoctorFooter(out io.Writer, results []checkResult, strict bool) error {
	errorCount := 0
	warnCount := 0
	for _, r := range results {
		switch r.status {
		case statusError:
			errorCount++
		case statusWarn:
			warnCount++
		}
	}
	if errorCount > 0 {
		fmt.Fprintf(out, "%s Doctor found issues in %d %s.\n", render.Red("✗"), errorCount, categoryWord(errorCount))
		if strict {
			return ErrDoctorIssuesFound
		}
		return nil
	}
	if warnCount > 0 {
		fmt.Fprintf(out, "%s Doctor found warnings in %d %s.\n", render.Yellow("!"), warnCount, categoryWord(warnCount))
		return nil
	}
	fmt.Fprintln(out, render.Green("✓"), "Doctor found no issues.")
	return nil
}

func categoryWord(count int) string {
	if count == 1 {
		return "category"
	}
	return "categories"
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func runDoctorCheckWithSpinner(ctx context.Context, out io.Writer, check doctorCheck) checkResult {
	done := make(chan checkResult, 1)
	go func() {
		done <- check.run(ctx)
	}()

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	printSpinnerLine(out, frames[frame], check.name)
	for {
		select {
		case result := <-done:
			clearSpinnerLine(out)
			return result
		case <-ticker.C:
			frame = (frame + 1) % len(frames)
			clearSpinnerLine(out)
			printSpinnerLine(out, frames[frame], check.name)
		}
	}
}

func spinnerBadge(frame string) string {
	return render.Yellow("[" + frame + "]")
}

func printSpinnerLine(out io.Writer, frame, name string) {
	fmt.Fprintf(out, "%s %s\n", spinnerBadge(frame), name)
}

func clearSpinnerLine(out io.Writer) {
	fmt.Fprint(out, "\033[1A\r\033[K")
}

func checkVersion(ctx context.Context) checkResult {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	report, err := doctorCheckLatest(ctx2, version.CheckOptions{
		Current:    version.Version,
		APIBaseURL: version.DefaultReleaseAPIBaseURL,
		HTTPClient: versionCheckHTTPClient,
	})
	if err != nil {
		return checkResult{
			name:    "Version",
			status:  statusWarn,
			summary: "Could not check for CLI updates",
			details: []string{"Run `grounds version --check` to try again. " + err.Error()},
		}
	}
	if report.UpdateAvailable {
		return checkResult{
			name:    "Version",
			status:  statusWarn,
			summary: fmt.Sprintf("Grounds CLI %s is outdated; latest is %s", report.Current, report.Latest),
			details: []string{"Run `grounds version --check` for update instructions."},
		}
	}
	if !report.Comparable {
		return checkResult{name: "Version", status: statusOK, summary: fmt.Sprintf("Grounds CLI %s is a local build (latest release is %s)", report.Current, report.Latest)}
	}
	return checkResult{name: "Version", status: statusOK, summary: "Grounds CLI " + report.Current + " is up to date"}
}

func checkConfig() checkResult {
	cfg, err := config.Load("")
	if err != nil {
		return checkResult{name: "Config", status: statusError, summary: "Could not load Grounds configuration", details: []string{err.Error()}}
	}
	return checkResult{name: "Config", status: statusOK, summary: "Using configuration from " + cfg.Dir}
}

func checkAuth(ctx context.Context) checkResult {
	cfg, err := config.Load("")
	if err != nil {
		return checkResult{name: "Auth", status: statusError, summary: "Could not load authentication configuration", details: []string{err.Error()}}
	}
	if t := auth.EnvToken(); t != "" {
		return checkResult{name: "Auth", status: statusOK, summary: "Using GROUNDS_TOKEN from the environment"}
	}
	store := auth.NewStore(cfg.Dir)
	c, err := store.Load()
	if err != nil {
		return checkResult{name: "Auth", status: statusError, summary: "You are not logged in", details: []string{"Run `grounds login` to authenticate."}}
	}
	// The access_token Keycloak issues is short-lived (≈5 min by default)
	// but the refresh_token is an offline token that doesn't expire
	// (Keycloak signals that with refresh_expires_in: 0, which we map
	// to a zero `time.Time` — see auth.RefreshExpiryFromSeconds).
	// Real commands go through FileTokenSource.Token which transparently
	// refreshes; doctor must do the same or it reports "expired" while
	// everything else works fine. Drop down to refresh + persist if needed.
	if time.Now().After(c.ExpiresAt.Add(-30 * time.Second)) {
		if !c.IsRefreshAlive() {
			return checkResult{name: "Auth", status: statusError, summary: "Your login session has expired", details: []string{"Run `grounds login` to authenticate again."}}
		}
		device := &auth.DeviceClient{
			Issuer:   defaultIssuer,
			ClientID: defaultClientID,
			HTTP:     defaultHTTP(),
		}
		fresh, err := device.Refresh(ctx, c.RefreshToken)
		if err != nil {
			return checkResult{name: "Auth", status: statusError, summary: "Could not refresh your login session", details: []string{err.Error(), "Run `grounds login` to authenticate again."}}
		}
		c.AccessToken = fresh.AccessToken
		c.RefreshToken = fresh.RefreshToken
		c.ExpiresAt = time.Now().Add(time.Duration(fresh.ExpiresIn) * time.Second)
		c.RefreshExpiresAt = auth.RefreshExpiryFromSeconds(fresh.RefreshExpiresIn)
		if err := store.Save(c); err != nil {
			return checkResult{name: "Auth", status: statusError, summary: "Refreshed your session but could not save it", details: []string{err.Error()}}
		}
		return checkResult{name: "Auth", status: statusOK, summary: c.PreferredUser + " is logged in (session refreshed, valid for " + time.Until(c.ExpiresAt).Round(time.Minute).String() + ")"}
	}
	refreshSummary := "no expiry (offline token)"
	if !c.RefreshExpiresAt.IsZero() {
		refreshSummary = "refresh token valid for " + time.Until(c.RefreshExpiresAt).Round(time.Hour).String()
	}
	return checkResult{name: "Auth", status: statusOK, summary: c.PreferredUser + " is logged in (session valid for " + time.Until(c.ExpiresAt).Round(time.Minute).String() + ", " + refreshSummary + ")"}
}

func checkAPI(ctx context.Context) checkResult {
	cfg, err := config.Load("")
	if err != nil {
		return checkResult{name: "API", status: statusError, summary: "Could not load API configuration", details: []string{err.Error()}}
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx2, "GET", cfg.APIURL+"/healthz", nil)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return checkResult{name: "API", status: statusError, summary: "Could not reach the Grounds API at " + cfg.APIURL, details: []string{err.Error()}}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return checkAPIResult(cfg.APIURL, resp.StatusCode, nil)
}

func checkAPIResult(apiURL string, statusCode int, details []string) checkResult {
	if statusCode != http.StatusOK {
		return checkResult{
			name:    "API",
			status:  statusError,
			summary: fmt.Sprintf("Grounds API at %s returned HTTP %d", apiURL, statusCode),
			details: details,
		}
	}
	return checkResult{
		name:    "API",
		status:  statusOK,
		summary: "Connected to the Grounds API at " + apiURL + " (/healthz returned HTTP 200)",
	}
}

func checkGradle() checkResult {
	name := "gradlew"
	if runtime.GOOS == "windows" {
		name = "gradlew.bat"
	}
	if _, err := exec.LookPath("./" + name); err != nil {
		return checkResult{name: "Gradle", status: statusError, summary: "Gradle wrapper was not found in this directory", details: []string{"Run `grounds doctor` from your project root, where `./" + name + "` exists."}}
	}
	return checkResult{name: "Gradle", status: statusOK, summary: "Found Gradle wrapper ./" + name}
}

func checkJava() checkResult {
	out, err := exec.Command("java", "-version").CombinedOutput()
	if err != nil {
		return checkResult{name: "Java", status: statusError, summary: "Java was not found on PATH", details: []string{"Install Java and make sure the java command is available."}}
	}
	return checkResult{name: "Java", status: statusOK, summary: "Found Java runtime (" + extractJavaVersion(string(out)) + ")"}
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
