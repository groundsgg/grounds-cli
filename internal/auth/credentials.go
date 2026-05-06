package auth

import (
	"encoding/json"
	"fmt"
	"time"
)

// CredentialsVersion is the schema version we write. Must match what
// `grounds-push`'s CredentialResolver accepts (currently 1). Bump in
// lockstep when the schema changes.
const CredentialsVersion = 1

// Credentials matches the schema used by the Gradle plugin
// (groundsgg/grounds-push CredentialResolver.kt). CLI is the canonical
// writer; the Gradle plugin reads it.
type Credentials struct {
	Version          int       `json:"version"`
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	Email            string    `json:"email,omitempty"`
	PreferredUser    string    `json:"preferredUsername,omitempty"`
}

// Marshal enforces the current schema version on every write so callers
// don't have to remember to set it. Files written before this field
// existed are silently upgraded on the next save (see ParseCredentials).
func (c *Credentials) Marshal() ([]byte, error) {
	c.Version = CredentialsVersion
	return json.MarshalIndent(c, "", "  ")
}

func ParseCredentials(b []byte) (*Credentials, error) {
	c := &Credentials{}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	// Legacy files written before the version field existed parse with
	// Version == 0. Treat as v1 — they'll be re-written with the correct
	// version on the next refresh / login.
	if c.Version == 0 {
		c.Version = CredentialsVersion
	}
	return c, nil
}

// RefreshExpiryFromSeconds turns the OAuth `refresh_expires_in` field
// into the absolute timestamp we persist. Keycloak returns `0` for
// offline tokens (the canonical "no expiry" indicator), so we map that
// to the zero `time.Time` value and let the IsRefreshAlive predicate
// treat it as "lives forever". Naively doing `time.Now().Add(0)` would
// stamp the token as expired the instant after login, which is exactly
// the regression that bit grounds-cli — see IsRefreshAlive for the
// matching read path.
func RefreshExpiryFromSeconds(seconds int) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

// IsRefreshAlive reports whether the stored refresh token is still
// usable. A zero RefreshExpiresAt means the token is an OIDC offline
// token (no expiry) and is always alive; otherwise compare to wall
// clock.
func (c *Credentials) IsRefreshAlive() bool {
	if c.RefreshExpiresAt.IsZero() {
		return true
	}
	return time.Now().Before(c.RefreshExpiresAt)
}
