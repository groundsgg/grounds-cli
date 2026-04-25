package gradle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindWrapper(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(inner, 0700)
	name := "gradlew"
	if runtime.GOOS == "windows" {
		name = "gradlew.bat"
	}
	os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh"), 0700)
	got, err := FindWrapper(inner)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if want := filepath.Join(dir, name); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindWrapper_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindWrapper(dir)
	if err == nil {
		t.Errorf("expected error")
	}
}
