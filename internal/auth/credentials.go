package auth

import (
	"encoding/json"
	"fmt"
	"time"
)

// Credentials matches the schema used by the Gradle plugin (Phase 2.2 §5).
// CLI is the canonical writer; the Gradle plugin reads it.
type Credentials struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	Email            string    `json:"email,omitempty"`
	PreferredUser    string    `json:"preferredUsername,omitempty"`
}

func (c *Credentials) Marshal() ([]byte, error) { return json.MarshalIndent(c, "", "  ") }

func ParseCredentials(b []byte) (*Credentials, error) {
	c := &Credentials{}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return c, nil
}
