package browser

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
)

type commandSpec struct {
	name string
	args []string
}

var startProcess = defaultStartProcess

// OpenURL opens url in the user's browser. On WSL, avoid PowerShell-based
// launchers because their console encoding setup can write spurious errors.
func OpenURL(url string) error {
	spec, err := commandForURL(runtime.GOOS, os.Getenv, exec.LookPath, url)
	if err != nil {
		return err
	}

	return launchCommand(spec)
}

func launchCommand(spec commandSpec) error {
	wait, err := startProcess(spec)
	if err != nil {
		return err
	}
	go func() {
		_ = wait()
	}()
	return nil
}

func defaultStartProcess(spec commandSpec) (func() error, error) {
	cmd := exec.CommandContext(context.Background(), spec.name, spec.args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd.Wait, nil
}

func commandForURL(goos string, getenv func(string) string, lookPath func(string) (string, error), url string) (commandSpec, error) {
	switch goos {
	case "darwin":
		return commandSpec{name: "open", args: []string{url}}, nil
	case "windows":
		return commandSpec{name: "rundll32", args: []string{"url.dll,FileProtocolHandler", url}}, nil
	case "linux":
		if isWSL(getenv) {
			if path, err := lookPath("rundll32.exe"); err == nil {
				return commandSpec{name: path, args: []string{"url.dll,FileProtocolHandler", url}}, nil
			}
			if path, err := lookPath("cmd.exe"); err == nil {
				return commandSpec{name: path, args: []string{"/C", "start", "", quoteCmdArg(url)}}, nil
			}
		}

		for _, name := range []string{"xdg-open", "sensible-browser"} {
			if path, err := lookPath(name); err == nil {
				return commandSpec{name: path, args: []string{url}}, nil
			}
		}
	}

	return commandSpec{}, errors.New("no browser opener found")
}

func isWSL(getenv func(string) string) bool {
	return getenv("WSL_DISTRO_NAME") != "" || getenv("WSL_INTEROP") != ""
}

func quoteCmdArg(arg string) string {
	quoted := `"`
	for _, r := range arg {
		switch r {
		case '&', '|', '(', ')', '<', '>', '^':
			quoted += "^"
		case '"':
			quoted += `\`
		}
		quoted += string(r)
	}
	return quoted + `"`
}
