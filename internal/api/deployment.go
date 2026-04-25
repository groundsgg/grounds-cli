package api

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type Deployment struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Type           string     `json:"type"`
	ImageTag       string     `json:"imageTag"`
	PublicURL      string     `json:"publicUrl"`
	State          string     `json:"state"`
	LastDeployedAt time.Time  `json:"lastDeployedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

type DeploymentList struct {
	Items      []Deployment `json:"items"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

func (c *Client) ListDeployments(ctx context.Context) (*DeploymentList, error) {
	out := &DeploymentList{}
	if err := c.doRequest(ctx, http.MethodGet, "/v1/deployments", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) DeleteDeployment(ctx context.Context, name string) error {
	return c.doRequest(ctx, http.MethodDelete,
		"/v1/deployments/"+url.PathEscape(name), nil, nil)
}
