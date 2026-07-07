package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/groundsgg/grounds-cli/internal/auth"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
	Tokens  TokenSource
	// ProjectID, when set, is appended as `?projectId=...` to every
	// project-scoped request. Command handlers resolve it from --project,
	// GROUNDS_PROJECT, saved project defaults, or the account's only
	// project before making project-scoped calls.
	ProjectID string
}

// WithProject returns a copy of the Client scoped to the given project id.
// Used by command handlers that resolve --project before each call so the
// underlying base client can be shared across goroutines.
func (c *Client) WithProject(id string) *Client {
	clone := *c
	clone.ProjectID = id
	return &clone
}

// ScopedURL returns an absolute API URL with the client's project scope
// applied to the path.
func (c *Client) ScopedURL(path string) string {
	return c.BaseURL + c.scopedPath(path)
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
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+c.scopedPath(path), rdr)
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

// scopedPath appends `?projectId=...` to the path when the client carries
// a project id. Idempotent: does nothing when ProjectID is empty or when the
// path already contains a `projectId=` query.
func (c *Client) scopedPath(path string) string {
	if c.ProjectID == "" {
		return path
	}
	// already scoped
	if strings.Contains(path, "projectId=") {
		return path
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + "projectId=" + url.QueryEscape(c.ProjectID)
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
