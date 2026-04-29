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
