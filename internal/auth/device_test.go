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
	var gotChallenge, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/auth/device") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		r.ParseForm()
		gotChallenge = r.Form.Get("code_challenge")
		gotMethod = r.Form.Get("code_challenge_method")
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
	// PKCE — Keycloak's device endpoint requires this since recent
	// versions; without it the request fails 400 invalid_request.
	if gotMethod != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", gotMethod)
	}
	if gotChallenge == "" {
		t.Error("code_challenge missing on /auth/device request")
	}
	if res.CodeVerifier == "" {
		t.Error("CodeVerifier should be populated for PollToken")
	}
}

func TestPollToken_Success(t *testing.T) {
	call := 0
	var gotVerifier string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		r.ParseForm()
		gotVerifier = r.Form.Get("code_verifier")
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
	tok, err := c.PollToken(ctx, "dc", "verifier-123", 1, 60)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if tok.AccessToken != "at" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if gotVerifier != "verifier-123" {
		t.Errorf("code_verifier = %q, want verifier-123", gotVerifier)
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
