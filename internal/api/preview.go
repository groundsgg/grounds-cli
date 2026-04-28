package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Preview env metadata as returned by forge /v1/preview-envs.
type PreviewEnv struct {
	ID        string             `json:"id"`
	PushID    string             `json:"pushId"`
	Namespace string             `json:"namespace"`
	Hostname  string             `json:"hostname"`
	PublicURL string             `json:"publicUrl"`
	Pinned    bool               `json:"pinned"`
	ExpiresAt *time.Time         `json:"expiresAt,omitempty"`
	DeletedAt *time.Time         `json:"deletedAt,omitempty"`
	Push      PreviewPushSummary `json:"push"`
}

type PreviewPushSummary struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ManifestName string `json:"manifestName"`
	ManifestType string `json:"manifestType"`
}

type PreviewEnvList struct {
	Items []PreviewEnv `json:"items"`
}

func (c *Client) ListPreviewEnvs(ctx context.Context, includeDeleted bool) (*PreviewEnvList, error) {
	q := url.Values{}
	if includeDeleted {
		q.Set("includeDeleted", "true")
	}
	path := "/v1/preview-envs"
	if e := q.Encode(); e != "" {
		path += "?" + e
	}
	out := &PreviewEnvList{}
	if err := c.doRequest(ctx, http.MethodGet, path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetPreviewEnv(ctx context.Context, id string) (*PreviewEnv, error) {
	out := &PreviewEnv{}
	if err := c.doRequest(ctx, http.MethodGet, "/v1/preview-envs/"+url.PathEscape(id), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SetPreviewPin flips the pinned flag — pinned envs are skipped by the
// PreviewJanitor TTL sweep and stay alive past expiresAt.
func (c *Client) SetPreviewPin(ctx context.Context, id string, pinned bool) (*PreviewEnv, error) {
	body := map[string]bool{"pinned": pinned}
	out := &PreviewEnv{}
	if err := c.doRequest(ctx, http.MethodPatch, "/v1/preview-envs/"+url.PathEscape(id), body, out); err != nil {
		return nil, err
	}
	return out, nil
}
