package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/r3labs/sse/v2"
	"gopkg.in/cenkalti/backoff.v1"
)

type Event struct {
	Type    string         `json:"-"`
	Status  string         `json:"status,omitempty"`
	Message string         `json:"message,omitempty"`
	Extra   map[string]any `json:"-"`
}

type Stream struct {
	URL    string
	Token  string
	Client *http.Client
}

// Subscribe streams events until the server closes (or ctx is cancelled).
// fn is called once per event; returning a non-nil error stops the
// stream cleanly. The subscriber reconnects with bounded exponential
// backoff (1s → 30s cap) on transport errors.
func (s *Stream) Subscribe(ctx context.Context, fn func(*Event) error) error {
	client := sse.NewClient(s.URL)
	client.Connection = s.Client
	client.Headers["Authorization"] = "Bearer " + s.Token
	client.ReconnectStrategy = &boundedBackoff{base: time.Second, cap: 30 * time.Second}

	errCh := make(chan error, 1)
	go func() {
		err := client.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
			ev := decode(msg)
			if err := fn(ev); err != nil {
				errCh <- err
			}
		})
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err == nil || err == context.Canceled {
			return nil
		}
		return err
	}
}

func decode(msg *sse.Event) *Event {
	ev := &Event{Type: string(msg.Event)}
	if len(msg.Data) == 0 {
		return ev
	}
	var raw map[string]any
	if err := json.Unmarshal(msg.Data, &raw); err == nil {
		ev.Extra = raw
		if s, ok := raw["status"].(string); ok {
			ev.Status = s
		}
		if m, ok := raw["message"].(string); ok {
			ev.Message = m
		}
	} else {
		ev.Message = string(msg.Data)
	}
	return ev
}

type boundedBackoff struct {
	base    time.Duration
	cap     time.Duration
	attempt int
}

func (b *boundedBackoff) NextBackOff() time.Duration {
	d := b.base
	for i := 0; i < b.attempt; i++ {
		d *= 2
		if d >= b.cap {
			d = b.cap
			break
		}
	}
	b.attempt++
	return d
}

func (b *boundedBackoff) Reset() {
	b.attempt = 0
}

// Errf is a tiny helper so callers can emit a typed terminal "stream
// failed" without rebuilding strings inline.
func Errf(format string, args ...any) error { return fmt.Errorf(format, args...) }

// Ensure boundedBackoff implements backoff.BackOff interface.
var _ backoff.BackOff = (*boundedBackoff)(nil)
