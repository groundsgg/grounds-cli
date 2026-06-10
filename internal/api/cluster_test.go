package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCluster(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/cluster" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"namespace":        "user-test",
			"state":            "active",
			"profile":          "minigame",
			"deploymentsReady": 2,
			"bundleProgress": map[string]any{
				"bundleRef":            "main",
				"phase":                "deploying_components",
				"message":              "Deploying bundle components",
				"currentComponent":     "plugin-config",
				"currentComponentType": "grpc-service",
				"currentComponentMode": "gradle-local",
				"componentsTotal":      14,
				"componentsDone":       7,
				"componentsSucceeded":  6,
				"componentsFailed":     1,
				"updatedAt":            "2026-06-07T20:45:12.000Z",
			},
		})
	}))
	defer srv.Close()
	c := New(srv.URL, nil)
	s, err := c.GetCluster(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if s.Namespace != "user-test" || s.State != "active" || s.DeploymentsReady != 2 {
		t.Errorf("status = %+v", s)
	}
	if s.BundleProgress == nil {
		t.Fatal("BundleProgress = nil")
	}
	if s.BundleProgress.Phase != "deploying_components" ||
		s.BundleProgress.CurrentComponent != "plugin-config" ||
		s.BundleProgress.ComponentsTotal != 14 ||
		s.BundleProgress.ComponentsDone != 7 {
		t.Errorf("BundleProgress = %+v", s.BundleProgress)
	}
}

func TestClusterDelete_Header(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Confirm-Delete")
		json.NewEncoder(w).Encode(map[string]string{"state": "deleted"})
	}))
	defer srv.Close()
	c := New(srv.URL, nil)
	res, err := c.ClusterDelete(context.Background(), "user-test")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "user-test" {
		t.Errorf("X-Confirm-Delete = %q", got)
	}
	if res.State != "deleted" {
		t.Errorf("State = %q", res.State)
	}
}
