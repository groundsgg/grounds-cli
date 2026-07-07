package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProjects(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"p1","slug":"main","name":"Main","role":"owner","createdAt":"2026-07-07T10:00:00.000Z"}]}`))
	}))
	t.Cleanup(server.Close)

	c := New(server.URL, nil)
	c.ProjectID = "ignored"

	got, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if seenPath != "/v1/projects" {
		t.Fatalf("path = %q, want /v1/projects", seenPath)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "p1" || got.Items[0].Slug != "main" {
		t.Fatalf("items = %#v", got.Items)
	}
}
