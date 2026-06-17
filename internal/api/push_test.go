package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetryPush(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/pushes/p1/retry" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "p1", "status": "received"})
	}))
	defer srv.Close()
	c := New(srv.URL, nil)
	p, err := c.RetryPush(context.Background(), "p1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if p.ID != "p1" || p.Status != "received" {
		t.Errorf("push = %+v", p)
	}
}

func TestCreatePushSendsMultipartDistribution(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "app.tar.gz")
	artifactBytes := []byte{0x1f, 0x8b, 0x08, 0x00}
	if err := os.WriteFile(artifact, artifactBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(artifact) error = %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/pushes" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("projectId"); got != "project-1" {
			t.Fatalf("projectId = %q", got)
		}
		if got := r.URL.Query().Get("force"); got != "true" {
			t.Fatalf("force = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.FormValue("target"); got != "staging" {
			t.Fatalf("target = %q", got)
		}
		if got := r.FormValue("flavor"); got != "minestom" {
			t.Fatalf("flavor = %q", got)
		}
		if !strings.Contains(r.FormValue("manifest"), `"type":"minestom-server"`) {
			t.Fatalf("manifest = %s", r.FormValue("manifest"))
		}
		file, header, err := r.FormFile("jar")
		if err != nil {
			t.Fatalf("FormFile(jar) error = %v", err)
		}
		defer file.Close()
		if header.Filename != "app.tar.gz" {
			t.Fatalf("filename = %q", header.Filename)
		}
		uploaded, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll(file) error = %v", err)
		}
		if !bytes.Equal(uploaded, artifactBytes) {
			t.Fatalf("uploaded bytes = %x, want %x", uploaded, artifactBytes)
		}
		json.NewEncoder(w).Encode(map[string]any{"pushId": "push-1", "status": "building", "logsUrl": "/v1/pushes/push-1/logs"})
	}))
	defer srv.Close()

	c := New(srv.URL, staticToken("test-token"))
	c.ProjectID = "project-1"
	res, err := c.CreatePush(context.Background(), CreatePushRequest{
		Target: "staging",
		Flavor: "minestom",
		Force:  true,
		Manifest: map[string]any{
			"name":      "minestom-demo",
			"type":      "minestom-server",
			"baseImage": "minestom",
		},
		ArtifactPath: artifact,
	})
	if err != nil {
		t.Fatalf("CreatePush() error = %v", err)
	}
	if res.PushID != "push-1" || res.Status != "building" {
		t.Fatalf("response = %#v", res)
	}
}

func TestCreatePushRequiresArtifactPathBeforeRequest(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := New(srv.URL, nil)
	_, err := c.CreatePush(context.Background(), CreatePushRequest{
		Target: "staging",
		Manifest: map[string]any{
			"type": "minestom-server",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "artifact path is required") {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestCreatePushTokenFailureBeforeRequest(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := New(srv.URL, errorToken{err: errors.New("token failed")})
	_, err := c.CreatePush(context.Background(), CreatePushRequest{
		Target:       "staging",
		Manifest:     map[string]any{"type": "minestom-server"},
		ArtifactPath: filepath.Join(t.TempDir(), "missing.tar.gz"),
	})
	if err == nil || err.Error() != "auth: token failed" {
		t.Fatalf("error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestCreatePushParsesErrorResponse(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "app.tar.gz")
	if err := os.WriteFile(artifact, []byte("artifact"), 0o600); err != nil {
		t.Fatalf("WriteFile(artifact) error = %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "push_conflict",
			"message": "push already running",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, nil)
	_, err := c.CreatePush(context.Background(), CreatePushRequest{
		Target:       "staging",
		Manifest:     map[string]any{"type": "minestom-server"},
		ArtifactPath: artifact,
	})
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error = %T %[1]v", err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Code != "push_conflict" || apiErr.Message != "push already running" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }

type errorToken struct {
	err error
}

func (e errorToken) Token(context.Context) (string, error) { return "", e.err }
