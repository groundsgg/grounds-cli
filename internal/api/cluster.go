package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// ClusterStatus mirrors the GET /v1/cluster response shape from
// groundsgg/grounds-platform docs/specs/2026-04-25-namespace-lifecycle.md §7.
type ClusterStatus struct {
	Namespace        string            `json:"namespace"`
	State            string            `json:"state"`
	Profile          string            `json:"profile"`
	CreatedAt        time.Time         `json:"createdAt"`
	LastActivityAt   time.Time         `json:"lastActivityAt"`
	PausedAt         *time.Time        `json:"pausedAt"`
	PauseScheduledAt *time.Time        `json:"pauseScheduledAt"`
	WarningAt        *time.Time        `json:"warningAt"`
	AutoPauseAt      *time.Time        `json:"autoPauseAt"`
	AutoDeleteAt     *time.Time        `json:"autoDeleteAt"`
	Quota            map[string]string `json:"quota"`
	DeploymentsReady int               `json:"deploymentsReady"`
	// platform-bundle async-provision surface (forge#299): the last apply's
	// component breakdown and, when state=failed, the reconcile error.
	BundleResult   *BundleResult   `json:"bundleResult"`
	BundleProgress *BundleProgress `json:"bundleProgress"`
	// Drift check: forge compares the components it installed against what is
	// actually running in the vCluster. Nil unless the workspace is an active
	// platform-bundle one that has had a successful apply.
	BundleHealth  *BundleHealth `json:"bundleHealth"`
	FailureReason string        `json:"failureReason"`
}

// BundleHealth reports whether an `active` workspace still contains the bundle
// components forge installed into it. A workspace can lose them without forge
// noticing (provisioning is request-driven, never drift-correcting), which used
// to surface as a healthy-looking workspace with no apps in it.
type BundleHealth struct {
	// "ok" | "degraded" | "unreachable"
	Status string `json:"status"`
	// Components from the last successful apply that are no longer running.
	Missing []string `json:"missing"`
}

// BundleResult is the { succeeded, failed } breakdown of a bundle apply,
// surfaced on GET /v1/cluster so a poller can report the outcome of an
// async (202-then-poll) `cluster up --bundle`.
type BundleResult struct {
	Succeeded []string `json:"succeeded"`
	Failed    []struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	} `json:"failed"`
}

type BundleProgress struct {
	BundleRef            string `json:"bundleRef"`
	Phase                string `json:"phase"`
	Message              string `json:"message"`
	CurrentComponent     string `json:"currentComponent"`
	CurrentComponentType string `json:"currentComponentType"`
	CurrentComponentMode string `json:"currentComponentMode"`
	UpdatedAt            string `json:"updatedAt"`
	ComponentsTotal      int    `json:"componentsTotal"`
	ComponentsDone       int    `json:"componentsDone"`
	ComponentsSucceeded  int    `json:"componentsSucceeded"`
	ComponentsFailed     int    `json:"componentsFailed"`
}

func (c *Client) GetCluster(ctx context.Context) (*ClusterStatus, error) {
	out := &ClusterStatus{}
	if err := c.doRequest(ctx, http.MethodGet, "/v1/cluster", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterUp spawns or resumes the workspace. `profile` may be empty
// (forge picks the default) or "platform". Forge enforces
// the immutable-profile rule: switching profiles on an existing
// workspace requires `grounds cluster delete` first.
func (c *Client) ClusterUp(ctx context.Context, profile string) (*ClusterStatus, error) {
	path := "/v1/cluster/up"
	if profile != "" {
		path += "?profile=" + url.QueryEscape(profile)
	}
	out := &ClusterStatus{}
	if err := c.doRequest(ctx, http.MethodPost, path, nil, out); err != nil {
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

// BundleUpRequest is the body shape forge expects on POST /v1/cluster/bundle.
// `Overrides` is intentionally a free-form map: forge validates the
// per-component union (image | gradle-local | enabled:false) — the CLI
// just passes through whatever YAML the engineer wrote.
type BundleUpRequest struct {
	Bundle    string         `json:"bundle"`
	Overrides map[string]any `json:"overrides,omitempty"`
}

// BundleUpResult mirrors the success body of POST /v1/cluster/bundle.
type BundleUpResult struct {
	State         string `json:"state"`
	DevClusterID  string `json:"devClusterId"`
	Namespace     string `json:"namespace"`
	Created       bool   `json:"created"`
	BundleVersion string `json:"bundleVersion"`
	Components    struct {
		Resolved  int      `json:"resolved"`
		Succeeded []string `json:"succeeded"`
		Failed    []struct {
			Name  string `json:"name"`
			Error string `json:"error"`
		} `json:"failed"`
	} `json:"components"`
}

// BundleComponentRestoreResult mirrors the asynchronous response from
// POST /v1/cluster/bundle/components/:name/restore.
type BundleComponentRestoreResult struct {
	State string `json:"state"`
	Poll  string `json:"poll"`
}

// RestoreBundleComponent removes a temporary push replacement and reconciles
// the named component back to the image pinned by the active bundle reference.
func (c *Client) RestoreBundleComponent(ctx context.Context, component string) (*BundleComponentRestoreResult, error) {
	out := &BundleComponentRestoreResult{}
	path := "/v1/cluster/bundle/components/" + url.PathEscape(component) + "/restore"
	if err := c.doRequest(ctx, http.MethodPost, path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterUpBundle drives a `platform-bundle` profile DevCluster: forge
// resolves the bundle ref, applies the engineer's overrides, and
// best-effort helm-installs each component into the vCluster. The
// breakdown of succeeded/failed components is in the result.
func (c *Client) ClusterUpBundle(ctx context.Context, body *BundleUpRequest) (*BundleUpResult, error) {
	out := &BundleUpResult{}
	if err := c.doRequest(ctx, http.MethodPost, "/v1/cluster/bundle", body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterResetRequest is the (optional) body for POST /v1/cluster/reset.
// An empty Bundle lets forge pick the default ref (main).
type ClusterResetRequest struct {
	Bundle string `json:"bundle,omitempty"`
}

// ClusterResetResult mirrors the 202 body of POST /v1/cluster/reset.
type ClusterResetResult struct {
	State string `json:"state"`
	Poll  string `json:"poll,omitempty"`
}

// ClusterReset wipes a platform-bundle workspace back to a clean bundle base:
// forge tears down the vCluster namespace and re-provisions from the bundle ref
// (body.Bundle ?? "main") with no engineer overrides. Returns 202 immediately;
// the caller polls GET /v1/cluster until the workspace is active again.
func (c *Client) ClusterReset(ctx context.Context, body *ClusterResetRequest) (*ClusterResetResult, error) {
	out := &ClusterResetResult{}
	if err := c.doRequest(ctx, http.MethodPost, "/v1/cluster/reset", body, out); err != nil {
		return nil, err
	}
	return out, nil
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
