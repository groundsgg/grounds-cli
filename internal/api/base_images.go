package api

import (
	"context"
	"net/http"
)

type BaseImageCatalog struct {
	Items []BaseImageSource `json:"items"`
}

type BaseImageSource struct {
	Key            string             `json:"key"`
	DisplayName    string             `json:"displayName"`
	ManifestType   string             `json:"manifestType"`
	Image          string             `json:"image"`
	DefaultChannel string             `json:"defaultChannel"`
	Channels       []BaseImageChannel `json:"channels"`
	Versions       []BaseImageVersion `json:"versions"`
}

type BaseImageChannel struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type BaseImageVersion struct {
	Version    string  `json:"version"`
	Tag        string  `json:"tag"`
	Digest     *string `json:"digest"`
	State      string  `json:"state"`
	Selectable bool    `json:"selectable"`
}

func (c *Client) ListBaseImages(ctx context.Context) (*BaseImageCatalog, error) {
	var out BaseImageCatalog
	if err := c.doRequest(ctx, http.MethodGet, "/v1/base-images", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
