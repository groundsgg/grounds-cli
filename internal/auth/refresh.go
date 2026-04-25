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

// Refresh exchanges a refresh_token for a fresh access_token. Returns
// the new TokenResponse on success.
func (d *DeviceClient) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	body := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {d.ClientID},
	}
	req, _ := http.NewRequestWithContext(ctx, "POST",
		d.Issuer+"/protocol/openid-connect/token",
		strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := d.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %d %s", resp.StatusCode, raw)
	}
	out := &TokenResponse{}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CredentialsFromToken converts a TokenResponse + IDToken email/preferred
// claims into the on-disk Credentials shape.
func CredentialsFromToken(t *TokenResponse, email, preferred string) *Credentials {
	return &Credentials{
		AccessToken:      t.AccessToken,
		RefreshToken:     t.RefreshToken,
		ExpiresAt:        time.Now().Add(time.Duration(t.ExpiresIn) * time.Second),
		RefreshExpiresAt: time.Now().Add(time.Duration(t.RefreshExpiresIn) * time.Second),
		Email:            email,
		PreferredUser:    preferred,
	}
}
