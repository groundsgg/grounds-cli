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

// store first tries the OS keyring; if unavailable, falls back to a
// 0600-perm JSON file in the config dir.
type store struct {
	configDir string
}

func NewStore(configDir string) Store { return &store{configDir: configDir} }

func (s *store) Save(c *Credentials) error {
	blob, err := c.Marshal()
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, keyringKey, string(blob)); err == nil {
		return s.cleanupFile() // keyring won — drop any stale file
	}
	return s.saveFile(blob)
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
