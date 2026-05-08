package auth

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "grounds"
	keyringKey     = "credentials"
	fileName       = "credentials.json"
)

type Store interface {
	Save(*Credentials) error
	Load() (*Credentials, error)
	Delete() error
}

// store writes credentials to the OS keyring (when available) AND to a
// 0600-perm JSON file in the config dir. Both copies are kept in sync
// on every save:
//
//   - The keyring is the CLI's primary read source — fast, OS-managed.
//   - The JSON file is the grounds-push Gradle plugin's *only* read
//     source (its CredentialResolver doesn't link go-keyring).
//
// The earlier behaviour (keyring won → file deleted) created a silent
// gap on macOS where the CLI looked authed but `groundsPush` reported
// "no credentials" — see grounds-cli #<TODO>. Dual-writing closes that.
//
// Read prefers the keyring (keeps it the source of truth on platforms
// that have one). If the keyring path is unavailable — Linux without a
// secret service, or a CI box — both Save and Load fall through to the
// file.
type store struct {
	configDir string
}

func NewStore(configDir string) Store { return &store{configDir: configDir} }

func (s *store) Save(c *Credentials) error {
	blob, err := c.Marshal()
	if err != nil {
		return err
	}
	keyringErr := keyring.Set(keyringService, keyringKey, string(blob))
	fileErr := s.saveFile(blob)
	// Success when at least one copy landed. We don't propagate keyring
	// failures when the file write succeeds — Linux CI hosts routinely
	// have no secret service and that's an expected fallback, not a
	// user-facing error.
	if keyringErr != nil && fileErr != nil {
		return fmt.Errorf("save credentials: keyring=%v file=%v", keyringErr, fileErr)
	}
	return nil
}

func (s *store) Load() (*Credentials, error) {
	if blob, err := keyring.Get(keyringService, keyringKey); err == nil {
		return ParseCredentials([]byte(blob))
	}
	return s.loadFile()
}

func (s *store) Delete() error {
	_ = keyring.Delete(keyringService, keyringKey)
	return s.cleanupFile()
}

func (s *store) saveFile(blob []byte) error {
	if err := os.MkdirAll(s.configDir, 0700); err != nil {
		return err
	}
	path := filepath.Join(s.configDir, fileName)
	return os.WriteFile(path, blob, 0600)
}

func (s *store) loadFile() (*Credentials, error) {
	path := filepath.Join(s.configDir, fileName)
	blob, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("not authenticated: run 'grounds login'")
		}
		return nil, err
	}
	return ParseCredentials(blob)
}

func (s *store) cleanupFile() error {
	path := filepath.Join(s.configDir, fileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
