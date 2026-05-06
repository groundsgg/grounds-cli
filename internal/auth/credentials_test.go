package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRoundtrip(t *testing.T) {
	// Force file path: fake keyring backend (zalando/go-keyring's
	// MockInit() returns a backed map, but here we just bypass and
	// exercise the file path by filling the configDir.)
	dir := t.TempDir()
	s := &store{configDir: dir}
	src := &Credentials{
		AccessToken: "at", RefreshToken: "rt",
		ExpiresAt: time.Now().Add(5*time.Minute).Truncate(time.Second),
		Email:     "x@example.com",
	}

	// Direct file save (bypass keyring) — proves the fallback works.
	if err := s.saveFile([]byte(`{"accessToken":"at","refreshToken":"rt","email":"x@example.com","expiresAt":"` + src.ExpiresAt.Format(time.RFC3339) + `"}`)); err != nil {
		t.Fatalf("saveFile: %v", err)
	}
	loaded, err := s.loadFile()
	if err != nil {
		t.Fatalf("loadFile: %v", err)
	}
	if loaded.AccessToken != "at" || loaded.RefreshToken != "rt" || loaded.Email != "x@example.com" {
		t.Errorf("loaded mismatch: %+v", loaded)
	}
	// Mode check (Unix permission bits — Windows reports 0666 regardless of chmod)
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(filepath.Join(dir, fileName))
		if info.Mode().Perm() != 0600 {
			t.Errorf("perm = %v", info.Mode().Perm())
		}
	}
}

func TestLoadMissing(t *testing.T) {
	s := &store{configDir: t.TempDir()}
	_, err := s.loadFile()
	if err == nil {
		t.Errorf("expected error on missing file")
	}
}

// TestMarshalAlwaysWritesVersion verifies the cross-repo schema contract
// with grounds-push's CredentialResolver (which requires version: 1).
func TestMarshalAlwaysWritesVersion(t *testing.T) {
	c := &Credentials{AccessToken: "at"}
	blob, err := c.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(blob), `"version": 1`) {
		t.Errorf("missing version field in output: %s", blob)
	}

	var roundtrip Credentials
	if err := json.Unmarshal(blob, &roundtrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundtrip.Version != CredentialsVersion {
		t.Errorf("version = %d, want %d", roundtrip.Version, CredentialsVersion)
	}
}

// TestParseLegacyFileWithoutVersion verifies that pre-version files
// written by an older CLI parse cleanly and get upgraded on next save.
func TestParseLegacyFileWithoutVersion(t *testing.T) {
	legacy := []byte(`{"accessToken":"at","refreshToken":"rt"}`)
	c, err := ParseCredentials(legacy)
	if err != nil {
		t.Fatalf("ParseCredentials: %v", err)
	}
	if c.Version != CredentialsVersion {
		t.Errorf("legacy file should be upgraded to v%d, got v%d", CredentialsVersion, c.Version)
	}
}

// TestRefreshExpiryFromSeconds_offline maps Keycloak's `0` for offline
// tokens to a zero `time.Time` so the CLI doesn't immediately decide
// the refresh token is dead the instant after login.
func TestRefreshExpiryFromSeconds_offline(t *testing.T) {
	if !RefreshExpiryFromSeconds(0).IsZero() {
		t.Error("seconds=0 should map to zero time.Time")
	}
	if !RefreshExpiryFromSeconds(-5).IsZero() {
		t.Error("negative seconds should also map to zero time.Time")
	}
}

func TestRefreshExpiryFromSeconds_finite(t *testing.T) {
	got := RefreshExpiryFromSeconds(60)
	if got.IsZero() {
		t.Fatal("seconds=60 should produce a non-zero expiry")
	}
	if got.Before(time.Now()) {
		t.Error("seconds=60 should produce an expiry in the future")
	}
}

func TestIsRefreshAlive(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		c    Credentials
		want bool
	}{
		{"offline (zero time)", Credentials{}, true},
		{"future", Credentials{RefreshExpiresAt: now.Add(time.Hour)}, true},
		{"past", Credentials{RefreshExpiresAt: now.Add(-time.Hour)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.IsRefreshAlive(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
