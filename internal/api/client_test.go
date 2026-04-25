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
