package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeTS struct{ tok string; err error }

func (f *fakeTS) Token(_ context.Context) (string, error) { return f.tok, f.err }

func TestDoRequest_AuthHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
	}))
	defer srv.Close()
	c := New(srv.URL, &fakeTS{tok: "abc"})
	out := map[string]string{}
	if err := c.doRequest(context.Background(), "GET", "/x", nil, &out); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "Bearer abc" {
		t.Errorf("Authorization = %q", got)
	}
	if out["hello"] != "world" {
		t.Errorf("decoded = %v", out)
	}
}

func TestScopedPath(t *testing.T) {
	cases := []struct {
		name      string
		projectID string
		in        string
		want      string
	}{
		{name: "empty project leaves path untouched", projectID: "", in: "/v1/cluster", want: "/v1/cluster"},
		{name: "appends as first query param", projectID: "p1", in: "/v1/cluster", want: "/v1/cluster?projectId=p1"},
		{name: "appends with & when path already has a query", projectID: "p1", in: "/v1/pushes?cursor=x", want: "/v1/pushes?cursor=x&projectId=p1"},
		{name: "url-escapes the project id", projectID: "with spaces", in: "/v1/cluster", want: "/v1/cluster?projectId=with+spaces"},
		{name: "no double-append when projectId already present", projectID: "p1", in: "/v1/cluster?projectId=p2", want: "/v1/cluster?projectId=p2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{ProjectID: tc.projectID}
			got := c.scopedPath(tc.in)
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestProjectIDPropagatedThroughDoRequest(t *testing.T) {
	var seenURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenURL = r.URL.String()
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil)
	c.ProjectID = "p-test"
	if _, err := c.GetCluster(context.Background()); err != nil {
		t.Fatalf("GetCluster: %v", err)
	}
	if seenURL != "/v1/cluster?projectId=p-test" {
		t.Errorf("server saw %q", seenURL)
	}
}

func TestWithProjectClonesIndependently(t *testing.T) {
	base := &Client{BaseURL: "x", ProjectID: ""}
	scoped := base.WithProject("p1")
	if base.ProjectID != "" {
		t.Errorf("WithProject mutated base.ProjectID = %q", base.ProjectID)
	}
	if scoped.ProjectID != "p1" {
		t.Errorf("scoped.ProjectID = %q", scoped.ProjectID)
	}
}

func TestDoRequest_ErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "in_flight_push"})
	}))
	defer srv.Close()
	c := New(srv.URL, nil)
	err := c.doRequest(context.Background(), "POST", "/x", nil, nil)
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if ExitCode(err) != 6 {
		t.Errorf("ExitCode = %d", ExitCode(err))
	}
	if e.Code != "in_flight_push" {
		t.Errorf("Code = %q", e.Code)
	}
}
