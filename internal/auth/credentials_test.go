package auth

import (
	"os"
	"path/filepath"
	"runtime"
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
