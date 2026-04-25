package auth

import (
	"context"
	"fmt"
	"time"
)

// FileTokenSource adapts a Store + DeviceClient into the
// api.TokenSource interface used by the HTTP client.
type FileTokenSource struct {
	Store  Store
	Device *DeviceClient
}

func (f *FileTokenSource) Token(ctx context.Context) (string, error) {
	c, err := f.Store.Load()
	if err != nil {
		return "", err
	}
	if time.Now().After(c.ExpiresAt.Add(-30 * time.Second)) {
		fresh, err := f.Device.Refresh(ctx, c.RefreshToken)
		if err != nil {
			_ = f.Store.Delete()
			return "", fmt.Errorf("session expired: run 'grounds login' again")
		}
		c.AccessToken = fresh.AccessToken
		c.RefreshToken = fresh.RefreshToken
		c.ExpiresAt = time.Now().Add(time.Duration(fresh.ExpiresIn) * time.Second)
		c.RefreshExpiresAt = time.Now().Add(time.Duration(fresh.RefreshExpiresIn) * time.Second)
		if err := f.Store.Save(c); err != nil {
			return "", err
		}
	}
	return c.AccessToken, nil
}
