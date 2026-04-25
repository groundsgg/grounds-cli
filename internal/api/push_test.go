package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
