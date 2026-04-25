package auth

import (
	"context"
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
	body := url.Values{
		"client_id": {d.ClientID},
		"scope":     {"openid profile email"},
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
	return out, nil
}

// PollToken loops until the user authorises in the browser, the device
// code expires, or ctx is cancelled. Returns the token response on success.
func (d *DeviceClient) PollToken(ctx context.Context, deviceCode string, interval, expiresIn int) (*TokenResponse, error) {
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
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
			"client_id":   {d.ClientID},
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
