package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	// CodeVerifier is the PKCE secret generated client-side by
	// StartDevice. The caller must pass it back to PollToken so the
	// token endpoint can validate the challenge that was bound to the
	// device_code at request time. Not part of the OIDC response —
	// stitched into the struct here so the device-flow API stays
	// stateless across the two RPCs.
	CodeVerifier string `json:"-"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	IDToken          string `json:"id_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
}

type DeviceClient struct {
	Issuer   string       // e.g. "https://account.grounds.gg/realms/grounds"
	ClientID string       // "grounds-cli"
	HTTP     *http.Client
}

func (d *DeviceClient) StartDevice(ctx context.Context) (*DeviceCodeResponse, error) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		return nil, fmt.Errorf("pkce: %w", err)
	}
	body := url.Values{
		"client_id":             {d.ClientID},
		"scope":                 {"openid profile email"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	req, _ := http.NewRequestWithContext(ctx, "POST",
		d.Issuer+"/protocol/openid-connect/auth/device",
		strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device endpoint %d: %s", resp.StatusCode, raw)
	}
	out := &DeviceCodeResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	if out.Interval == 0 {
		out.Interval = 5
	}
	out.CodeVerifier = verifier
	return out, nil
}

// newPKCE generates an RFC 7636 PKCE pair: a 43-character URL-safe
// random verifier and its SHA-256 challenge. Keycloak's device-flow
// implementation requires both since the recent enforcement of
// `code_challenge_method` on the device endpoint.
func newPKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// PollToken loops until the user authorises in the browser, the device
// code expires, or ctx is cancelled. `codeVerifier` is the PKCE secret
// returned by StartDevice — passing it here lets Keycloak validate the
// challenge that was bound to the device_code. Returns the token
// response on success.
func (d *DeviceClient) PollToken(ctx context.Context, deviceCode, codeVerifier string, interval, expiresIn int) (*TokenResponse, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	tick := time.NewTicker(time.Duration(interval) * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tick.C:
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("device authorisation expired; run 'grounds login' again")
		}

		body := url.Values{
			"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code":   {deviceCode},
			"client_id":     {d.ClientID},
			"code_verifier": {codeVerifier},
		}
		req, _ := http.NewRequestWithContext(ctx, "POST",
			d.Issuer+"/protocol/openid-connect/token",
			strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := d.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			out := &TokenResponse{}
			if err := json.Unmarshal(raw, out); err != nil {
				return nil, err
			}
			return out, nil
		}

		var oerr struct{ Error, ErrorDescription string }
		_ = json.Unmarshal(raw, &oerr)
		switch oerr.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			tick.Reset(time.Duration(interval+5) * time.Second)
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired; run 'grounds login' again")
		case "access_denied":
			return nil, fmt.Errorf("authorisation denied")
		default:
			return nil, fmt.Errorf("token endpoint %d: %s", resp.StatusCode, string(raw))
		}
	}
}
