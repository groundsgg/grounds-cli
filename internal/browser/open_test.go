package browser

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCommandForURLUsesCmdExeOnWSL(t *testing.T) {
	spec, err := commandForURL("linux", env(map[string]string{
		"WSL_DISTRO_NAME": "Ubuntu",
	}), path(map[string]string{
		"cmd.exe": "/mnt/c/WINDOWS/system32/cmd.exe",
	}), "https://example.test/device")
	if err != nil {
		t.Fatalf("commandForURL returned error: %v", err)
	}

	want := commandSpec{name: "/mnt/c/WINDOWS/system32/cmd.exe", args: []string{"/C", "start", "", `"https://example.test/device"`}}
	if !reflect.DeepEqual(spec, want) {
		t.Fatalf("commandForURL() = %#v, want %#v", spec, want)
	}
}

func TestCommandForURLQuotesCmdExeURLOnWSLWhenURLContainsCmdMetacharacters(t *testing.T) {
	spec, err := commandForURL("linux", env(map[string]string{
		"WSL_DISTRO_NAME": "Ubuntu",
	}), path(map[string]string{
		"cmd.exe": "/mnt/c/WINDOWS/system32/cmd.exe",
	}), "https://example.test/device?foo=1&bar=(2)|baz=a%26b")
	if err != nil {
		t.Fatalf("commandForURL returned error: %v", err)
	}

	want := commandSpec{name: "/mnt/c/WINDOWS/system32/cmd.exe", args: []string{"/C", "start", "", `"https://example.test/device?foo=1^&bar=^(2^)^|baz=a%26b"`}}
	if !reflect.DeepEqual(spec, want) {
		t.Fatalf("commandForURL() = %#v, want %#v", spec, want)
	}
}

func TestCommandForURLUsesRundll32OnWSL(t *testing.T) {
	spec, err := commandForURL("linux", env(map[string]string{
		"WSL_INTEROP": "/run/WSL/1_interop",
	}), path(map[string]string{
		"rundll32.exe": "/mnt/c/WINDOWS/system32/rundll32.exe",
		"cmd.exe":      "/mnt/c/WINDOWS/system32/cmd.exe",
	}), `https://example.test/device?code=A&B=(C)|D`)
	if err != nil {
		t.Fatalf("commandForURL returned error: %v", err)
	}

	want := commandSpec{name: "/mnt/c/WINDOWS/system32/rundll32.exe", args: []string{"url.dll,FileProtocolHandler", `https://example.test/device?code=A&B=(C)|D`}}
	if !reflect.DeepEqual(spec, want) {
		t.Fatalf("commandForURL() = %#v, want %#v", spec, want)
	}
}

func TestCommandForURLUsesXDGOpenOnLinux(t *testing.T) {
	spec, err := commandForURL("linux", env(nil), path(map[string]string{
		"xdg-open": "/usr/bin/xdg-open",
	}), "https://example.test/device")
	if err != nil {
		t.Fatalf("commandForURL returned error: %v", err)
	}

	want := commandSpec{name: "/usr/bin/xdg-open", args: []string{"https://example.test/device"}}
	if !reflect.DeepEqual(spec, want) {
		t.Fatalf("commandForURL() = %#v, want %#v", spec, want)
	}
}

func TestCommandForURLReturnsErrorWhenNoLinuxOpenerExists(t *testing.T) {
	_, err := commandForURL("linux", env(nil), path(nil), "https://example.test/device")
	if err == nil {
		t.Fatal("commandForURL returned nil error")
	}
}

func TestLaunchCommandWaitsForStartedProcess(t *testing.T) {
	waited := make(chan struct{})
	started := false
	startProcess = func(commandSpec) (func() error, error) {
		started = true
		return func() error {
			close(waited)
			return nil
		}, nil
	}
	t.Cleanup(func() {
		startProcess = defaultStartProcess
	})

	if err := launchCommand(commandSpec{name: "opener", args: []string{"https://example.test/device"}}); err != nil {
		t.Fatalf("launchCommand returned error: %v", err)
	}
	if !started {
		t.Fatal("launchCommand did not start process")
	}

	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("launchCommand did not wait for process")
	}
}

func env(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func path(available map[string]string) func(string) (string, error) {
	return func(name string) (string, error) {
		if path, ok := available[name]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}
