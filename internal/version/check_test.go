package version

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheckLatestReportsUpdateAvailable(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/repos/groundsgg/grounds-cli/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return jsonResponse(200, `{"tag_name":"v0.1.13","html_url":"https://github.com/groundsgg/grounds-cli/releases/tag/v0.1.13"}`), nil
	})}

	report, err := CheckLatest(context.Background(), CheckOptions{
		Current:    "0.1.12",
		APIBaseURL: "https://api.github.test",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("check latest: %v", err)
	}

	if !report.UpdateAvailable {
		t.Fatalf("expected update to be available: %#v", report)
	}
	if !report.Comparable {
		t.Fatalf("expected release versions to be comparable: %#v", report)
	}
	if report.Current != "0.1.12" || report.Latest != "0.1.13" {
		t.Fatalf("unexpected versions: %#v", report)
	}
	if report.ReleaseURL != "https://github.com/groundsgg/grounds-cli/releases/tag/v0.1.13" {
		t.Fatalf("unexpected release url: %s", report.ReleaseURL)
	}
}

func TestCheckLatestReportsLocalBuildAsIncomparable(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"tag_name":"v0.1.13"}`), nil
	})}

	report, err := CheckLatest(context.Background(), CheckOptions{
		Current:    "dev",
		APIBaseURL: "https://api.github.test",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("check latest: %v", err)
	}
	if report.Comparable {
		t.Fatalf("expected local build to be incomparable: %#v", report)
	}
	if report.UpdateAvailable {
		t.Fatalf("incomparable local build should not report update available: %#v", report)
	}
}

func TestCheckLatestReportsUpToDate(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"tag_name":"v0.1.13"}`), nil
	})}

	report, err := CheckLatest(context.Background(), CheckOptions{
		Current:    "0.1.13",
		APIBaseURL: "https://api.github.test",
		HTTPClient: client,
	})
	if err != nil {
		t.Fatalf("check latest: %v", err)
	}

	if report.UpdateAvailable {
		t.Fatalf("expected current version to be up to date: %#v", report)
	}
}

func TestCheckLatestRejectsInvalidReleaseResponse(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"tag_name":""}`), nil
	})}

	_, err := CheckLatest(context.Background(), CheckOptions{
		Current:    "0.1.13",
		APIBaseURL: "https://api.github.test",
		HTTPClient: client,
	})
	if err == nil {
		t.Fatal("expected invalid release response error")
	}
	if !strings.Contains(err.Error(), "missing tag_name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
	}{
		{name: "latest patch newer", current: "0.1.12", latest: "0.1.13", want: -1},
		{name: "same version", current: "v0.1.13", latest: "0.1.13", want: 0},
		{name: "current newer", current: "0.2.0", latest: "0.1.13", want: 1},
		{name: "dev current is not comparable", current: "dev", latest: "0.1.13", want: 0},
		{name: "git describe current is not comparable", current: "284e1b8-dirty", latest: "0.1.13", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(tt.current, tt.latest)
			if got != tt.want {
				t.Fatalf("Compare(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
