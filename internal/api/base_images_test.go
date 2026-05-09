package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListBaseImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/base-images" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(BaseImageCatalog{
			Items: []BaseImageSource{{
				Key:          "paper",
				DisplayName:  "Paper",
				ManifestType: "plugin-paper",
				Image:        "ghcr.io/groundsgg/paper",
				Versions:     []BaseImageVersion{{Version: "0.8.2", Selectable: true}},
			}},
		})
	}))
	defer server.Close()

	client := New(server.URL, nil)
	got, err := client.ListBaseImages(context.Background())
	if err != nil {
		t.Fatalf("ListBaseImages: %v", err)
	}
	if got.Items[0].Key != "paper" || got.Items[0].ManifestType != "plugin-paper" {
		t.Fatalf("catalog = %#v", got)
	}
}
