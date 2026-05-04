package commands

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/groundsgg/grounds-cli/internal/version"
)

func TestVersionCommand(t *testing.T) {
	version.Version = "0.1.0"
	version.Commit = "abc123"
	version.BuildAt = "2026-04-25T00:00:00Z"

	cmd := NewVersionCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"0.1.0", "abc123", "2026-04-25"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestVersionCommandCheckReportsUpdate(t *testing.T) {
	version.Version = "0.1.12"
	version.Commit = "abc123"
	version.BuildAt = "2026-04-25T00:00:00Z"

	oldClient := versionCheckHTTPClient
	versionCheckHTTPClient = &http.Client{Transport: commandRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v0.1.13","html_url":"https://github.com/groundsgg/grounds-cli/releases/tag/v0.1.13"}`)),
		}, nil
	})}
	defer func() { versionCheckHTTPClient = oldClient }()

	cmd := NewVersionCommand()
	cmd.SetArgs([]string{"--check"})
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Flags().Set("release-api-url", "https://api.github.test"); err != nil {
		t.Fatalf("set release api url: %v", err)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"grounds version 0.1.12",
		"latest: 0.1.13",
		"status: update available",
		"https://github.com/groundsgg/grounds-cli/releases/tag/v0.1.13",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

type commandRoundTripFunc func(*http.Request) (*http.Response, error)

func (f commandRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
