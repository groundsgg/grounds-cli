package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

type CreatePushRequest struct {
	Target                 string
	Flavor                 string
	Force                  bool
	Manifest               any
	EffectivePluginSources any
	ArtifactPath           string
}

type CreatePushResponse struct {
	PushID    string `json:"pushId"`
	Status    string `json:"status"`
	Reused    bool   `json:"reused,omitempty"`
	FlavorKey string `json:"flavorKey,omitempty"`
	LogsURL   string `json:"logsUrl,omitempty"`
}

func (c *Client) CreatePush(ctx context.Context, req CreatePushRequest) (*CreatePushResponse, error) {
	if req.ArtifactPath == "" {
		return nil, fmt.Errorf("artifact path is required")
	}

	var tok string
	useAuth := c.Tokens != nil
	if c.Tokens != nil {
		var err error
		tok, err = c.Tokens.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
	}

	manifestRaw, err := json.Marshal(req.Manifest)
	if err != nil {
		return nil, err
	}
	var effectivePluginSourcesRaw []byte
	if req.EffectivePluginSources != nil {
		effectivePluginSourcesRaw, err = json.Marshal(req.EffectivePluginSources)
		if err != nil {
			return nil, err
		}
	}

	file, err := os.Open(req.ArtifactPath)
	if err != nil {
		return nil, err
	}

	path := "/v1/pushes"
	if req.Force {
		path += "?force=true"
	}
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	contentType := writer.FormDataContentType()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+c.scopedPath(path), pr)
	if err != nil {
		_ = file.Close()
		_ = pr.Close()
		_ = pw.Close()
		return nil, err
	}
	httpReq.Header.Set("Content-Type", contentType)
	if useAuth {
		httpReq.Header.Set("Authorization", "Bearer "+tok)
	}

	writeErrCh := make(chan error, 1)
	go func() {
		writeErrCh <- writeCreatePushMultipart(pw, writer, file, req, manifestRaw, effectivePluginSourcesRaw)
	}()

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		_ = pr.CloseWithError(err)
		if writeErr := <-writeErrCh; writeErr != nil && !errors.Is(writeErr, err) && !errors.Is(writeErr, io.ErrClosedPipe) {
			return nil, writeErr
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		err := parseError(resp)
		_ = pr.CloseWithError(err)
		if writeErr := <-writeErrCh; writeErr != nil && !errors.Is(writeErr, err) && !errors.Is(writeErr, io.ErrClosedPipe) {
			return nil, writeErr
		}
		return nil, err
	}
	if writeErr := <-writeErrCh; writeErr != nil {
		return nil, writeErr
	}
	out := &CreatePushResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}

func writeCreatePushMultipart(
	pw *io.PipeWriter,
	writer *multipart.Writer,
	file *os.File,
	req CreatePushRequest,
	manifestRaw []byte,
	effectivePluginSourcesRaw []byte,
) (err error) {
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if closeErr := writer.Close(); closeErr != nil {
			err = closeErr
			_ = pw.CloseWithError(closeErr)
			return
		}
		if closeErr := pw.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if err := writer.WriteField("target", req.Target); err != nil {
		return err
	}
	if req.Flavor != "" {
		if err := writer.WriteField("flavor", req.Flavor); err != nil {
			return err
		}
	}
	if err := writer.WriteField("manifest", string(manifestRaw)); err != nil {
		return err
	}
	if req.EffectivePluginSources != nil {
		if err := writer.WriteField("effectivePluginSources", string(effectivePluginSourcesRaw)); err != nil {
			return err
		}
	}
	part, err := writer.CreateFormFile("jar", filepath.Base(req.ArtifactPath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	return nil
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
