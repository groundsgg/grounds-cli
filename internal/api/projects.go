package api

import (
	"context"
	"net/http"
)

type Project struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type ProjectList struct {
	Items []Project `json:"items"`
}

func (c *Client) ListProjects(ctx context.Context) (*ProjectList, error) {
	out := &ProjectList{}
	base := c.WithProject("")
	if err := base.doRequest(ctx, http.MethodGet, "/v1/projects", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
