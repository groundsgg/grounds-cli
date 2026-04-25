package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/groundsgg/grounds-cli/internal/auth"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
	Tokens  TokenSource
}

// TokenSource produces a fresh bearer token, refreshing on demand. The
// CLI wires this to read the keyring + refresh through DeviceClient.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

func New(baseURL string, ts TokenSource) *Client {
	return &Client{BaseURL: baseURL, HTTP: &http.Client{}, Tokens: ts}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		blob, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(blob)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Tokens != nil {
		tok, err := c.Tokens.Token(ctx)
		if err != nil {
			return fmt.Errorf("auth: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// envTokenSource uses GROUNDS_TOKEN verbatim, no refresh.
type envTokenSource struct{ token string }

func (e *envTokenSource) Token(_ context.Context) (string, error) { return e.token, nil }

// NewEnvTokenSource returns nil if GROUNDS_TOKEN is unset.
func NewEnvTokenSource() TokenSource {
	if t := auth.EnvToken(); t != "" {
		return &envTokenSource{token: t}
	}
	return nil
}
