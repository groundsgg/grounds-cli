package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DevspaceGenerateRequest mirrors the body shape forge accepts on
// POST /v1/devspace/generate.
type DevspaceGenerateRequest struct {
	Bundle        string `json:"bundle"`
	ComponentName string `json:"componentName"`
	// Override is opaque to the CLI — forge owns the per-component
	// override union (image | gradle-local | enabled:false). The CLI
	// just lifts it from the engineer's override-file and passes it
	// through.
	Override map[string]any `json:"override,omitempty"`
}

// DevspaceGenerate POSTs the request and returns the substituted YAML
// as bytes. Unlike most cluster routes it returns text/yaml rather
// than JSON — we keep the call inline rather than threading a raw
// body through doRequest.
func (c *Client) DevspaceGenerate(ctx context.Context, body *DevspaceGenerateRequest) ([]byte, error) {
	blob, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+c.scopedPath("/v1/devspace/generate"), bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/yaml")

	if c.Tokens != nil {
		tok, err := c.Tokens.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
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
	return io.ReadAll(resp.Body)
}
