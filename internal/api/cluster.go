package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ClusterStatus mirrors the GET /v1/cluster response shape from
// groundsgg/grounds-platform docs/specs/2026-04-25-namespace-lifecycle.md §7.
type ClusterStatus struct {
	Namespace        string             `json:"namespace"`
	State            string             `json:"state"`
	Profile          string             `json:"profile"`
	CreatedAt        time.Time          `json:"createdAt"`
	LastActivityAt   time.Time          `json:"lastActivityAt"`
	PausedAt         *time.Time         `json:"pausedAt"`
	PauseScheduledAt *time.Time         `json:"pauseScheduledAt"`
	WarningAt        *time.Time         `json:"warningAt"`
	AutoPauseAt      *time.Time         `json:"autoPauseAt"`
	AutoDeleteAt     *time.Time         `json:"autoDeleteAt"`
	Quota            map[string]string  `json:"quota"`
	DeploymentsReady int                `json:"deploymentsReady"`
}

func (c *Client) GetCluster(ctx context.Context) (*ClusterStatus, error) {
	out := &ClusterStatus{}
	if err := c.doRequest(ctx, http.MethodGet, "/v1/cluster", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ClusterUp(ctx context.Context) (*ClusterStatus, error) {
	out := &ClusterStatus{}
	if err := c.doRequest(ctx, http.MethodPost, "/v1/cluster/up", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ClusterDown(ctx context.Context) (*ClusterStatus, error) {
	out := &ClusterStatus{}
	if err := c.doRequest(ctx, http.MethodPost, "/v1/cluster/down", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterDelete sends DELETE /v1/cluster with the X-Confirm-Delete
// header set to namespace. Returns the deletion outcome.
type ClusterDeleteResult struct {
	State string `json:"state"`
	Poll  string `json:"poll,omitempty"`
}

func (c *Client) ClusterDelete(ctx context.Context, namespace string) (*ClusterDeleteResult, error) {
	// We can't reuse doRequest because we need a custom header. Inline.
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+c.scopedPath("/v1/cluster"), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Confirm-Delete", namespace)

	if c.Tokens != nil {
		tok, err := c.Tokens.Token(ctx)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, parseError(resp)
	}
	out := &ClusterDeleteResult{}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return out, nil
}
