package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultReleaseAPIBaseURL = "https://api.github.com"
	releasePath              = "/repos/groundsgg/grounds-cli/releases/latest"
)

type CheckOptions struct {
	Current    string
	APIBaseURL string
	HTTPClient *http.Client
}

type CheckReport struct {
	Current         string
	Latest          string
	ReleaseURL      string
	Comparable      bool
	UpdateAvailable bool
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func CheckLatest(ctx context.Context, opts CheckOptions) (CheckReport, error) {
	current := Normalize(opts.Current)
	if current == "" {
		current = Normalize(Version)
	}

	apiBaseURL := strings.TrimRight(opts.APIBaseURL, "/")
	if apiBaseURL == "" {
		apiBaseURL = DefaultReleaseAPIBaseURL
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+releasePath, nil)
	if err != nil {
		return CheckReport{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "grounds-cli/"+current)

	resp, err := client.Do(req)
	if err != nil {
		return CheckReport{}, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CheckReport{}, fmt.Errorf("failed to fetch latest release: status=%d", resp.StatusCode)
	}

	var latest latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return CheckReport{}, fmt.Errorf("failed to decode latest release: %w", err)
	}

	latestVersion := Normalize(latest.TagName)
	if latestVersion == "" {
		return CheckReport{}, fmt.Errorf("invalid latest release response: missing tag_name")
	}

	comparable := Comparable(current, latestVersion)
	return CheckReport{
		Current:         current,
		Latest:          latestVersion,
		ReleaseURL:      latest.HTMLURL,
		Comparable:      comparable,
		UpdateAvailable: comparable && Compare(current, latestVersion) < 0,
	}, nil
}

func Normalize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

func Compare(current, latest string) int {
	current = Normalize(current)
	latest = Normalize(latest)

	currentParts, currentOK := parseVersion(current)
	latestParts, latestOK := parseVersion(latest)
	if !currentOK && !latestOK {
		return strings.Compare(current, latest)
	}
	if !currentOK || !latestOK {
		return 0
	}

	for i := range currentParts {
		if currentParts[i] < latestParts[i] {
			return -1
		}
		if currentParts[i] > latestParts[i] {
			return 1
		}
	}
	return 0
}

func Comparable(current, latest string) bool {
	current = Normalize(current)
	latest = Normalize(latest)
	_, currentOK := parseVersion(current)
	_, latestOK := parseVersion(latest)
	return currentOK == latestOK
}

func parseVersion(v string) ([3]int, bool) {
	var parts [3]int
	fields := strings.Split(v, ".")
	if len(fields) != 3 {
		return parts, false
	}
	for i, field := range fields {
		if field == "" {
			return parts, false
		}
		for _, r := range field {
			if r < '0' || r > '9' {
				return parts, false
			}
			parts[i] = parts[i]*10 + int(r-'0')
		}
	}
	return parts, true
}
