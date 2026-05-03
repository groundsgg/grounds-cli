package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// BundleRelease mirrors one entry of GET /v1/bundle/releases.
type BundleRelease struct {
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"publishedAt"`
	HtmlURL     string    `json:"htmlUrl"`
	IsLatest    bool      `json:"isLatest"`
}

type bundleReleasesResponse struct {
	Releases []BundleRelease `json:"releases"`
}

// ListBundleReleases returns released library-platform-bundle versions
// (drafts + prereleases filtered out, newest-first).
func (c *Client) ListBundleReleases(ctx context.Context) ([]BundleRelease, error) {
	out := &bundleReleasesResponse{}
	if err := c.doRequest(ctx, http.MethodGet, "/v1/bundle/releases", nil, out); err != nil {
		return nil, err
	}
	return out.Releases, nil
}

// BundleSummary is the parsed bundle.yaml as JSON. The CLI doesn't
// need every field — Components is the most useful surface; the rest
// is opaque-passthrough so we don't have to keep this in sync with
// every bundle-schema bump.
type BundleSummary struct {
	APIVersion string                              `json:"apiVersion"`
	Kind       string                              `json:"kind"`
	Metadata   BundleSummaryMetadata               `json:"metadata"`
	Shared     map[string]any                      `json:"shared"`
	Components map[string]BundleSummaryComponent   `json:"components"`
}

type BundleSummaryMetadata struct {
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type BundleSummaryComponent struct {
	Type     string `json:"type"`
	Image    string `json:"image"`
	Version  string `json:"version"`
	Optional bool   `json:"optional,omitempty"`
	Chart    struct {
		URL     string `json:"url"`
		Version string `json:"version"`
	} `json:"chart"`
	Devspace *struct {
		Workflow string `json:"workflow,omitempty"`
	} `json:"devspace,omitempty"`
}

// GetBundle returns the parsed bundle.yaml at the given ref. `ref`
// accepts the same shapes loadBundle does — semver, "v…", the full
// tag, or "main".
func (c *Client) GetBundle(ctx context.Context, ref string) (*BundleSummary, error) {
	out := &BundleSummary{}
	path := "/v1/bundle/" + url.PathEscape(ref)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
