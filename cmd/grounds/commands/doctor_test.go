package commands

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/groundsgg/grounds-cli/internal/version"
)

func TestDoctorRuns(t *testing.T) {
	cmd := NewDoctorCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	// Doctor returns a non-nil error if checks fail; that's fine for the
	// test — we just verify it ran and produced output.
	_ = cmd.Execute()
	out := buf.String()
	for _, want := range []string{"Config", "Auth", "API", "Gradle", "Java"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunDoctorChecksRendersFlutterStyleOutput(t *testing.T) {
	checks := []doctorCheck{
		{
			name: "Version",
			run: func(context.Context) checkResult {
				return checkResult{name: "Version", status: statusOK, summary: "Grounds CLI 0.1.13 is up to date"}
			},
		},
		{
			name: "Auth",
			run: func(context.Context) checkResult {
				return checkResult{
					name:    "Auth",
					status:  statusWarn,
					summary: "You are not logged in",
					details: []string{"Run `grounds login` to authenticate."},
				}
			},
		},
	}

	var buf bytes.Buffer
	err := runDoctorChecks(context.Background(), &buf, checks, false, false)
	if err != nil {
		t.Fatalf("warning result should not fail doctor: %v", err)
	}

	want := "Doctor summary:\n\n" +
		"[✓] Version - Grounds CLI 0.1.13 is up to date\n" +
		"[!] Auth - You are not logged in\n" +
		"    ! Run `grounds login` to authenticate.\n\n" +
		"! Doctor found warnings in 1 category.\n"
	if buf.String() != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestCheckVersionWarnsWhenUpdateAvailable(t *testing.T) {
	oldCheckLatest := doctorCheckLatest
	doctorCheckLatest = func(ctx context.Context, opts version.CheckOptions) (version.CheckReport, error) {
		return version.CheckReport{
			Current:         "0.1.12",
			Latest:          "0.1.13",
			UpdateAvailable: true,
		}, nil
	}
	defer func() { doctorCheckLatest = oldCheckLatest }()

	result := checkVersion(context.Background())
	if result.status != statusWarn {
		t.Fatalf("expected version warning: %#v", result)
	}
	combined := result.summary + "\n" + strings.Join(result.details, "\n")
	for _, want := range []string{"Grounds CLI 0.1.12 is outdated; latest is 0.1.13", "Run `grounds version --check` for update instructions."} {
		if !strings.Contains(combined, want) {
			t.Errorf("message missing %q: %s", want, combined)
		}
	}
}

func TestCheckVersionReportsLocalBuildWithoutWarning(t *testing.T) {
	oldCheckLatest := doctorCheckLatest
	doctorCheckLatest = func(ctx context.Context, opts version.CheckOptions) (version.CheckReport, error) {
		return version.CheckReport{
			Current:    "dev",
			Latest:     "0.1.13",
			Comparable: false,
		}, nil
	}
	defer func() { doctorCheckLatest = oldCheckLatest }()

	result := checkVersion(context.Background())
	if result.status != statusOK {
		t.Fatalf("expected local build to be OK, got %#v", result)
	}
	if result.summary != "Grounds CLI dev is a local build (latest release is 0.1.13)" {
		t.Fatalf("unexpected summary: %s", result.summary)
	}
}

func TestRunDoctorChecksReportsErrorsWithoutCommandError(t *testing.T) {
	checks := []doctorCheck{
		{
			name: "Gradle",
			run: func(context.Context) checkResult {
				return checkResult{
					name:    "Gradle",
					status:  statusError,
					summary: "Gradle wrapper was not found in this directory",
					details: []string{"Run `grounds doctor` from your project root, where `./gradlew` exists."},
				}
			},
		},
	}

	var buf bytes.Buffer
	err := runDoctorChecks(context.Background(), &buf, checks, false, false)
	if err != nil {
		t.Fatalf("doctor findings should not fail by default: %v", err)
	}
	for _, want := range []string{"[✗] Gradle - Gradle wrapper was not found in this directory", "    ✗ Run `grounds doctor` from your project root, where `./gradlew` exists.", "✗ Doctor found issues in 1 category."} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output missing %q:\n%s", want, buf.String())
		}
	}
}

func TestRunDoctorChecksStrictFailsForErrorStatus(t *testing.T) {
	checks := []doctorCheck{
		{
			name: "Gradle",
			run: func(context.Context) checkResult {
				return checkResult{name: "Gradle", status: statusError, summary: "Gradle wrapper was not found in this directory"}
			},
		},
	}

	var buf bytes.Buffer
	err := runDoctorChecks(context.Background(), &buf, checks, false, true)
	if err != ErrDoctorIssuesFound {
		t.Fatalf("expected strict doctor sentinel error, got %v", err)
	}
}

func TestCheckAPISummaryIsUserFriendly(t *testing.T) {
	result := checkAPIResult("https://api.grounds.gg", http.StatusOK, nil)
	if result.summary != "Connected to the Grounds API at https://api.grounds.gg (/healthz returned HTTP 200)" {
		t.Fatalf("unexpected API summary: %s", result.summary)
	}
	if len(result.details) != 0 {
		t.Fatalf("OK result should not include detail lines: %#v", result.details)
	}
}
