package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Push struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	Target        string    `json:"target"`
	Status        string    `json:"status"`
	ContentHash   string    `json:"contentHash"`
	ImageTag      string    `json:"imageTag,omitempty"`
	PublicURL     string    `json:"publicUrl,omitempty"`
	FailureReason string    `json:"failureReason,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type PushList struct {
	Items      []Push `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

func (c *Client) GetPush(ctx context.Context, id string) (*Push, error) {
	out := &Push{}
	if err := c.doRequest(ctx, http.MethodGet, "/v1/pushes/"+url.PathEscape(id), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListPushes(ctx context.Context, mine bool, limit int) (*PushList, error) {
	q := url.Values{}
	if mine {
		// No-op on the server today; reserved for future scope filtering.
		q.Set("mine", "true")
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	path := "/v1/pushes"
	if e := q.Encode(); e != "" {
		path += "?" + e
	}
	out := &PushList{}
	if err := c.doRequest(ctx, http.MethodGet, path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) RetryPush(ctx context.Context, id string) (*Push, error) {
	out := &Push{}
	if err := c.doRequest(ctx, http.MethodPost,
		"/v1/pushes/"+url.PathEscape(id)+"/retry", nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
