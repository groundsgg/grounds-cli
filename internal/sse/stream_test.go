package sse

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStream_Subscribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher := w.(http.Flusher)
		fmt.Fprintln(w, "event: status")
		fmt.Fprintln(w, `data: {"status":"deploying"}`)
		fmt.Fprintln(w)
		flusher.Flush()
		fmt.Fprintln(w, "event: status")
		fmt.Fprintln(w, `data: {"status":"ready","publicUrl":"https://x.test"}`)
		fmt.Fprintln(w)
		flusher.Flush()
	}))
	defer srv.Close()

	s := &Stream{URL: srv.URL, Client: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var statuses []string
	err := s.Subscribe(ctx, func(ev *Event) error {
		statuses = append(statuses, ev.Status)
		if ev.Status == "ready" {
			cancel() // signal terminal
		}
		return nil
	})
	if err != nil && err != context.Canceled {
		t.Fatalf("err: %v", err)
	}
	if len(statuses) < 2 || statuses[0] != "deploying" || statuses[1] != "ready" {
		t.Errorf("statuses = %v", statuses)
	}
}
