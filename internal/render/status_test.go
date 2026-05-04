package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/groundsgg/grounds-cli/internal/api"
)

func TestStatus_Active(t *testing.T) {
	SetEnabled(true) // disable colour for stable assertions
	buf := &bytes.Buffer{}
	in := time.Now().Add(2 * time.Hour)
	Status(buf, &api.ClusterStatus{
		Namespace:        "user-x",
		State:            "active",
		Profile:          "minigame",
		AutoPauseAt:      &in,
		Quota:            map[string]string{"cpu": "4", "memory": "8Gi", "storage": "20Gi"},
		DeploymentsReady: 1,
	})
	out := buf.String()
	for _, want := range []string{"user-x", "active", "auto-pause at", "1 ready", "4 CPU"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestStatus_PausedShowsWarning(t *testing.T) {
	SetEnabled(true)
	buf := &bytes.Buffer{}
	in := time.Now().Add(48 * time.Hour)
	Status(buf, &api.ClusterStatus{
		Namespace:    "user-x",
		State:        "paused",
		Profile:      "minigame",
		AutoDeleteAt: &in,
	})
	out := buf.String()
	if !strings.Contains(out, "auto-delete at") {
		t.Errorf("no auto-delete row\n%s", out)
	}
	if !strings.Contains(out, "Workspace - Paused") {
		t.Errorf("no warning line\n%s", out)
	}
	if !strings.Contains(out, "Next push or `grounds cluster up` resumes it.") {
		t.Errorf("no warning detail\n%s", out)
	}
}
