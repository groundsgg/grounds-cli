package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStartDevice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/auth/device") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:              "dc",
			UserCode:                "ABCD-EFGH",
			VerificationURI:         "https://example.test/device",
			VerificationURIComplete: "https://example.test/device?u=ABCD-EFGH",
			Interval:                1,
			ExpiresIn:               60,
		})
	}))
	defer srv.Close()
	c := &DeviceClient{Issuer: srv.URL, ClientID: "grounds-cli", HTTP: srv.Client()}
	res, err := c.StartDevice(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.UserCode != "ABCD-EFGH" {
		t.Errorf("UserCode = %q", res.UserCode)
	}
}

func TestPollToken_Success(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			// First poll: authorization_pending
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		json.NewEncoder(w).Encode(TokenResponse{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 300})
	}))
	defer srv.Close()
	c := &DeviceClient{Issuer: srv.URL, ClientID: "grounds-cli", HTTP: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tok, err := c.PollToken(ctx, "dc", 1, 60)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if tok.AccessToken != "at" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
}

func TestRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "rt-old" {
			t.Errorf("refresh_token = %q", r.Form.Get("refresh_token"))
		}
		json.NewEncoder(w).Encode(TokenResponse{AccessToken: "at-new", RefreshToken: "rt-new", ExpiresIn: 300})
	}))
	defer srv.Close()
	c := &DeviceClient{Issuer: srv.URL, ClientID: "grounds-cli", HTTP: srv.Client()}
	tok, err := c.Refresh(context.Background(), "rt-old")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if tok.AccessToken != "at-new" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
}
